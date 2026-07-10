package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func minimalTestConfig() ConfigDocument {
	return ConfigDocument{
		"servers": []any{
			map[string]any{
				"username": "s01", "name": "node1", "type": "kvm", "host": "host1",
				"location": "CN", "password": "secret", "monthstart": 1,
			},
		},
		"monitors": []any{
			map[string]any{"name": "example", "host": "https://example.com", "interval": 60, "type": "https"},
		},
		"sslcerts": []any{},
		"watchdog": []any{},
	}
}

func newTestApp(t *testing.T, doc ConfigDocument) *App {
	t.Helper()
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	statsPath := filepath.Join(directory, "data", "stats.json")
	webDir := filepath.Join(directory, "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<!doctype html><title>test-ui</title>"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp(Options{
		ConfigPath: configPath,
		StatsPath:  statsPath,
		WebDir:     webDir,
		HTTPAddr:   "127.0.0.1:0",
		AgentAddr:  "127.0.0.1:0",
		AdminToken: "test-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	return app
}

func performRequest(handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func eventually(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
