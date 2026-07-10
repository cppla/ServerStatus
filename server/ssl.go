package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type CertState struct {
	Config            SSLCertConfig
	ExpireTS          int64
	Mismatch          bool
	LastError         string
	LastCheck         time.Time
	Checking          bool
	LastAlarm7        time.Time
	LastAlarm3        time.Time
	LastAlarm1        time.Time
	LastAlarmMismatch time.Time
}

func certKey(config SSLCertConfig) string {
	return fmt.Sprintf("%s\x00%s\x00%d", config.Name, config.Domain, config.Port)
}

func (a *App) reconcileCerts(configs []SSLCertConfig) {
	a.certMu.Lock()
	defer a.certMu.Unlock()
	next := make(map[string]*CertState, len(configs))
	for _, config := range configs {
		key := certKey(config)
		if existing := a.certs[key]; existing != nil {
			existing.Config = config
			next[key] = existing
		} else {
			next[key] = &CertState{Config: config}
		}
	}
	a.certs = next
}

func (a *App) sslLoop() {
	a.runDueSSLChecks()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.runDueSSLChecks()
		}
	}
}

func (a *App) runDueSSLChecks() {
	now := time.Now()
	type dueCheck struct {
		key    string
		config SSLCertConfig
	}
	due := make([]dueCheck, 0)
	a.certMu.Lock()
	for key, state := range a.certs {
		interval := secondsDuration(state.Config.Interval)
		if !state.Checking && (state.LastCheck.IsZero() || now.Sub(state.LastCheck) >= interval) {
			state.Checking = true
			due = append(due, dueCheck{key: key, config: state.Config})
		}
	}
	a.certMu.Unlock()
	for _, check := range due {
		check := check
		go a.executeSSLCheck(check.key, check.config)
	}
}

func (a *App) executeSSLCheck(key string, config SSLCertConfig) {
	expireTS, mismatch, err := checkCertificate(config)
	now := time.Now()
	type notification struct {
		message string
	}
	alerts := make([]notification, 0, 2)

	a.certMu.Lock()
	state := a.certs[key]
	if state == nil {
		a.certMu.Unlock()
		return
	}
	state.Checking = false
	state.LastCheck = now
	if err != nil {
		state.LastError = err.Error()
		a.certMu.Unlock()
		a.wakeStatsWriter()
		return
	}
	state.ExpireTS = expireTS
	state.Mismatch = mismatch
	state.LastError = ""
	if config.Callback != "" && mismatch && (state.LastAlarmMismatch.IsZero() || now.Sub(state.LastAlarmMismatch) >= 24*time.Hour) {
		state.LastAlarmMismatch = now
		alerts = append(alerts, notification{message: fmt.Sprintf("【SSL证书域名不匹配】%s(%s) 证书域名与配置不一致", config.Name, config.Domain)})
	}
	days := int((expireTS - now.Unix()) / 86400)
	if config.Callback != "" {
		var lastAlarm *time.Time
		switch {
		case days <= 7 && days > 3:
			lastAlarm = &state.LastAlarm7
		case days <= 3 && days > 1:
			lastAlarm = &state.LastAlarm3
		case days <= 1:
			lastAlarm = &state.LastAlarm1
		}
		if lastAlarm != nil && (lastAlarm.IsZero() || now.Sub(*lastAlarm) >= 20*time.Hour) {
			*lastAlarm = now
			expire := time.Unix(expireTS, 0).UTC().Format("2006-01-02 15:04:05")
			alerts = append(alerts, notification{message: fmt.Sprintf("【SSL证书提醒】%s(%s) 将在 %d 天后(%s UTC) 到期", config.Name, config.Domain, days, expire)})
		}
	}
	a.certMu.Unlock()
	a.wakeStatsWriter()

	for _, alert := range alerts {
		if err := a.sendCallback(config.Callback, alert.message, "ServerStatusSSL"); err != nil {
			a.logger.Printf("SSL certificate %q callback: %v", config.Name, err)
		}
	}
}

func checkCertificate(config SSLCertConfig) (int64, bool, error) {
	host, err := certificateHost(config.Domain)
	if err != nil {
		return 0, false, err
	}
	address := net.JoinHostPort(host, strconv.Itoa(config.Port))
	dialer := &net.Dialer{Timeout: 6 * time.Second}
	connection, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // VerifyHostname below preserves the legacy chain-independent check.
	})
	if err != nil {
		return 0, false, err
	}
	defer connection.Close()
	certificates := connection.ConnectionState().PeerCertificates
	if len(certificates) == 0 {
		return 0, false, fmt.Errorf("server returned no certificate")
	}
	leaf := certificates[0]
	mismatch := leaf.VerifyHostname(host) != nil
	return leaf.NotAfter.Unix(), mismatch, nil
}

func certificateHost(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("domain is empty")
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Hostname() != "" {
		return parsed.Hostname(), nil
	}
	withoutPath := strings.SplitN(raw, "/", 2)[0]
	if host, _, err := net.SplitHostPort(withoutPath); err == nil {
		return strings.Trim(host, "[]"), nil
	}
	host := strings.Trim(withoutPath, "[]")
	if host == "" {
		return "", fmt.Errorf("domain is invalid")
	}
	return host, nil
}

func (a *App) sslSnapshot(configs []SSLCertConfig, now time.Time) []any {
	a.certMu.RLock()
	defer a.certMu.RUnlock()
	result := make([]any, 0, len(configs))
	for _, config := range configs {
		entry := map[string]any{
			"name": config.Name, "domain": config.Domain, "port": config.Port,
			"expire_ts": int64(0), "expire_days": 0, "mismatch": false,
		}
		if state := a.certs[certKey(config)]; state != nil {
			entry["expire_ts"] = state.ExpireTS
			if state.ExpireTS != 0 {
				entry["expire_days"] = int((state.ExpireTS - now.Unix()) / 86400)
			}
			entry["mismatch"] = state.Mismatch
			if state.LastError != "" {
				entry["error"] = state.LastError
			}
		}
		result = append(result, entry)
	}
	return result
}
