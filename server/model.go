package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr/vm"
)

const maxRequestBody = 1 << 20

type ConfigDocument map[string]any

type ServerConfig struct {
	Username   string `json:"username"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Host       string `json:"host"`
	Location   string `json:"location"`
	Password   string `json:"password"`
	MonthStart int    `json:"monthstart"`
	Disabled   bool   `json:"disabled,omitempty"`
}

type MonitorConfig struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Interval int    `json:"interval"`
	Type     string `json:"type"`
}

type SSLCertConfig struct {
	Name     string `json:"name"`
	Domain   string `json:"domain"`
	Port     int    `json:"port"`
	Interval int    `json:"interval"`
	Callback string `json:"callback"`
}

type WatchdogConfig struct {
	Name     string `json:"name"`
	Rule     string `json:"rule"`
	Interval int    `json:"interval"`
	Callback string `json:"callback"`
}

type CompiledWatchdog struct {
	WatchdogConfig
	Key        string
	Normalized string
	Program    *vm.Program
}

type RuntimeConfig struct {
	Servers   []ServerConfig
	Monitors  []MonitorConfig
	SSLCerts  []SSLCertConfig
	Watchdogs []CompiledWatchdog
}

type AgentStats struct {
	Uptime       int64   `json:"uptime"`
	Load1        float64 `json:"load_1"`
	Load5        float64 `json:"load_5"`
	Load15       float64 `json:"load_15"`
	Ping10010    float64 `json:"ping_10010"`
	Ping189      float64 `json:"ping_189"`
	Ping10086    float64 `json:"ping_10086"`
	Time10010    int64   `json:"time_10010"`
	Time189      int64   `json:"time_189"`
	Time10086    int64   `json:"time_10086"`
	TCPCount     int64   `json:"tcp"`
	UDPCount     int64   `json:"udp"`
	ProcessCount int64   `json:"process"`
	ThreadCount  int64   `json:"thread"`
	NetworkRX    int64   `json:"network_rx"`
	NetworkTX    int64   `json:"network_tx"`
	NetworkIn    int64   `json:"network_in"`
	NetworkOut   int64   `json:"network_out"`
	MemoryTotal  int64   `json:"memory_total"`
	MemoryUsed   int64   `json:"memory_used"`
	SwapTotal    int64   `json:"swap_total"`
	SwapUsed     int64   `json:"swap_used"`
	HDDTotal     int64   `json:"hdd_total"`
	HDDUsed      int64   `json:"hdd_used"`
	IORead       int64   `json:"io_read"`
	IOWrite      int64   `json:"io_write"`
	CPU          float64 `json:"cpu"`
	CPUCores     int64   `json:"cpu_cores"`
	CPUModel     string  `json:"cpu_model"`
	Custom       string  `json:"custom"`
	OS           string  `json:"os"`
	Online4      *bool   `json:"online4"`
	Online6      *bool   `json:"online6"`
}

type APIError struct {
	Status  int
	Message string
	Details any
}

func (e *APIError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Details)
	}
	return e.Message
}

func decodeDocument(data []byte) (ConfigDocument, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var doc ConfigDocument
	if err := decoder.Decode(&doc); err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("config must be a JSON object")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("config contains more than one JSON value")
	}
	return doc, nil
}

func cloneDocument(doc ConfigDocument) (ConfigDocument, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return decodeDocument(data)
}

type collectionSpec struct {
	itemName string
	idField  string
	required []string
	optional []string
	defaults map[string]int
}

var collectionSpecs = map[string]collectionSpec{
	"servers": {
		itemName: "server", idField: "username",
		required: []string{"username", "name", "type", "host", "location", "password"},
		optional: []string{"monthstart", "disabled"},
		defaults: map[string]int{"monthstart": 1},
	},
	"monitors": {
		itemName: "monitor", idField: "name",
		required: []string{"name", "host", "type"},
		optional: []string{"interval"},
		defaults: map[string]int{"interval": 600},
	},
	"sslcerts": {
		itemName: "sslcert", idField: "name",
		required: []string{"name", "domain"},
		optional: []string{"port", "interval", "callback"},
		defaults: map[string]int{"port": 443, "interval": 7200},
	},
	"watchdog": {
		itemName: "watchdog", idField: "name",
		required: []string{"name", "rule"},
		optional: []string{"interval", "callback"},
		defaults: map[string]int{"interval": 600},
	},
}

func normalizeConfig(input ConfigDocument) (ConfigDocument, RuntimeConfig, *APIError) {
	doc, err := cloneDocument(input)
	if err != nil {
		return nil, RuntimeConfig{}, &APIError{Status: 400, Message: "config must be a JSON object", Details: map[string]any{"error": err.Error()}}
	}

	for _, key := range []string{"servers", "monitors", "sslcerts", "watchdog"} {
		spec := collectionSpecs[key]
		raw, exists := doc[key]
		if !exists || raw == nil {
			raw = []any{}
		}
		items, ok := raw.([]any)
		if !ok {
			return nil, RuntimeConfig{}, &APIError{Status: 400, Message: key + " must be an array"}
		}
		normalized := make([]any, 0, len(items))
		seen := make(map[string]struct{})
		for index, rawItem := range items {
			item, ok := rawItem.(map[string]any)
			if !ok {
				return nil, RuntimeConfig{}, &APIError{Status: 400, Message: spec.itemName + " must be an object", Details: map[string]any{"index": index}}
			}
			missing := make([]string, 0)
			for _, field := range spec.required {
				value, ok := item[field]
				text := ""
				if ok && value != nil {
					text = strings.TrimSpace(fmt.Sprint(value))
				}
				if text == "" {
					missing = append(missing, field)
				} else {
					item[field] = text
				}
			}
			if len(missing) > 0 {
				return nil, RuntimeConfig{}, &APIError{Status: 400, Message: spec.itemName + " has missing required fields", Details: map[string]any{"missing": missing, "index": index}}
			}

			for field, fallback := range spec.defaults {
				value, apiErr := normalizeInteger(item[field], field, index, fallback)
				if apiErr != nil {
					return nil, RuntimeConfig{}, apiErr
				}
				switch {
				case key == "servers" && field == "monthstart":
					value = clamp(value, 1, 28)
				case key == "sslcerts" && field == "port":
					value = clamp(value, 1, 65535)
				default:
					if value < 1 {
						value = 1
					}
				}
				item[field] = value
			}

			if key == "servers" {
				if value, exists := item["disabled"]; exists {
					disabled, ok := normalizeBool(value)
					if !ok {
						return nil, RuntimeConfig{}, &APIError{Status: 400, Message: "disabled must be a boolean", Details: map[string]any{"index": index}}
					}
					item["disabled"] = disabled
				}
				username := item["username"].(string)
				if _, duplicate := seen[username]; duplicate {
					return nil, RuntimeConfig{}, &APIError{Status: 409, Message: "duplicate server username", Details: map[string]any{"username": username}}
				}
				seen[username] = struct{}{}
			}
			if key == "sslcerts" || key == "watchdog" {
				value := item["callback"]
				if value == nil {
					item["callback"] = ""
				} else {
					item["callback"] = strings.TrimSpace(fmt.Sprint(value))
				}
			}
			normalized = append(normalized, item)
		}
		doc[key] = normalized
	}

	runtime, apiErr := buildRuntimeConfig(doc)
	if apiErr != nil {
		return nil, RuntimeConfig{}, apiErr
	}
	return doc, runtime, nil
}

func buildRuntimeConfig(doc ConfigDocument) (RuntimeConfig, *APIError) {
	data, err := json.Marshal(doc)
	if err != nil {
		return RuntimeConfig{}, &APIError{Status: 400, Message: "config could not be encoded", Details: map[string]any{"error": err.Error()}}
	}
	var raw struct {
		Servers   []ServerConfig   `json:"servers"`
		Monitors  []MonitorConfig  `json:"monitors"`
		SSLCerts  []SSLCertConfig  `json:"sslcerts"`
		Watchdogs []WatchdogConfig `json:"watchdog"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return RuntimeConfig{}, &APIError{Status: 400, Message: "config has invalid field types", Details: map[string]any{"error": err.Error()}}
	}
	runtime := RuntimeConfig{Servers: raw.Servers, Monitors: raw.Monitors, SSLCerts: raw.SSLCerts}
	for index, rule := range raw.Watchdogs {
		compiled, err := compileWatchdog(rule, index)
		if err != nil {
			return RuntimeConfig{}, &APIError{Status: 400, Message: "watchdog rule is invalid", Details: map[string]any{"index": index, "name": rule.Name, "error": err.Error()}}
		}
		runtime.Watchdogs = append(runtime.Watchdogs, compiled)
	}
	return runtime, nil
}

func normalizeInteger(raw any, field string, index, fallback int) (int, *APIError) {
	if raw == nil || raw == "" {
		return fallback, nil
	}
	var value int64
	var err error
	switch v := raw.(type) {
	case json.Number:
		value, err = strconv.ParseInt(v.String(), 10, 64)
	case float64:
		if math.Trunc(v) != v {
			err = fmt.Errorf("not an integer")
		} else {
			value = int64(v)
		}
	case float32:
		if math.Trunc(float64(v)) != float64(v) {
			err = fmt.Errorf("not an integer")
		} else {
			value = int64(v)
		}
	case int:
		value = int64(v)
	case int64:
		value = v
	case string:
		value, err = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	default:
		err = fmt.Errorf("unsupported value")
	}
	if err != nil || value > math.MaxInt || value < math.MinInt {
		return 0, &APIError{Status: 400, Message: field + " must be an integer", Details: map[string]any{"index": index}}
	}
	return int(value), nil
}

func normalizeBool(raw any) (bool, bool) {
	switch value := raw.(type) {
	case bool:
		return value, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		return parsed, err == nil
	case json.Number:
		if value.String() == "0" {
			return false, true
		}
		if value.String() == "1" {
			return true, true
		}
	}
	return false, false
}

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func secondsDuration(value int) time.Duration {
	if value < 1 {
		value = 1
	}
	maxDuration := time.Duration(1<<63 - 1)
	if int64(value) > int64(maxDuration/time.Second) {
		return maxDuration
	}
	return time.Duration(value) * time.Second
}
