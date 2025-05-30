// Copyright (c) 2025 Neomantra BV

package bp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/diamondburned/go-buttplug"
	"github.com/diamondburned/go-buttplug/device"
)

// DefaultDebounceDuration is hte Default used for DebounceDuration.  Default is 50ms, 20Hz.
const DefaultDebounceDuration = device.DebounceFrequency

// Config conifgures Manager
type Config struct {
	// DebounceDuration determines the period of frequency to debounce certain commands
	// when sending them to the websocket. It works around the internal event
	// buffers for real-time control. Default is '50ms', 20Hz.   -1 to disable.
	DebounceDuration time.Duration // Duration for debounce (default is 50ms)
	WsPort           int           // websocket port to start from
}

// Manager keeps track of Buttplug resources
type Manager struct {
	config Config
	logger *slog.Logger

	conn    *buttplug.Websocket
	manager *device.Manager
}

// NewManager returns a new buttplug.Manager given a Config.
// Returns nil and an error if any.
func NewManager(config Config, logger *slog.Logger) (*Manager, error) {
	return &Manager{
		config: config,
		logger: logger,
	}, nil
}

func (m *Manager) GetConnection() *buttplug.Websocket {
	return m.conn
}

func (m *Manager) GetConfig() Config {
	return m.config
}

func (m *Manager) GetDeviceManager() *device.Manager {
	return m.manager
}

///////////////////////////////////////////////////////////////////////////////

// From https://github.com/diamondburned/go-buttplug/blob/plug/cmd/buttplughttp/main.go
func (m *Manager) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	m.conn = buttplug.NewWebsocket()
	m.conn.DialTimeout = time.Second
	m.conn.DialDelay = 250 * time.Millisecond

	broadcaster := buttplug.NewBroadcaster()

	m.manager = device.NewManager()
	m.manager.DebounceFrequency = m.config.DebounceDuration
	m.manager.Listen(broadcaster.Listen())

	msgCh := broadcaster.Listen()

	// Start connecting and broadcasting messages at the same time.
	urlStr := fmt.Sprintf("ws://127.0.0.1:%d", m.config.WsPort)
	broadcaster.Start(m.conn.Open(ctx, urlStr))

	m.logger.Info("Working Buttplug connection", "url", urlStr)
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgCh:
			switch msg := msg.(type) {
			case *buttplug.ServerInfo:
				// Server is ready. Start scanning and ask for the list of
				// devices. The device manager will pick up the device messages for us.
				m.conn.Send(ctx,
					&buttplug.StartScanning{},
					&buttplug.RequestDeviceList{},
				)
			case *buttplug.DeviceList:
				for _, device := range msg.Devices {
					m.logger.Info("listed device", "name", device.DeviceName, "index", device.DeviceIndex)
				}
			case *buttplug.DeviceAdded:
				m.logger.Info("added device", "name", msg.DeviceName, "index", msg.DeviceIndex)
			case *buttplug.DeviceRemoved:
				m.logger.Info("removed device", "index", msg.DeviceIndex)
			case error:
				m.logger.Error("buttplug error", "msg", msg)
			}
		}
	}
}
