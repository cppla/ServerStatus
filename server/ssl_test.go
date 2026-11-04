package main

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func sslTestServer(t *testing.T, handshake func(*tls.ClientHelloInfo) (*tls.Config, error)) (*httptest.Server, ConfigDocument) {
	t.Helper()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
	server.TLS = &tls.Config{GetConfigForClient: handshake}
	server.StartTLS()
	t.Cleanup(server.Close)
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatal(err)
	}
	doc := minimalTestConfig()
	doc["sslcerts"] = []any{map[string]any{
		"name": "local", "domain": server.URL, "port": port, "interval": 7200,
	}}
	return server, doc
}

func sslTestState(app *App, key string) CertState {
	app.certMu.RLock()
	defer app.certMu.RUnlock()
	return *app.certs[key]
}

func waitForSSLCheck(t *testing.T, app *App, config SSLCertConfig, previous time.Time) CertState {
	t.Helper()
	eventually(t, 2*time.Second, func() bool {
		state := sslTestState(app, certKey(config))
		return !state.Checking && state.LastCheck.After(previous)
	})
	return sslTestState(app, certKey(config))
}

func TestSSLFailureAndRecovery(t *testing.T) {
	var rejectHandshake atomic.Bool
	server, doc := sslTestServer(t, func(*tls.ClientHelloInfo) (*tls.Config, error) {
		if rejectHandshake.Load() {
			return nil, errors.New("test TLS failure")
		}
		return nil, nil
	})
	app := newTestApp(t, doc)
	config := app.RuntimeSnapshot().SSLCerts[0]
	key := certKey(config)
	app.runDueSSLChecks()
	state := waitForSSLCheck(t, app, config, time.Time{})
	if state.ExpireTS != server.Certificate().NotAfter.Unix() || state.LastError != "" {
		t.Fatalf("initial certificate check: %+v", state)
	}

	for _, phase := range []string{"TLS failure", "recovery", "connection failure"} {
		t.Run(phase, func(t *testing.T) {
			switch phase {
			case "TLS failure":
				rejectHandshake.Store(true)
			case "recovery":
				rejectHandshake.Store(false)
			case "connection failure":
				server.Close()
			}
			previous := state.LastCheck
			app.certMu.Lock()
			// Simulate the next scheduled interval without waiting two hours.
			app.certs[key].LastCheck = time.Time{}
			if phase != "recovery" {
				// A failed check must also discard any prior hostname mismatch.
				app.certs[key].Mismatch = true
			}
			app.certMu.Unlock()
			app.runDueSSLChecks()
			state = waitForSSLCheck(t, app, config, previous)
			entry := app.sslSnapshot([]SSLCertConfig{config}, time.Now())[0].(map[string]any)
			if phase == "recovery" {
				if state.LastError != "" || state.ExpireTS != server.Certificate().NotAfter.Unix() || state.Mismatch {
					t.Errorf("certificate did not recover: %+v", state)
				}
				if _, hasError := entry["error"]; hasError || entry["expire_days"].(int) <= 0 {
					t.Errorf("recovered snapshot: %#v", entry)
				}
			} else if state.LastError == "" || state.ExpireTS != 0 || state.Mismatch ||
				entry["error"] == nil || entry["expire_ts"] != int64(0) || entry["expire_days"] != 0 || entry["mismatch"] != false {
				t.Errorf("failed check retained a certificate result: state=%+v snapshot=%#v", state, entry)
			}
		})
	}
}

func TestSSLConfigReloadRechecks(t *testing.T) {
	for _, change := range []string{"unchanged reload", "interval", "callback"} {
		t.Run(change, func(t *testing.T) {
			_, doc := sslTestServer(t, nil)
			app := newTestApp(t, doc)
			config := app.RuntimeSnapshot().SSLCerts[0]
			key := certKey(config)
			app.runDueSSLChecks()
			before := waitForSSLCheck(t, app, config, time.Time{})
			alarmTime := time.Now()
			app.certMu.Lock()
			app.certs[key].LastAlarm7 = alarmTime
			app.certs[key].LastAlarm3 = alarmTime
			app.certs[key].LastAlarm1 = alarmTime
			app.certs[key].LastAlarmMismatch = alarmTime
			app.certMu.Unlock()

			if change == "unchanged reload" {
				if apiErr := app.ReloadConfig(); apiErr != nil {
					t.Fatal(apiErr)
				}
			} else {
				updated := app.ConfigSnapshot()
				cert := updated["sslcerts"].([]any)[0].(map[string]any)
				if change == "interval" {
					cert["interval"] = 14400
				} else {
					cert["callback"] = "http://127.0.0.1:1/?message="
				}
				if _, apiErr := app.ReplaceConfig(updated); apiErr != nil {
					t.Fatal(apiErr)
				}
			}
			state := sslTestState(app, key)
			if !state.LastCheck.IsZero() || state.Checking || state.ExpireTS != 0 || state.Mismatch || state.LastError != "" {
				t.Errorf("reload did not reset the certificate check: %+v", state)
			}
			if state.LastAlarm7 != alarmTime || state.LastAlarm3 != alarmTime || state.LastAlarm1 != alarmTime || state.LastAlarmMismatch != alarmTime {
				t.Errorf("reload reset notification cooldowns: %+v", state)
			}
			app.runDueSSLChecks()
			state = waitForSSLCheck(t, app, config, before.LastCheck)
			if state.ExpireTS == 0 || state.LastError != "" {
				t.Errorf("reload did not publish a fresh result: %+v", state)
			}
		})
	}
}

func TestSSLDiscardInFlightResultAfterConfigChange(t *testing.T) {
	for _, change := range []string{"reload", "remove and readd"} {
		t.Run(change, func(t *testing.T) {
			oldStarted, newStarted := make(chan struct{}), make(chan struct{})
			releaseOld, releaseNew := make(chan struct{}), make(chan struct{})
			var oldOnce, newOnce sync.Once
			unblockOld := func() { oldOnce.Do(func() { close(releaseOld) }) }
			unblockNew := func() { newOnce.Do(func() { close(releaseNew) }) }
			var handshakes atomic.Int32
			_, doc := sslTestServer(t, func(*tls.ClientHelloInfo) (*tls.Config, error) {
				switch handshakes.Add(1) {
				case 1:
					close(oldStarted)
					<-releaseOld
				case 2:
					close(newStarted)
					<-releaseNew
				}
				return nil, nil
			})
			// Unblock handshakes before httptest.Server cleanup, including failures.
			t.Cleanup(unblockOld)
			t.Cleanup(unblockNew)
			app := newTestApp(t, doc)
			config := app.RuntimeSnapshot().SSLCerts[0]
			key := certKey(config)
			app.certMu.Lock()
			oldState := app.certs[key]
			oldState.Checking = true
			app.certMu.Unlock()
			oldDone := make(chan struct{})
			go func() {
				app.executeSSLCheck(key, oldState)
				close(oldDone)
			}()
			await := func(channel <-chan struct{}) {
				t.Helper()
				select {
				case <-channel:
				case <-time.After(2 * time.Second):
					t.Fatal("certificate handshake did not reach the expected stage")
				}
			}
			await(oldStarted)
			if change == "reload" {
				if apiErr := app.ReloadConfig(); apiErr != nil {
					t.Fatal(apiErr)
				}
			} else {
				withoutCert := app.ConfigSnapshot()
				withoutCert["sslcerts"] = []any{}
				if _, apiErr := app.ReplaceConfig(withoutCert); apiErr != nil {
					t.Fatal(apiErr)
				}
				if _, apiErr := app.ReplaceConfig(doc); apiErr != nil {
					t.Fatal(apiErr)
				}
			}
			app.runDueSSLChecks()
			await(newStarted)
			unblockOld()
			await(oldDone)
			state := sslTestState(app, key)
			if !state.Checking || !state.LastCheck.IsZero() || state.ExpireTS != 0 || state.Mismatch || state.LastError != "" {
				t.Errorf("old check changed the new in-flight state: %+v", state)
			}
			unblockNew()
			state = waitForSSLCheck(t, app, config, time.Time{})
			if state.ExpireTS == 0 || state.LastError != "" {
				t.Errorf("new check did not publish its result: %+v", state)
			}
		})
	}
}

func TestCertificateCheckAndSnapshot(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(parsed.Port())
	config := SSLCertConfig{Name: "local", Domain: server.URL, Port: port, Interval: 1}
	expireTS, _, err := checkCertificate(config)
	if err != nil {
		t.Fatal(err)
	}
	if expireTS <= time.Now().Unix() {
		t.Fatalf("unexpected expiration %d", expireTS)
	}

	doc := minimalTestConfig()
	doc["sslcerts"] = []any{map[string]any{"name": config.Name, "domain": config.Domain, "port": config.Port, "interval": 1}}
	app := newTestApp(t, doc)
	app.runDueSSLChecks()
	eventually(t, 2*time.Second, func() bool {
		app.certMu.RLock()
		defer app.certMu.RUnlock()
		state := app.certs[certKey(config)]
		return state != nil && !state.LastCheck.IsZero() && state.ExpireTS > 0
	})
	certs := app.SnapshotStats()["sslcerts"].([]any)
	if certs[0].(map[string]any)["expire_ts"].(int64) <= time.Now().Unix() {
		t.Fatalf("certificate snapshot: %#v", certs[0])
	}
}

func TestCallbackDelivery(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.Method != http.MethodPost || request.URL.Query().Get("message") == "" {
			t.Errorf("unexpected callback request: %s %s", request.Method, request.URL.String())
		}
		w.WriteHeader(204)
	}))
	defer server.Close()
	doc := minimalTestConfig()
	doc["watchdog"] = []any{map[string]any{
		"name": "cpu", "rule": "cpu>90", "interval": 60, "callback": server.URL + "?message=",
	}}
	app := newTestApp(t, doc)
	app.nodeMu.Lock()
	node := app.nodes["s01"]
	node.Connected = true
	node.Stats = AgentStats{CPU: 99}
	node.HasUpdate = true
	app.nodeMu.Unlock()
	app.evaluateWatchdogs("s01", false)
	eventually(t, time.Second, func() bool { return calls.Load() == 1 })
	app.evaluateWatchdogs("s01", false)
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("watchdog cooldown failed: calls=%d", calls.Load())
	}
}
