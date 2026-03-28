export interface DeviceRow {
  device_id: string;
  event_count: number;
  latest_event_time: string;
  latest_seen_at: string;
}

export interface AccessibleSnapshot {
  name?: string;
  service?: string;
  path?: string;
  role?: string;
  states?: string[];
  interfaces?: string[];
}

export interface EventSnapshot {
  signal?: string;
  sender?: string;
  path?: string;
  interface?: string;
  member?: string;
  accessible_name?: string;
  accessible_role?: string;
  application_name?: string;
  body_summary?: string;
  timestamp: string;
}

export interface SystemSnapshot {
  hostname?: string;
  pretty_os?: string;
  kernel?: string;
  architecture?: string;
  desktop_session?: string;
  current_desktop?: string;
  session_type?: string;
  display?: string;
  wayland_display?: string;
  power_summary?: string;
  wifi_ssid?: string;
  bluetooth_devices?: string[];
  media_sessions?: string[];
  session_bus_address?: string;
  atspi_bus_address?: string;
}

export interface FocusedSnapshot {
  application: AccessibleSnapshot;
  object?: AccessibleSnapshot;
}

export interface IngestPayload {
  device_id: string;
  trigger: string;
  event_time: string;
  captured_at: string;
  state_hash: string;
  system: SystemSnapshot;
  focused_app?: FocusedSnapshot;
  trigger_event?: EventSnapshot;
}

export interface EventRow {
  received_at: string;
  payload: IngestPayload;
}
