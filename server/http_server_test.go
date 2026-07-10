package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestHTTPAPIAndStaticUI(t *testing.T) {
	app := newTestApp(t, minimalTestConfig())
	router := app.router()

	response := performRequest(router, http.MethodGet, "/", "", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "test-ui") {
		t.Fatalf("static UI: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodGet, "/api/health", "", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"enabled":true`) {
		t.Fatalf("health: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodGet, "/api/openapi.json", "", "")
	if response.Code != 200 {
		t.Fatalf("openapi: status=%d body=%s", response.Code, response.Body.String())
	}
	var openapi map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &openapi); err != nil {
		t.Fatal(err)
	}
	paths := openapi["paths"].(map[string]any)
	if openapi["openapi"] != "3.1.0" || paths["/api/servers/{username}"] == nil || paths["/api/watchdog/{id}"] == nil {
		t.Fatalf("OpenAPI document is incomplete: %#v", openapi)
	}
	monitorOperations := paths["/api/monitors"].(map[string]any)
	createResponses := monitorOperations["post"].(map[string]any)["responses"].(map[string]any)
	if createResponses["201"] == nil || createResponses["200"] != nil {
		t.Fatalf("OpenAPI create response must describe HTTP 201: %#v", createResponses)
	}
	listResponses := monitorOperations["get"].(map[string]any)["responses"].(map[string]any)
	listSchema := listResponses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if listSchema["properties"].(map[string]any)["monitors"] == nil {
		t.Fatalf("OpenAPI list response does not expose monitors: %#v", listSchema)
	}
	restartResponses := paths["/api/restart"].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)
	if restartResponses["202"] == nil {
		t.Fatalf("OpenAPI restart response must describe HTTP 202: %#v", restartResponses)
	}
	response = performRequest(router, http.MethodGet, "/api/config", "", "wrong")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	response = performRequest(router, http.MethodGet, "/api/config", "", "test-token")
	if response.Code != 200 {
		t.Fatalf("config: status=%d body=%s", response.Code, response.Body.String())
	}

	monitor := `{"name":"新增服务","host":"tcp://127.0.0.1:80","type":"tcp","interval":"30"}`
	response = performRequest(router, http.MethodPost, "/api/monitors", monitor, "test-token")
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), "新增服务") {
		t.Fatalf("create monitor: status=%d body=%s", response.Code, response.Body.String())
	}
	path := "/api/monitors/" + url.PathEscape("新增服务")
	monitor = `{"name":"更新服务","host":"https://example.org","type":"https","interval":45}`
	response = performRequest(router, http.MethodPut, path, monitor, "test-token")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "更新服务") {
		t.Fatalf("update monitor: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodDelete, "/api/monitors/"+url.PathEscape("更新服务"), "", "test-token")
	if response.Code != 200 {
		t.Fatalf("delete monitor: status=%d body=%s", response.Code, response.Body.String())
	}
	percentMonitor := `{"name":"rate%check","host":"https://example.com","type":"https","interval":30}`
	response = performRequest(router, http.MethodPost, "/api/monitors", percentMonitor, "test-token")
	if response.Code != http.StatusCreated {
		t.Fatalf("create percent monitor: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodDelete, "/api/monitors/"+url.PathEscape("rate%check"), "", "test-token")
	if response.Code != 200 {
		t.Fatalf("delete percent monitor: status=%d body=%s", response.Code, response.Body.String())
	}

	server := `{"username":"s02","name":"node2","type":"kvm","host":"host2","location":"JP","password":"secret","monthstart":31}`
	response = performRequest(router, http.MethodPost, "/api/servers", server, "test-token")
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"monthstart":28`) {
		t.Fatalf("create server: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodDelete, "/api/servers/s02", "", "test-token")
	if response.Code != 200 {
		t.Fatalf("delete server: status=%d body=%s", response.Code, response.Body.String())
	}
	numericServer := `{"username":"0","name":"numeric","type":"kvm","host":"host0","location":"US","password":"secret","monthstart":1}`
	response = performRequest(router, http.MethodPost, "/api/servers", numericServer, "test-token")
	if response.Code != http.StatusCreated {
		t.Fatalf("create numeric server: status=%d body=%s", response.Code, response.Body.String())
	}
	numericServer = `{"username":"0","name":"numeric-updated","type":"kvm","host":"host0","location":"US","password":"secret","monthstart":1}`
	response = performRequest(router, http.MethodPut, "/api/servers/0", numericServer, "test-token")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "numeric-updated") {
		t.Fatalf("update numeric server: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodDelete, "/api/servers/0", "", "test-token")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "numeric-updated") {
		t.Fatalf("delete numeric server: status=%d body=%s", response.Code, response.Body.String())
	}
	if app.RuntimeSnapshot().Servers[0].Username != "s01" {
		t.Fatal("numeric username operation modified the server at numeric index")
	}

	response = performRequest(router, http.MethodPost, "/api/servers/s01/reset-traffic", "", "test-token")
	if response.Code != http.StatusConflict {
		t.Fatalf("offline reset should conflict: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodPost, "/api/reload", "", "test-token")
	if response.Code != 200 {
		t.Fatalf("reload: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodPost, "/api/restart", "", "test-token")
	if response.Code != http.StatusAccepted || !strings.Contains(response.Body.String(), "in-process") {
		t.Fatalf("restart: status=%d body=%s", response.Code, response.Body.String())
	}

	data, err := os.ReadFile(app.opts.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]any
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("persisted config is invalid: %v", err)
	}
}

func TestHTTPRejectsInvalidAndOversizedBodies(t *testing.T) {
	app := newTestApp(t, minimalTestConfig())
	router := app.router()
	response := performRequest(router, http.MethodPost, "/api/servers", `{"name":`, "test-token")
	if response.Code != 400 {
		t.Fatalf("invalid JSON: status=%d body=%s", response.Code, response.Body.String())
	}
	response = performRequest(router, http.MethodPost, "/api/servers", `{"name":"x"}`, "test-token")
	if response.Code != 400 {
		t.Fatalf("missing fields: status=%d body=%s", response.Code, response.Body.String())
	}
	oversized := `{"name":"` + strings.Repeat("x", maxRequestBody) + `"}`
	response = performRequest(router, http.MethodPost, "/api/servers", oversized, "test-token")
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body: status=%d body=%s", response.Code, response.Body.String())
	}

	doc := app.ConfigSnapshot()
	doc["watchdog"] = []any{map[string]any{"name": "broken", "rule": "cpu >", "interval": 10}}
	data, _ := json.Marshal(doc)
	response = performRequest(router, http.MethodPut, "/api/config", string(data), "test-token")
	if response.Code != 400 {
		t.Fatalf("invalid watchdog: status=%d body=%s", response.Code, response.Body.String())
	}
}
