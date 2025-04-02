// Copyright (c) 2025 Neomantra BV

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"

	"github.com/ConAcademy/buttplug-mcp/internal/bp"
	"github.com/diamondburned/go-buttplug"
	"github.com/diamondburned/go-buttplug/device"
	"github.com/mark3labs/mcp-go/mcp"
	mcp_server "github.com/mark3labs/mcp-go/server"
)

const (
	regex10  = `^(0(\.\d+)?|1(\.0+)?)$`
	regexInt = `^[0-9]*$`
)

// Config is configuration for our MCP server
type Config struct {
	Name    string // Service Name
	Version string // Service Version

	UseSSE      bool   // Use SSE Transport instead of STDIO
	SSEHostPort string // HostPort to use for SSE
}

// we resort to module-global variable rather than setting up closures
var bpManager *bp.Manager

//////////////////////////////////////////////////////////////////////////////

func RunRouter(config Config, bpm *bp.Manager, logger *slog.Logger) error {
	// Set module global for handlers
	bpManager = bpm

	// Create the MCP Server
	mcpServer := mcp_server.NewMCPServer(config.Name, config.Version)
	registerTools(mcpServer)

	if config.UseSSE {
		sseServer := mcp_server.NewSSEServer(mcpServer)
		logger.Info("MCP SSE server started", "hostPort", config.SSEHostPort)
		if err := sseServer.Start(config.SSEHostPort); err != nil {
			return fmt.Errorf("MCP SSE server error: %w", err)
		}
	} else {
		logger.Info("MCP STDIO server started")
		if err := mcp_server.ServeStdio(mcpServer); err != nil {
			return fmt.Errorf("MCP STDIO server error: %w", err)
		}
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////

// registerTools registers tools+metadata with the passed MCPServer
func registerTools(mcpServer *mcp_server.MCPServer) error {
	// /devices
	mcpServer.AddResource(mcp.NewResource("/devices", "Device List",
		mcp.WithResourceDescription("List of connected Buttplug devices in JSON"),
		mcp.WithMIMEType("application/json"),
	), getDeviceListHandler)
	// /device/{id}
	mcpServer.AddResourceTemplate(mcp.NewResourceTemplate("/device/{id}", "Device Info by ID",
		mcp.WithTemplateDescription("Device information by device ID where `id` is a number from `/devices`"),
		mcp.WithTemplateMIMEType("application/json"),
	), getDeviceInfoHandler)
	// /device/{id}/rssi
	mcpServer.AddResourceTemplate(mcp.NewResourceTemplate("/device/{id}/rssi", "Signal Level for Device by ID",
		mcp.WithTemplateDescription("RSSI signal level by device ID where `id` is a number from `/devices`"),
		mcp.WithTemplateMIMEType("application/json"),
	), getDeviceRssiHandler)
	// /device/{id}/battery
	mcpServer.AddResourceTemplate(mcp.NewResourceTemplate("/device/{id}/battery", "Battery Level for Device by ID",
		mcp.WithTemplateDescription("Battery level by device ID where `id` is a number from `/devices`"),
		mcp.WithTemplateMIMEType("application/json"),
	), getDeviceBatteryHandler)
	// /device/vibrate
	mcpServer.AddTool(mcp.NewTool("device_vibrate",
		mcp.WithDescription("Vibrates device by `id`, selecting `strength` and optional `motor`"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Device ID to query, sourced from `/devices`"),
			mcp.Pattern(regexInt),
		),
		mcp.WithNumber("strength",
			mcp.Required(),
			mcp.Description("Strength from 0.0 to 1.0, with 0.0 being off and 1.0 being full"),
			mcp.Pattern(regex10),
		),
		mcp.WithNumber("motor",
			mcp.Description("Motor number to vibrate, defaults to 0"),
			mcp.Pattern(regexInt),
		),
	), vibrateDeviceHandler)

	return nil
}

type RssiResponse struct {
	RssiLevel float64 `json:"rssi_level"`
}

type BatteryResponse struct {
	BatteryLevel float64 `json:"battery_level"`
}

///////////////////////////////////////////////////////////////////////////////

func getDeviceListHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	if bpManager == nil {
		return nil, fmt.Errorf("Buttplug manager not initialized")
	}

	devices := bpManager.GetDeviceManager().Devices()

	jbytes, err := json.Marshal(devices)
	if err != nil {
		return nil, fmt.Errorf("failed to json.Marshal devices: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "/devices",
			MIMEType: "application/json",
			Text:     string(jbytes),
		},
	}, nil
}

func getDeviceInfoHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	ctrl, err := controllerFromPattern(request, "/device/:id")
	if err != nil {
		return nil, fmt.Errorf("failed to extract controller: %w", err)
	}

	jbytes, err := json.Marshal(ctrl.Device)
	if err != nil {
		return nil, fmt.Errorf("failed to json.Marshal devices: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jbytes),
		},
	}, nil
}

func getDeviceRssiHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	ctrl, err := controllerFromPattern(request, "/device/:id/rssi")
	if err != nil {
		return nil, fmt.Errorf("failed to extract controller: %w", err)
	}

	rssiLevel, err := ctrl.RSSILevel()
	if err != nil {
		return nil, fmt.Errorf("failed to query rssi: %w", err)
	}

	jbytes, err := json.Marshal(RssiResponse{
		RssiLevel: rssiLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to json.Marshal rssi: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jbytes),
		},
	}, nil
}

func getDeviceBatteryHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	ctrl, err := controllerFromPattern(request, "/device/:id/battery")
	if err != nil {
		return nil, fmt.Errorf("failed to extract controller: %w", err)
	}

	batteryLevel, err := ctrl.Battery()
	if err != nil {
		return nil, fmt.Errorf("failed to query battery: %w", err)
	}

	jbytes, err := json.Marshal(BatteryResponse{
		BatteryLevel: batteryLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to json.Marshal battery: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jbytes),
		},
	}, nil
}

func vibrateDeviceHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var deviceID, motorID int
	var strength float64
	var err error
	var ok bool
	args := request.Params.Arguments

	if deviceID, ok = args["id"].(int); !ok {
		return nil, errors.New("id must be set")
	}
	if strength, ok = args["strength"].(float64); !ok {
		return nil, errors.New("strength must be set")
	}
	if motorID, ok = args["motor"].(int); !ok {
		motorID = 0 // it's OK, it's optional and we default to 0
	}

	ctrl := bpManager.GetDeviceManager().Controller(
		bpManager.GetConnection(),
		buttplug.DeviceIndex(deviceID))
	if ctrl == nil {
		return nil, fmt.Errorf("Device %d not found", deviceID)
	}

	// var form struct {
	// 	Motor    int     `json:"motor"` // default 0
	// 	Strength float64 `json:"strength,required"`
	// }

	err = ctrl.Vibrate(map[int]float64{
		motorID: strength,
	})
	if err != nil {
		return nil, fmt.Errorf("Vibrate on device %d failed: %w", deviceID, err)
	}

	return mcp.NewToolResultText(`{ "success": true }`), nil
}

///////////////////////////////////////////////////////////////////////////////

// controllerFromPattern gets the device's controller from the request. It
// writes the error directly into the given response writer and returns nil if
// the device cannot be found.
func controllerFromPattern(request mcp.ReadResourceRequest, pattern string) (*device.Controller, error) {
	// Parse URL for analysis
	parsedURL, err := url.Parse(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("error parsing uri: %w", err)
	}

	// Extract the ID from the path
	params := extractPattern(pattern, parsedURL.Path)
	deviceIDStr, found := params["id"]
	if !found {
		return nil, fmt.Errorf("Device ID not found in path")
	}

	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		return nil, fmt.Errorf("Device ID could not be converted to integer")
	}

	ctrl := bpManager.GetDeviceManager().Controller(bpManager.GetConnection(), buttplug.DeviceIndex(deviceID))
	if ctrl == nil {
		return nil, fmt.Errorf("Device %d not found", deviceID)
	}

	return ctrl, nil
}

///////////////////////////////////////////////////////////////////////////////

// extractPattern scans a path for pattern, putting `:var` into a returned map by name
func extractPattern(pattern string, path string) map[string]string {
	regex := regexp.MustCompile(`:([a-zA-Z0-9]+)`)
	matches := regex.FindAllStringSubmatch(pattern, -1)

	patternRegex := regexp.MustCompile("^" + regex.ReplaceAllString(pattern, "([^/]+)") + "$")
	pathMatches := patternRegex.FindStringSubmatch(path)

	if len(pathMatches) == 0 {
		return nil
	}

	params := make(map[string]string)
	for i, match := range matches {
		params[match[1]] = pathMatches[i+1]
	}

	return params
}
