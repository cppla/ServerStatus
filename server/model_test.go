package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/expr-lang/expr"
)

func TestNormalizeConfigPreservesCompatibility(t *testing.T) {
	doc := minimalTestConfig()
	doc["future"] = map[string]any{"enabled": true}
	doc["watchdog"] = []any{
		map[string]any{"name": "legacy", "rule": "cpu>90&load_1>5&username!='s01'", "interval": "600", "callback": nil},
		map[string]any{"name": "type", "rule": "tcp_count>600&type='Oracle'", "interval": 60},
	}
	doc["servers"].([]any)[0].(map[string]any)["future_field"] = "kept"

	normalized, runtime, apiErr := normalizeConfig(doc)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if len(runtime.Watchdogs) != 2 || runtime.Watchdogs[0].Normalized != "cpu>90&&load_1>5&&username!='s01'" {
		t.Fatalf("legacy rules were not normalized: %#v", runtime.Watchdogs)
	}
	server := normalized["servers"].([]any)[0].(map[string]any)
	if server["future_field"] != "kept" || normalized["future"] == nil {
		t.Fatal("unknown config fields were discarded")
	}
	if server["monthstart"] != json.Number("1") && server["monthstart"] != 1 {
		t.Fatalf("monthstart not normalized: %#v", server["monthstart"])
	}
}

func TestWatchdogLegacyRulesEvaluate(t *testing.T) {
	rules := []string{
		"online4=0&online6=0",
		"(memory_used/memory_total)*100>90&memory_total>1048576",
		"tcp_count>600&type='Oracle'",
		"(network_out-last_network_out)/1024/1024/1024>18&(username='aliyun1'|username='aliyun2')",
	}
	for index, rule := range rules {
		compiled, err := compileWatchdog(WatchdogConfig{Name: "test", Rule: rule, Interval: 1}, index)
		if err != nil {
			t.Fatalf("rule %q: %v", rule, err)
		}
		environment := watchdogEnvironment(ServerConfig{Username: "aliyun1", Type: "Oracle"}, AgentStats{MemoryTotal: 2_000_000, MemoryUsed: 1_900_000, TCPCount: 700, NetworkOut: 30 << 30}, false, false, 0, 0)
		if _, err := expr.Run(compiled.Program, environment); err != nil {
			t.Fatalf("run %q: %v", rule, err)
		}
	}
}

func TestConfigValidationErrors(t *testing.T) {
	doc := minimalTestConfig()
	doc["servers"] = append(doc["servers"].([]any), doc["servers"].([]any)[0])
	_, _, apiErr := normalizeConfig(doc)
	if apiErr == nil || apiErr.Status != 409 {
		t.Fatalf("expected duplicate username error, got %#v", apiErr)
	}

	if _, err := decodeDocument([]byte(`{"servers":[]} {"servers":[]}`)); err == nil || !strings.Contains(err.Error(), "more than one") {
		t.Fatalf("expected trailing JSON error, got %v", err)
	}
}

func TestFormattingHelpers(t *testing.T) {
	if got := formatUptime(90061); got != "1 天" {
		t.Fatalf("formatUptime=%q", got)
	}
	if got := formatUptime(3661); got != "01:01:01" {
		t.Fatalf("formatUptime=%q", got)
	}
	if got, err := certificateHost("https://example.com/path"); err != nil || got != "example.com" {
		t.Fatalf("certificateHost=%q", got)
	}
	if got := secondsDuration(0); got != time.Second {
		t.Fatalf("secondsDuration(0)=%s", got)
	}
	if got := secondsDuration(int(^uint(0) >> 1)); got < 365*24*time.Hour {
		t.Fatalf("large interval overflowed or was truncated: %s", got)
	}
}
