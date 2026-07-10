package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

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
