package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	var opts Options
	var legacyBind string
	var legacyPort int
	var printVersion bool

	flag.StringVar(&opts.ConfigPath, "config", envOr("CONFIG_PATH", "config.json"), "configuration file")
	flag.StringVar(&opts.ConfigPath, "c", envOr("CONFIG_PATH", "config.json"), "configuration file (shorthand)")
	flag.StringVar(&opts.StatsPath, "stats", os.Getenv("STATS_PATH"), "persistent stats JSON file")
	flag.StringVar(&opts.WebDir, "web-dir", envOr("WEB_DIR", "../web"), "WebUI directory")
	flag.StringVar(&opts.WebDir, "d", envOr("WEB_DIR", "../web"), "WebUI directory (shorthand)")
	flag.StringVar(&opts.HTTPAddr, "http", envOr("HTTP_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&opts.AgentAddr, "agent", envOr("AGENT_ADDR", ":35601"), "agent TCP listen address")
	flag.StringVar(&legacyBind, "bind", "", "agent bind address (legacy compatibility)")
	flag.StringVar(&legacyBind, "b", "", "agent bind address (legacy shorthand)")
	flag.IntVar(&legacyPort, "port", 0, "agent TCP port (legacy compatibility)")
	flag.IntVar(&legacyPort, "p", 0, "agent TCP port (legacy shorthand)")
	flag.BoolVar(&opts.Verbose, "verbose", envBool("VERBOSE", false), "verbose HTTP logging")
	flag.BoolVar(&opts.Verbose, "v", envBool("VERBOSE", false), "verbose HTTP logging (shorthand)")
	flag.BoolVar(&printVersion, "version", false, "print version and exit")
	flag.Parse()

	if printVersion {
		fmt.Printf("serverstatus %s commit=%s built=%s\n", version, commit, buildTime)
		return
	}
	if opts.StatsPath == "" {
		opts.StatsPath = filepath.Join(opts.WebDir, "json", "stats.json")
	}
	if legacyPort != 0 || legacyBind != "" {
		if legacyPort == 0 {
			legacyPort = 35601
		}
		opts.AgentAddr = net.JoinHostPort(legacyBind, strconv.Itoa(legacyPort))
	}
	opts.AdminToken = os.Getenv("ADMIN_TOKEN")
	opts.CORSOrigin = os.Getenv("ADMIN_CORS_ORIGIN")
	opts.InsecureCallbackTLS = envBool("INSECURE_CALLBACK_TLS", false)

	if err := os.MkdirAll(filepath.Dir(opts.StatsPath), 0o755); err != nil {
		fatalf("create stats directory: %v", err)
	}
	app, err := NewApp(opts)
	if err != nil {
		fatalf("start: %v", err)
	}
	app.StartBackground()

	httpServer := app.HTTPServer()
	agentServer := NewAgentServer(app)
	errorsChannel := make(chan error, 2)
	go func() {
		app.logger.Printf("HTTP listening on %s", opts.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errorsChannel <- fmt.Errorf("HTTP server: %w", err)
		}
	}()
	go func() {
		if err := agentServer.ListenAndServe(); err != nil {
			errorsChannel <- fmt.Errorf("agent server: %w", err)
		}
	}()

	reloadSignals := make(chan os.Signal, 1)
	stopSignals := make(chan os.Signal, 1)
	signal.Notify(reloadSignals, syscall.SIGHUP)
	signal.Notify(stopSignals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(reloadSignals)
	defer signal.Stop(stopSignals)

	go func() {
		for range reloadSignals {
			if apiErr := app.ReloadConfig(); apiErr != nil {
				app.logger.Printf("reload config: %s", apiErr.Message)
				continue
			}
			app.logger.Printf("configuration reloaded; generation=%d", app.generation.Load())
		}
	}()

	var fatalErr error
	select {
	case signalValue := <-stopSignals:
		app.logger.Printf("received %s; shutting down", signalValue)
	case fatalErr = <-errorsChannel:
		app.logger.Printf("fatal: %v", fatalErr)
	}
	app.cancel()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_ = httpServer.Shutdown(shutdownContext)
	cancel()
	app.Close()
	if fatalErr != nil {
		os.Exit(1)
	}
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "serverstatus: "+format+"\n", args...)
	os.Exit(1)
}
