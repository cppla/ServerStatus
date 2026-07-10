package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestAgentProtocolAndTrafficState(t *testing.T) {
	app := newTestApp(t, minimalTestConfig())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- NewAgentServer(app).Serve(listener) }()

	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	reader := bufio.NewReader(connection)
	readContains(t, reader, "Authentication required")
	if _, err := fmt.Fprintln(connection, "s01:secret"); err != nil {
		t.Fatal(err)
	}
	readContains(t, reader, "Authentication successful")
	readContains(t, reader, "You are connecting via: IPv4")
	readContains(t, reader, `"monitor":0`)

	if _, err := fmt.Fprintln(connection, "pong on"); err != nil {
		t.Fatal(err)
	}
	online6 := true
	update := AgentStats{
		Uptime: 90061, Load1: 1.25, Load5: 1, Load15: 0.75, CPU: 33.5, CPUCores: 4, CPUModel: "Test CPU",
		MemoryTotal: 1024, MemoryUsed: 512, SwapTotal: 128, SwapUsed: 2, HDDTotal: 10000, HDDUsed: 4000,
		NetworkRX: 123, NetworkTX: 456, NetworkIn: 1_000_000, NetworkOut: 2_000_000,
		Ping10010: 1, Ping189: 2, Ping10086: 3, Time10010: 10, Time189: 20, Time10086: 30,
		TCPCount: 10, UDPCount: 2, ProcessCount: 30, ThreadCount: 60, IORead: 1, IOWrite: 2,
		OS: "linux", Custom: "example=12", Online6: &online6,
	}
	payload, _ := json.Marshal(update)
	if _, err := fmt.Fprintf(connection, "update %s\n", payload); err != nil {
		t.Fatal(err)
	}
	readContains(t, reader, "0")

	eventually(t, time.Second, func() bool {
		servers := app.SnapshotStats()["servers"].([]any)
		server := servers[0].(map[string]any)
		return server["online4"] == true && server["online6"] == true && server["cpu_model"] == "Test CPU" && server["uptime"] == "1 天"
	})

	result, apiErr := app.ResetTraffic("s01")
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	stats := result["stats"].(map[string]any)
	if stats["last_network_in"] != int64(1_000_000) {
		t.Fatalf("traffic reset result: %#v", stats)
	}

	duplicate, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	duplicateReader := bufio.NewReader(duplicate)
	readContains(t, duplicateReader, "Authentication required")
	_, _ = fmt.Fprintln(duplicate, "s01:secret")
	readContains(t, duplicateReader, "Only one connection per user")
	_ = duplicate.Close()

	wrong, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	wrongReader := bufio.NewReader(wrong)
	readContains(t, wrongReader, "Authentication required")
	_, _ = fmt.Fprintln(wrong, "s01:wrong")
	readContains(t, wrongReader, "Wrong username")
	_ = wrong.Close()

	if apiErr := app.ReloadConfig(); apiErr != nil {
		t.Fatal(apiErr)
	}
	readContains(t, reader, "Server reloading")
	eventually(t, time.Second, func() bool {
		server := app.SnapshotStats()["servers"].([]any)[0].(map[string]any)
		return server["online4"] == false && server["online6"] == false
	})

	app.cancel()
	_ = listener.Close()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("agent server did not stop")
	}
}

func readContains(t *testing.T, reader *bufio.Reader, expected string) string {
	t.Helper()
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read %q: %v", expected, err)
	}
	if !strings.Contains(line, expected) {
		t.Fatalf("expected %q in %q", expected, line)
	}
	return line
}
