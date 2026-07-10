package main

import (
	"net"
	"testing"
)

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
