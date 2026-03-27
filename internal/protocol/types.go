package protocol

import "time"

type IngestPayload struct {
	DeviceID     string           `json:"device_id"`
	Trigger      string           `json:"trigger"`
	EventTime    time.Time        `json:"event_time"`
	CapturedAt   time.Time        `json:"captured_at"`
	StateHash    string           `json:"state_hash"`
	System       SystemSnapshot   `json:"system"`
	FocusedApp   *FocusedSnapshot `json:"focused_app,omitempty"`
	TriggerEvent *EventSnapshot   `json:"trigger_event,omitempty"`
}

type SystemSnapshot struct {
	Hostname          string   `json:"hostname,omitempty"`
	PrettyOS          string   `json:"pretty_os,omitempty"`
	Kernel            string   `json:"kernel,omitempty"`
	Architecture      string   `json:"architecture,omitempty"`
	DesktopSession    string   `json:"desktop_session,omitempty"`
	CurrentDesktop    string   `json:"current_desktop,omitempty"`
	SessionType       string   `json:"session_type,omitempty"`
	Display           string   `json:"display,omitempty"`
	WaylandDisplay    string   `json:"wayland_display,omitempty"`
	PowerSummary      string   `json:"power_summary,omitempty"`
	WiFiSSID          string   `json:"wifi_ssid,omitempty"`
	BluetoothDevices  []string `json:"bluetooth_devices,omitempty"`
	MediaSessions     []string `json:"media_sessions,omitempty"`
	SessionBusAddress string   `json:"session_bus_address,omitempty"`
	ATSPIBusAddress   string   `json:"atspi_bus_address,omitempty"`
}

type FocusedSnapshot struct {
	Application AccessibleSnapshot  `json:"application"`
	Object      *AccessibleSnapshot `json:"object,omitempty"`
}

type AccessibleSnapshot struct {
	Name       string   `json:"name,omitempty"`
	Service    string   `json:"service,omitempty"`
	Path       string   `json:"path,omitempty"`
	Role       string   `json:"role,omitempty"`
	States     []string `json:"states,omitempty"`
	Interfaces []string `json:"interfaces,omitempty"`
}

type EventSnapshot struct {
	Signal          string    `json:"signal,omitempty"`
	Sender          string    `json:"sender,omitempty"`
	Path            string    `json:"path,omitempty"`
	Interface       string    `json:"interface,omitempty"`
	Member          string    `json:"member,omitempty"`
	AccessibleName  string    `json:"accessible_name,omitempty"`
	AccessibleRole  string    `json:"accessible_role,omitempty"`
	ApplicationName string    `json:"application_name,omitempty"`
	BodySummary     string    `json:"body_summary,omitempty"`
	Timestamp       time.Time `json:"timestamp,omitempty"`
}
