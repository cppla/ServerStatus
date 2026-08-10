package main

import (
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"
)

func readPersistedServer(t *testing.T, app *App) map[string]any {
	t.Helper()
	data, err := os.ReadFile(app.opts.StatsPath)
	if err != nil {
		t.Fatal(err)
	}
	var stats map[string]any
	if err := json.Unmarshal(data, &stats); err != nil {
		t.Fatal(err)
	}
	servers, ok := stats["servers"].([]any)
	if !ok || len(servers) != 1 {
		t.Fatalf("unexpected persisted servers: %#v", stats["servers"])
	}
	server, ok := servers[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected persisted server: %#v", servers[0])
	}
	return server
}

func TestTrafficBaselinesResetIndependently(t *testing.T) {
	node := &NodeState{LastNetworkIn: 100, LastNetworkOut: 0}
	updateTrafficBaselines(node, 150, 500, false)
	if node.LastNetworkIn != 100 || node.LastNetworkOut != 500 {
		t.Fatalf("missing outbound baseline was not initialized independently: %#v", node)
	}

	node.LastNetworkOut = 700
	updateTrafficBaselines(node, 200, 50, false)
	if node.LastNetworkIn != 100 || node.LastNetworkOut != 50 {
		t.Fatalf("outbound counter reset changed the wrong baseline: %#v", node)
	}

	updateTrafficBaselines(node, 900, 800, true)
	if node.LastNetworkIn != 900 || node.LastNetworkOut != 800 {
		t.Fatalf("monthly reset did not reset both baselines: %#v", node)
	}
}

func TestDisconnectPreservesOfflineDisplayMetadata(t *testing.T) {
	app := newTestApp(t, minimalTestConfig())
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	app.nodeMu.Lock()
	node := app.nodes["s01"]
	node.Connected = true
	node.Connection = server
	node.ConnectionID = 42
	node.HasUpdate = true
	node.Stats = AgentStats{OS: "linux", CPUModel: "Test CPU"}
	app.nodeMu.Unlock()

	app.disconnectAgent("s01", server, 42)
	serverStats := app.SnapshotStats()["servers"].([]any)[0].(map[string]any)
	if serverStats["online4"] != false || serverStats["online6"] != false {
		t.Fatalf("disconnected node remained online: %#v", serverStats)
	}
	if serverStats["os"] != "linux" || serverStats["cpu_model"] != "Test CPU" {
		t.Fatalf("offline display metadata was discarded: %#v", serverStats)
	}
}

func TestStatsPersistenceUsesLiveMemoryBetweenFlushes(t *testing.T) {
	if statsFlushInterval != time.Minute {
		t.Fatalf("unexpected stats flush interval: %s", statsFlushInterval)
	}

	app := newTestApp(t, minimalTestConfig())
	if err := app.PersistStats(); err != nil {
		t.Fatal(err)
	}
	initialInfo, err := os.Stat(app.opts.StatsPath)
	if err != nil {
		t.Fatal(err)
	}

	app.nodeMu.Lock()
	node := app.nodes["s01"]
	node.Connected = true
	node.ConnectionID = 7
	app.nodeMu.Unlock()
	if !app.updateAgent("s01", 7, AgentStats{CPU: 42, NetworkIn: 1000, NetworkOut: 2000}) {
		t.Fatal("agent update was rejected")
	}

	live := app.SnapshotStats()["servers"].([]any)[0].(map[string]any)
	if live["cpu"] != 42 {
		t.Fatalf("live snapshot was not updated: %#v", live)
	}
	beforeFlushInfo, err := os.Stat(app.opts.StatsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(initialInfo, beforeFlushInfo) {
		t.Fatal("reading the live snapshot unexpectedly rewrote stats.json")
	}
	if _, exists := readPersistedServer(t, app)["cpu"]; exists {
		t.Fatal("agent update reached disk before the next persistence run")
	}

	if err := app.PersistStats(); err != nil {
		t.Fatal(err)
	}
	afterFlushInfo, err := os.Stat(app.opts.StatsPath)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(beforeFlushInfo, afterFlushInfo) {
		t.Fatal("updated stats were not persisted")
	}
	if cpu := readPersistedServer(t, app)["cpu"]; cpu != float64(42) {
		t.Fatalf("unexpected persisted CPU value: %#v", cpu)
	}

	if err := app.PersistStats(); err != nil {
		t.Fatal(err)
	}
	cleanInfo, err := os.Stat(app.opts.StatsPath)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(afterFlushInfo, cleanInfo) {
		t.Fatal("fixed-interval persistence unexpectedly skipped a write")
	}
}

func TestReloadNoticeIsIndependentFromPersistence(t *testing.T) {
	app := newTestApp(t, minimalTestConfig())
	app.reloadPending.Store(true)
	if err := app.PersistStats(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(app.opts.StatsPath)
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]any
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if _, exists := persisted["reload"]; exists {
		t.Fatal("reload notice must not be persisted to stats.json")
	}
	if reloaded, _ := app.SnapshotStats()["reload"].(bool); !reloaded {
		t.Fatal("pending reload notice was not returned by the live endpoint")
	}
	if _, exists := app.SnapshotStats()["reload"]; exists {
		t.Fatal("reload notice was returned more than once")
	}
}

func TestResetTrafficPersistsImmediately(t *testing.T) {
	app := newTestApp(t, minimalTestConfig())
	app.nodeMu.Lock()
	node := app.nodes["s01"]
	node.Connected = true
	node.HasUpdate = true
	node.Stats = AgentStats{NetworkIn: 1234, NetworkOut: 5678}
	app.nodeMu.Unlock()

	if _, apiErr := app.ResetTraffic("s01"); apiErr != nil {
		t.Fatal(apiErr)
	}
	persisted := readPersistedServer(t, app)
	if persisted["last_network_in"] != float64(1234) || persisted["last_network_out"] != float64(5678) {
		t.Fatalf("traffic reset was not persisted immediately: %#v", persisted)
	}
}
