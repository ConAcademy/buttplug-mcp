// Copyright (c) 2025 Neomantra BV

package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ConAcademy/buttplug-mcp/internal/bp"
	"github.com/ConAcademy/buttplug-mcp/internal/mcp"
	"github.com/spf13/pflag"
)

///////////////////////////////////////////////////////////////////////////////

const (
	mcpServerVersion = "0.0.1"

	defaultSSEHostPort = ":8889"
	defaultLogDest     = "buttplug-mcp.log"
)

type Config struct {
	LogJSON bool // Log in JSON format instead of text
	Verbose bool // Verbose logging

	MCPConfig mcp.Config // MCP config
	BPConfig  bp.Config  // Buttplug config
}

///////////////////////////////////////////////////////////////////////////////

func main() {
	var config Config
	var showHelp bool
	var logFilename string

	pflag.StringVarP(&logFilename, "log-file", "l", "", "Log file destination (or MCP_LOG_FILE envvar). Default is stderr")
	pflag.BoolVarP(&config.LogJSON, "log-json", "j", false, "Log in JSON (default is plaintext)")
	pflag.StringVarP(&config.MCPConfig.SSEHostPort, "sse-host", "", "", "host:port to listen to SSE connections")
	pflag.BoolVarP(&config.MCPConfig.UseSSE, "sse", "", false, "Use SSE Transport (default is STDIO transport)")
	pflag.IntVarP(&config.BPConfig.WsPort, "ws-port", "", 0, "port to connect to the Buttplug Websocket server")
	pflag.DurationVarP(&config.BPConfig.DebounceDuration, "debounce", "d", bp.DefaultDebounceDuration, "duration for debounce (default is 20Hz = '50ms')")
	pflag.BoolVarP(&config.Verbose, "verbose", "v", false, "Verbose logging")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show help")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, "usage: %s [opts]\n\n", os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}

	if config.MCPConfig.SSEHostPort == "" {
		config.MCPConfig.SSEHostPort = defaultSSEHostPort
	}

	config.MCPConfig.Name = "buttplug-mcp"
	config.MCPConfig.Version = mcpServerVersion

	// Set up logging
	logWriter := os.Stderr // default is stderr
	if logFilename == "" { // prefer CLI option
		logFilename = os.Getenv("MCP_LOG_FILE")
	}
	if logFilename != "" {
		logFile, err := os.OpenFile(logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file: %s\n", err.Error())
			os.Exit(1)
		}
		logWriter = logFile
		defer logFile.Close()
	}

	var logLevel = slog.LevelInfo
	if config.Verbose {
		logLevel = slog.LevelDebug
	}

	var logger *slog.Logger
	if config.LogJSON {
		logger = slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{Level: logLevel}))
	} else {
		logger = slog.New(slog.NewTextHandler(logWriter, &slog.HandlerOptions{Level: logLevel}))
	}

	// Run our Buttplug manager
	var err error
	bpManager, err := bp.NewManager(config.BPConfig, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start Buttplug manager: %s\n", err.Error())
		os.Exit(1)
	}
	go func() {
		err := bpManager.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "bpm failed: %s\n", err.Error())
			os.Exit(1)
		}
	}()

	// Run our MCP server
	if err := mcp.RunRouter(config.MCPConfig, bpManager, logger); err != nil {
		logger.Error("mcp router error", "error", err.Error())
		os.Exit(1)
	}

	// TODO: I guess we should clean up the buttplug?
}
