package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"linux-stalk/internal/protocol"
)

type clientFileConfig struct {
	ServerURL          string `json:"server_url"`
	APIKey             string `json:"api_key"`
	DeviceID           string `json:"device_id"`
	MinSendInterval    string `json:"min_send_interval"`
	MaxEventsPerMinute int    `json:"max_events_per_minute"`
	RequestTimeout     string `json:"request_timeout"`
}

type clientRuntimeConfig struct {
	ServerURL          string
	APIKey             string
	DeviceID           string
	MinSendInterval    time.Duration
	MaxEventsPerMinute int
	RequestTimeout     time.Duration
}

type uploadController struct {
	minSendInterval    time.Duration
	maxEventsPerMinute int
	requestTimeout     time.Duration
	lastStateHash      string
	lastSent           time.Time
	recentSent         []time.Time
}

func runPush(probe *atspiProbe, cfg config) error {
	fileConfig, err := loadClientConfig(cfg.configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client := &http.Client{
		Timeout: fileConfig.RequestTimeout,
	}
	controller := &uploadController{
		minSendInterval:    fileConfig.MinSendInterval,
		maxEventsPerMinute: fileConfig.MaxEventsPerMinute,
		requestTimeout:     fileConfig.RequestTimeout,
	}

	fmt.Println("== Push Client ==")
	fmt.Printf("Server URL: %s\n", fileConfig.ServerURL)
	fmt.Printf("Device ID: %s\n", fileConfig.DeviceID)
	fmt.Printf("Min Send Interval: %s\n", fileConfig.MinSendInterval)
	fmt.Printf("Max Events Per Minute: %d\n", fileConfig.MaxEventsPerMinute)
	fmt.Println("Trigger: Window.Activate")
	fmt.Println("Listening. Stop with Ctrl+C.")

	return probe.ListenAllEvents(ctx, func(event FocusEvent) {
		if !isPushTrigger(event) {
			return
		}

		payload, err := collectPayload(probe, fileConfig.DeviceID, event)
		if err != nil {
			fmt.Fprintf(os.Stderr, "push: collect payload: %v\n", err)
			return
		}

		if !controller.shouldSend(payload.StateHash, time.Now()) {
			return
		}

		if err := postPayload(ctx, client, fileConfig, payload); err != nil {
			fmt.Fprintf(os.Stderr, "push: post payload: %v\n", err)
			return
		}

		controller.markSent(payload.StateHash, time.Now())
		fmt.Printf("[%s] pushed %s | %s\n", time.Now().Format(time.RFC3339), payload.Trigger, payloadSummary(payload))
	})
}

func loadClientConfig(path string) (clientRuntimeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return clientRuntimeConfig{}, err
	}

	var raw clientFileConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return clientRuntimeConfig{}, err
	}

	cfg := clientRuntimeConfig{
		ServerURL:          raw.ServerURL,
		APIKey:             raw.APIKey,
		DeviceID:           raw.DeviceID,
		MinSendInterval:    5 * time.Second,
		MaxEventsPerMinute: 10,
		RequestTimeout:     10 * time.Second,
	}

	if raw.MinSendInterval != "" {
		if cfg.MinSendInterval, err = time.ParseDuration(raw.MinSendInterval); err != nil {
			return clientRuntimeConfig{}, fmt.Errorf("parse min_send_interval: %w", err)
		}
	}
	if raw.MaxEventsPerMinute > 0 {
		cfg.MaxEventsPerMinute = raw.MaxEventsPerMinute
	}
	if raw.RequestTimeout != "" {
		if cfg.RequestTimeout, err = time.ParseDuration(raw.RequestTimeout); err != nil {
			return clientRuntimeConfig{}, fmt.Errorf("parse request_timeout: %w", err)
		}
	}
	if cfg.ServerURL == "" || cfg.APIKey == "" || cfg.DeviceID == "" {
		return clientRuntimeConfig{}, fmt.Errorf("server_url, api_key, and device_id are required")
	}

	return cfg, nil
}

func isPushTrigger(event FocusEvent) bool {
	return event.Signal == ifaceEventWindow+".Activate" && !isNoisyApp(event.ApplicationName)
}

func collectPayload(probe *atspiProbe, deviceID string, event FocusEvent) (protocol.IngestPayload, error) {
	systemInfo, err := collectSystemInfo()
	if err != nil {
		return protocol.IngestPayload{}, err
	}
	systemInfo.ATSPIBusAddress = probe.BusAddress()

	apps, err := probe.ListApplications()
	if err != nil {
		return protocol.IngestPayload{}, err
	}
	focus, err := probe.FindFocusedApplication(apps)
	if err != nil {
		return protocol.IngestPayload{}, err
	}

	payload := protocol.IngestPayload{
		DeviceID:     deviceID,
		Trigger:      event.Signal,
		EventTime:    event.Timestamp,
		CapturedAt:   time.Now(),
		System:       systemToSnapshot(systemInfo),
		FocusedApp:   focusToSnapshot(focus),
		TriggerEvent: eventToSnapshot(event),
	}
	payload.StateHash = computeStateHash(payload)

	return payload, nil
}

func systemToSnapshot(info SystemInfo) protocol.SystemSnapshot {
	return protocol.SystemSnapshot{
		Hostname:          info.Hostname,
		PrettyOS:          info.PrettyOS,
		Kernel:            info.Kernel,
		Architecture:      info.Architecture,
		DesktopSession:    info.DesktopSession,
		CurrentDesktop:    info.CurrentDesktop,
		SessionType:       info.SessionType,
		Display:           info.Display,
		WaylandDisplay:    info.WaylandDisplay,
		PowerSummary:      info.PowerSummary,
		WiFiSSID:          info.WiFiSSID,
		BluetoothDevices:  append([]string(nil), info.BluetoothDevices...),
		MediaSessions:     append([]string(nil), info.MediaSessions...),
		SessionBusAddress: info.SessionBusAddress,
		ATSPIBusAddress:   info.ATSPIBusAddress,
	}
}

func focusToSnapshot(focus *FocusInfo) *protocol.FocusedSnapshot {
	if focus == nil {
		return nil
	}

	out := &protocol.FocusedSnapshot{
		Application: accessibleFromApp(focus.Application),
	}
	if focus.Object != nil {
		obj := protocol.AccessibleSnapshot{
			Name:    focus.Object.Name,
			Service: focus.Object.Ref.Service,
			Path:    string(focus.Object.Ref.Path),
			Role:    focus.Object.RoleName,
			States:  append([]string(nil), focus.Object.States...),
		}
		out.Object = &obj
	}
	return out
}

func accessibleFromApp(app ApplicationInfo) protocol.AccessibleSnapshot {
	return protocol.AccessibleSnapshot{
		Name:       app.Name,
		Service:    app.Ref.Service,
		Path:       string(app.Ref.Path),
		Role:       app.RoleName,
		States:     append([]string(nil), app.States...),
		Interfaces: append([]string(nil), app.Interfaces...),
	}
}

func eventToSnapshot(event FocusEvent) *protocol.EventSnapshot {
	return &protocol.EventSnapshot{
		Signal:          event.Signal,
		Sender:          event.Sender,
		Path:            string(event.Path),
		Interface:       event.Interface,
		Member:          event.Member,
		AccessibleName:  event.AccessibleName,
		AccessibleRole:  event.AccessibleRole,
		ApplicationName: event.ApplicationName,
		BodySummary:     event.BodySummary,
		Timestamp:       event.Timestamp,
	}
}

func computeStateHash(payload protocol.IngestPayload) string {
	copyPayload := payload
	copyPayload.StateHash = ""
	copyPayload.CapturedAt = time.Time{}
	copyPayload.EventTime = time.Time{}
	if copyPayload.TriggerEvent != nil {
		eventCopy := *copyPayload.TriggerEvent
		eventCopy.Timestamp = time.Time{}
		copyPayload.TriggerEvent = &eventCopy
	}

	data, _ := json.Marshal(copyPayload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c *uploadController) shouldSend(stateHash string, now time.Time) bool {
	if stateHash == c.lastStateHash {
		return false
	}
	if !c.lastSent.IsZero() && now.Sub(c.lastSent) < c.minSendInterval {
		return false
	}

	c.prune(now)
	return len(c.recentSent) < c.maxEventsPerMinute
}

func (c *uploadController) markSent(stateHash string, now time.Time) {
	c.lastStateHash = stateHash
	c.lastSent = now
	c.prune(now)
	c.recentSent = append(c.recentSent, now)
}

func (c *uploadController) prune(now time.Time) {
	cutoff := now.Add(-1 * time.Minute)
	kept := c.recentSent[:0]
	for _, ts := range c.recentSent {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	c.recentSent = kept
}

func postPayload(ctx context.Context, client *http.Client, cfg clientRuntimeConfig, payload protocol.IngestPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.ServerURL, "/")+"/ingest", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return nil
}

func payloadSummary(payload protocol.IngestPayload) string {
	if payload.FocusedApp != nil && payload.FocusedApp.Application.Name != "" {
		return payload.FocusedApp.Application.Name
	}
	if payload.TriggerEvent != nil && payload.TriggerEvent.ApplicationName != "" {
		return payload.TriggerEvent.ApplicationName
	}
	return "(unknown)"
}
