package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

func compileWatchdog(rule WatchdogConfig, index int) (CompiledWatchdog, error) {
	normalized := normalizeLegacyExpression(rule.Rule)
	program, err := expr.Compile(normalized, expr.Env(WatchdogEnvironment{}))
	if err != nil {
		return CompiledWatchdog{}, err
	}
	return CompiledWatchdog{
		WatchdogConfig: rule,
		Key:            fmt.Sprintf("%d:%s", index, rule.Name),
		Normalized:     normalized,
		Program:        program,
	}, nil
}

func normalizeLegacyExpression(input string) string {
	var output strings.Builder
	var quote rune
	runes := []rune(input)
	for index := 0; index < len(runes); index++ {
		current := runes[index]
		if quote != 0 {
			output.WriteRune(current)
			if current == quote && (index == 0 || runes[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			output.WriteRune(current)
			continue
		}
		switch current {
		case '&':
			output.WriteString("&&")
			if index+1 < len(runes) && runes[index+1] == '&' {
				index++
			}
		case '|':
			output.WriteString("||")
			if index+1 < len(runes) && runes[index+1] == '|' {
				index++
			}
		case '=':
			previousOperator := index > 0 && strings.ContainsRune("!<>=", runes[index-1])
			nextEqual := index+1 < len(runes) && runes[index+1] == '='
			if previousOperator || nextEqual {
				output.WriteRune(current)
			} else {
				output.WriteString("==")
			}
		default:
			output.WriteRune(current)
		}
	}
	return output.String()
}

func (a *App) evaluateWatchdogs(username string, offline bool) {
	runtime := a.RuntimeSnapshot()
	now := time.Now()
	type pendingAlert struct {
		rule WatchdogConfig
		node ServerConfig
	}
	pending := make([]pendingAlert, 0)

	a.nodeMu.Lock()
	node := a.nodes[username]
	if node == nil || (!offline && !node.Connected) || (offline && node.Connected) {
		a.nodeMu.Unlock()
		return
	}
	stats := node.Stats
	if offline {
		stats = AgentStats{}
	}
	environment := watchdogEnvironment(node.Config, stats, node.Online4, node.Online6, node.LastNetworkIn, node.LastNetworkOut)
	for _, rule := range runtime.Watchdogs {
		result, err := expr.Run(rule.Program, environment)
		if err != nil || !expressionTruthy(result) {
			continue
		}
		if last := node.AlarmLast[rule.Key]; !last.IsZero() && now.Sub(last) < secondsDuration(rule.Interval) {
			continue
		}
		node.AlarmLast[rule.Key] = now
		if rule.Callback != "" {
			pending = append(pending, pendingAlert{rule: rule.WatchdogConfig, node: node.Config})
		}
	}
	a.nodeMu.Unlock()

	for _, alert := range pending {
		alert := alert
		go func() {
			message := fmt.Sprintf("【告警名称】 %s \n\n【告警时间】 %s  \n\n【用户名】 %s \n\n【节点名】 %s \n\n【虚拟化】 %s \n\n【主机名】 %s \n\n【位  置】 %s",
				alert.rule.Name, now.Format("2006-01-02 15:04:05"), alert.node.Username, alert.node.Name, alert.node.Type, alert.node.Host, alert.node.Location)
			if err := a.sendCallback(alert.rule.Callback, message, "ServerStatus"); err != nil {
				a.logger.Printf("watchdog %q callback: %v", alert.rule.Name, err)
			}
		}()
	}
}

type WatchdogEnvironment struct {
	Username       string  `expr:"username"`
	Name           string  `expr:"name"`
	NodeType       string  `expr:"type"`
	Host           string  `expr:"host"`
	Location       string  `expr:"location"`
	Load1          float64 `expr:"load_1"`
	Load5          float64 `expr:"load_5"`
	Load15         float64 `expr:"load_15"`
	Ping10010      float64 `expr:"ping_10010"`
	Ping189        float64 `expr:"ping_189"`
	Ping10086      float64 `expr:"ping_10086"`
	Time10010      float64 `expr:"time_10010"`
	Time189        float64 `expr:"time_189"`
	Time10086      float64 `expr:"time_10086"`
	TCPCount       float64 `expr:"tcp_count"`
	UDPCount       float64 `expr:"udp_count"`
	ProcessCount   float64 `expr:"process_count"`
	ThreadCount    float64 `expr:"thread_count"`
	NetworkRX      float64 `expr:"network_rx"`
	NetworkTX      float64 `expr:"network_tx"`
	NetworkIn      float64 `expr:"network_in"`
	NetworkOut     float64 `expr:"network_out"`
	LastNetworkIn  float64 `expr:"last_network_in"`
	LastNetworkOut float64 `expr:"last_network_out"`
	MemoryTotal    float64 `expr:"memory_total"`
	MemoryUsed     float64 `expr:"memory_used"`
	SwapTotal      float64 `expr:"swap_total"`
	SwapUsed       float64 `expr:"swap_used"`
	HDDTotal       float64 `expr:"hdd_total"`
	HDDUsed        float64 `expr:"hdd_used"`
	IORead         float64 `expr:"io_read"`
	IOWrite        float64 `expr:"io_write"`
	CPU            float64 `expr:"cpu"`
	Online4        float64 `expr:"online4"`
	Online6        float64 `expr:"online6"`
}

func watchdogEnvironment(config ServerConfig, stats AgentStats, online4, online6 bool, lastNetworkIn, lastNetworkOut int64) WatchdogEnvironment {
	boolNumber := func(value bool) float64 {
		if value {
			return 1
		}
		return 0
	}
	return WatchdogEnvironment{
		Username: config.Username, Name: config.Name, NodeType: config.Type, Host: config.Host, Location: config.Location,
		Load1: stats.Load1, Load5: stats.Load5, Load15: stats.Load15,
		Ping10010: stats.Ping10010, Ping189: stats.Ping189, Ping10086: stats.Ping10086,
		Time10010: float64(stats.Time10010), Time189: float64(stats.Time189), Time10086: float64(stats.Time10086),
		TCPCount: float64(stats.TCPCount), UDPCount: float64(stats.UDPCount),
		ProcessCount: float64(stats.ProcessCount), ThreadCount: float64(stats.ThreadCount),
		NetworkRX: float64(stats.NetworkRX), NetworkTX: float64(stats.NetworkTX),
		NetworkIn: float64(stats.NetworkIn), NetworkOut: float64(stats.NetworkOut),
		LastNetworkIn: float64(lastNetworkIn), LastNetworkOut: float64(lastNetworkOut),
		MemoryTotal: float64(stats.MemoryTotal), MemoryUsed: float64(stats.MemoryUsed),
		SwapTotal: float64(stats.SwapTotal), SwapUsed: float64(stats.SwapUsed),
		HDDTotal: float64(stats.HDDTotal), HDDUsed: float64(stats.HDDUsed),
		IORead: float64(stats.IORead), IOWrite: float64(stats.IOWrite), CPU: stats.CPU,
		Online4: boolNumber(online4), Online6: boolNumber(online6),
	}
}

func expressionTruthy(result any) bool {
	switch value := result.(type) {
	case bool:
		return value
	case int:
		return value != 0
	case int64:
		return value != 0
	case float64:
		return value != 0
	case string:
		parsed, _ := strconv.ParseBool(value)
		return parsed
	default:
		return false
	}
}

func (a *App) sendCallback(baseURL, message, signature string) error {
	requestURL := baseURL + url.QueryEscape(message)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if a.opts.InsecureCallbackTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec -- explicit compatibility option
	}
	client := &http.Client{Timeout: 6 * time.Second, Transport: transport}
	response, err := client.Post(requestURL, "application/x-www-form-urlencoded", strings.NewReader("signature="+url.QueryEscape(signature)))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %s", response.Status)
	}
	return nil
}
