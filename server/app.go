package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Options struct {
	ConfigPath          string
	StatsPath           string
	WebDir              string
	HTTPAddr            string
	AgentAddr           string
	AdminToken          string
	CORSOrigin          string
	InsecureCallbackTLS bool
	Verbose             bool
}

type NodeState struct {
	Config         ServerConfig
	Connected      bool
	Connection     net.Conn
	ConnectionID   uint64
	Family         int
	Online4        bool
	Online6        bool
	Stats          AgentStats
	HasUpdate      bool
	LastNetworkIn  int64
	LastNetworkOut int64
	LastUpdate     time.Time
	AlarmLast      map[string]time.Time
	Pong           bool
}

type App struct {
	opts      Options
	startedAt time.Time
	ctx       context.Context
	cancel    context.CancelFunc

	mutationMu sync.Mutex
	configMu   sync.RWMutex
	document   ConfigDocument
	runtime    RuntimeConfig

	nodeMu       sync.RWMutex
	nodes        map[string]*NodeState
	connectionID atomic.Uint64
	generation   atomic.Uint64
	agentRunning atomic.Bool
	reloadWrites atomic.Int32

	certMu sync.RWMutex
	certs  map[string]*CertState

	statsWake chan struct{}
	persistMu sync.Mutex
	logger    *log.Logger
}

func NewApp(opts Options) (*App, error) {
	doc, runtime, err := readConfig(opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		opts:      opts,
		startedAt: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
		nodes:     make(map[string]*NodeState),
		certs:     make(map[string]*CertState),
		statsWake: make(chan struct{}, 1),
		logger:    log.New(os.Stdout, "serverstatus ", log.LstdFlags|log.Lmicroseconds),
	}
	app.applyValidatedConfig(doc, runtime, false)
	app.restorePersistentState()
	return app, nil
}

func (a *App) StartBackground() {
	go a.statsLoop()
	go a.sslLoop()
	a.wakeStatsWriter()
}

func (a *App) Close() {
	a.cancel()
	a.disconnectAll("Server shutting down...")
	_ = a.PersistStats()
}

func (a *App) ConfigSnapshot() ConfigDocument {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	clone, err := cloneDocument(a.document)
	if err != nil {
		panic(err)
	}
	return clone
}

func (a *App) RuntimeSnapshot() RuntimeConfig {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	result := a.runtime
	result.Servers = append([]ServerConfig(nil), a.runtime.Servers...)
	result.Monitors = append([]MonitorConfig(nil), a.runtime.Monitors...)
	result.SSLCerts = append([]SSLCertConfig(nil), a.runtime.SSLCerts...)
	result.Watchdogs = append([]CompiledWatchdog(nil), a.runtime.Watchdogs...)
	return result
}

func (a *App) ReplaceConfig(input ConfigDocument) (ConfigDocument, *APIError) {
	a.mutationMu.Lock()
	defer a.mutationMu.Unlock()
	normalized, runtime, apiErr := normalizeConfig(input)
	if apiErr != nil {
		return nil, apiErr
	}
	if err := writeConfig(a.opts.ConfigPath, normalized); err != nil {
		return nil, &APIError{Status: 500, Message: "config could not be written", Details: map[string]any{"error": err.Error()}}
	}
	a.applyValidatedConfig(normalized, runtime, true)
	return a.ConfigSnapshot(), nil
}

func (a *App) MutateConfig(mutate func(ConfigDocument) *APIError) (ConfigDocument, *APIError) {
	a.mutationMu.Lock()
	defer a.mutationMu.Unlock()
	doc := a.ConfigSnapshot()
	if apiErr := mutate(doc); apiErr != nil {
		return nil, apiErr
	}
	normalized, runtime, apiErr := normalizeConfig(doc)
	if apiErr != nil {
		return nil, apiErr
	}
	if err := writeConfig(a.opts.ConfigPath, normalized); err != nil {
		return nil, &APIError{Status: 500, Message: "config could not be written", Details: map[string]any{"error": err.Error()}}
	}
	a.applyValidatedConfig(normalized, runtime, true)
	return a.ConfigSnapshot(), nil
}

func (a *App) ReloadConfig() *APIError {
	a.mutationMu.Lock()
	defer a.mutationMu.Unlock()
	doc, runtime, err := readConfig(a.opts.ConfigPath)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			return apiErr
		}
		return &APIError{Status: 500, Message: "config could not be reloaded", Details: map[string]any{"error": err.Error()}}
	}
	a.applyValidatedConfig(doc, runtime, true)
	return nil
}

func (a *App) applyValidatedConfig(doc ConfigDocument, runtime RuntimeConfig, disconnect bool) {
	a.configMu.Lock()
	a.nodeMu.Lock()
	oldNodes := a.nodes
	newNodes := make(map[string]*NodeState, len(runtime.Servers))
	connections := make([]net.Conn, 0)
	for _, server := range runtime.Servers {
		node := &NodeState{Config: server, AlarmLast: make(map[string]time.Time)}
		if old := oldNodes[server.Username]; old != nil && sameServerIdentity(old.Config, server) {
			node.LastNetworkIn = old.LastNetworkIn
			node.LastNetworkOut = old.LastNetworkOut
			node.Stats = old.Stats
			node.HasUpdate = old.HasUpdate
			node.AlarmLast = old.AlarmLast
			if !disconnect {
				node.Connected = old.Connected
				node.Connection = old.Connection
				node.ConnectionID = old.ConnectionID
				node.Family = old.Family
				node.Online4 = old.Online4
				node.Online6 = old.Online6
			}
		}
		newNodes[server.Username] = node
	}
	if disconnect {
		for _, node := range oldNodes {
			if node.Connection != nil {
				connections = append(connections, node.Connection)
			}
		}
	}
	a.document = doc
	a.runtime = runtime
	a.nodes = newNodes
	a.generation.Add(1)
	a.nodeMu.Unlock()
	a.configMu.Unlock()

	a.reconcileCerts(runtime.SSLCerts)
	if disconnect {
		for _, conn := range connections {
			_, _ = conn.Write([]byte("Server reloading...\n"))
			_ = conn.Close()
		}
	}
	a.reloadWrites.Store(2)
	a.wakeStatsWriter()
}

func sameServerIdentity(left, right ServerConfig) bool {
	return left.Username == right.Username && left.Name == right.Name && left.Type == right.Type && left.Host == right.Host && left.Location == right.Location
}

func (a *App) disconnectAll(reason string) {
	a.nodeMu.Lock()
	connections := make([]net.Conn, 0)
	for _, node := range a.nodes {
		if node.Connection != nil {
			connections = append(connections, node.Connection)
			node.Connection = nil
			node.Connected = false
			node.Online4 = false
			node.Online6 = false
		}
	}
	a.nodeMu.Unlock()
	for _, conn := range connections {
		if reason != "" {
			_, _ = conn.Write([]byte(reason + "\n"))
		}
		_ = conn.Close()
	}
}

func (a *App) statsLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
		case <-a.statsWake:
		}
		if err := a.PersistStats(); err != nil {
			a.logger.Printf("write stats: %v", err)
		}
	}
}

func (a *App) wakeStatsWriter() {
	select {
	case a.statsWake <- struct{}{}:
	default:
	}
}

func (a *App) SnapshotStats() map[string]any {
	return a.snapshotStats(false)
}

func (a *App) snapshotStats(consumeReload bool) map[string]any {
	runtime := a.RuntimeSnapshot()
	now := time.Now()
	servers := make([]any, 0, len(runtime.Servers))
	a.nodeMu.Lock()
	for _, server := range runtime.Servers {
		if server.Disabled {
			continue
		}
		node := a.nodes[server.Username]
		if node == nil {
			continue
		}
		base := map[string]any{
			"name": server.Name, "type": server.Type, "host": server.Host, "location": server.Location,
			"online4": false, "online6": false,
		}
		if node.Connected && node.HasUpdate {
			s := node.Stats
			updateTrafficBaselines(node, s.NetworkIn, s.NetworkOut, monthResetWindow(now, server.MonthStart))
			base["online4"] = node.Online4
			base["online6"] = node.Online6
			base["uptime"] = formatUptime(s.Uptime)
			base["load_1"], base["load_5"], base["load_15"] = round2(s.Load1), round2(s.Load5), round2(s.Load15)
			base["ping_10010"], base["ping_189"], base["ping_10086"] = round2(s.Ping10010), round2(s.Ping189), round2(s.Ping10086)
			base["time_10010"], base["time_189"], base["time_10086"] = s.Time10010, s.Time189, s.Time10086
			base["tcp_count"], base["udp_count"] = s.TCPCount, s.UDPCount
			base["process_count"], base["thread_count"] = s.ProcessCount, s.ThreadCount
			base["network_rx"], base["network_tx"] = s.NetworkRX, s.NetworkTX
			base["network_in"], base["network_out"] = s.NetworkIn, s.NetworkOut
			base["cpu"], base["cpu_cores"], base["cpu_model"] = int(s.CPU), s.CPUCores, s.CPUModel
			base["memory_total"], base["memory_used"] = s.MemoryTotal, s.MemoryUsed
			base["swap_total"], base["swap_used"] = s.SwapTotal, s.SwapUsed
			base["hdd_total"], base["hdd_used"] = s.HDDTotal, s.HDDUsed
			base["last_network_in"] = trafficBaseline(s.NetworkIn, node.LastNetworkIn)
			base["last_network_out"] = trafficBaseline(s.NetworkOut, node.LastNetworkOut)
			base["io_read"], base["io_write"] = s.IORead, s.IOWrite
			base["custom"], base["os"] = s.Custom, s.OS
		} else {
			base["last_network_in"] = node.LastNetworkIn
			base["last_network_out"] = node.LastNetworkOut
			base["os"] = node.Stats.OS
			base["cpu_model"] = node.Stats.CPUModel
		}
		servers = append(servers, base)
	}
	a.nodeMu.Unlock()

	result := map[string]any{
		"servers":  servers,
		"sslcerts": a.sslSnapshot(runtime.SSLCerts, now),
		"updated":  strconv.FormatInt(now.Unix(), 10),
	}
	if a.reloadWrites.Load() > 0 {
		result["reload"] = true
		if consumeReload {
			a.reloadWrites.Add(-1)
		}
	}
	return result
}

func (a *App) PersistStats() error {
	a.persistMu.Lock()
	defer a.persistMu.Unlock()
	return writeStatsFile(a.opts.StatsPath, a.snapshotStats(true))
}

func monthResetWindow(now time.Time, monthStart int) bool {
	return now.Day() == clamp(monthStart, 1, 28) && now.Hour() == 0 && now.Minute() < 5
}

func trafficBaseline(current, baseline int64) int64 {
	if current == 0 || baseline == 0 {
		return current
	}
	return baseline
}

func updateTrafficBaselines(node *NodeState, currentIn, currentOut int64, reset bool) {
	if reset {
		node.LastNetworkIn = currentIn
		node.LastNetworkOut = currentOut
		return
	}
	if node.LastNetworkIn == 0 || (currentIn != 0 && node.LastNetworkIn > currentIn) {
		node.LastNetworkIn = currentIn
	}
	if node.LastNetworkOut == 0 || (currentOut != 0 && node.LastNetworkOut > currentOut) {
		node.LastNetworkOut = currentOut
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func formatUptime(seconds int64) string {
	days := seconds / 86400
	if days > 0 {
		return fmt.Sprintf("%d 天", days)
	}
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, (seconds/60)%60, seconds%60)
}

func (a *App) restorePersistentState() {
	data, err := os.ReadFile(a.opts.StatsPath)
	if err != nil {
		data, err = os.ReadFile(a.opts.StatsPath + "~")
	}
	if err != nil {
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var previous struct {
		Servers []map[string]any `json:"servers"`
	}
	if err := decoder.Decode(&previous); err != nil {
		a.logger.Printf("read previous stats: %v", err)
		return
	}
	a.nodeMu.Lock()
	defer a.nodeMu.Unlock()
	for _, node := range a.nodes {
		for _, saved := range previous.Servers {
			if fmt.Sprint(saved["name"]) != node.Config.Name || fmt.Sprint(saved["type"]) != node.Config.Type || fmt.Sprint(saved["host"]) != node.Config.Host || fmt.Sprint(saved["location"]) != node.Config.Location {
				continue
			}
			node.LastNetworkIn = anyInt64(saved["last_network_in"])
			node.LastNetworkOut = anyInt64(saved["last_network_out"])
			node.Stats.OS = anyString(saved["os"])
			node.Stats.CPUModel = anyString(saved["cpu_model"])
			break
		}
	}
}

func anyString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func anyInt64(value any) int64 {
	switch number := value.(type) {
	case json.Number:
		parsed, _ := number.Int64()
		return parsed
	case float64:
		return int64(number)
	case int64:
		return number
	case int:
		return int64(number)
	case string:
		parsed, _ := strconv.ParseInt(number, 10, 64)
		return parsed
	default:
		return 0
	}
}

func (a *App) ResetTraffic(username string) (map[string]any, *APIError) {
	a.nodeMu.Lock()
	node := a.nodes[username]
	if node == nil {
		a.nodeMu.Unlock()
		return nil, &APIError{Status: 404, Message: "server was not found", Details: map[string]any{"username": username}}
	}
	if !node.Connected || !node.HasUpdate {
		a.nodeMu.Unlock()
		return nil, &APIError{Status: 409, Message: "server has no current traffic counters; it may be offline", Details: map[string]any{"username": username}}
	}
	previousIn, previousOut := node.LastNetworkIn, node.LastNetworkOut
	networkIn, networkOut := node.Stats.NetworkIn, node.Stats.NetworkOut
	node.LastNetworkIn, node.LastNetworkOut = networkIn, networkOut
	server := node.Config
	a.nodeMu.Unlock()
	a.wakeStatsWriter()
	return map[string]any{
		"server": server,
		"stats": map[string]any{
			"network_in": networkIn, "network_out": networkOut,
			"previous_last_network_in": previousIn, "previous_last_network_out": previousOut,
			"last_network_in": networkIn, "last_network_out": networkOut,
			"month_in_before": max64(0, networkIn-previousIn), "month_out_before": max64(0, networkOut-previousOut),
		},
	}, nil
}

func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
