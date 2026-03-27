#!/usr/bin/env bash
set -u

status=0

section() {
  printf '\n== %s ==\n' "$1"
}

ok() {
  printf '[OK] %s\n' "$1"
}

warn() {
  printf '[WARN] %s\n' "$1"
}

fail() {
  printf '[FAIL] %s\n' "$1"
  status=1
}

have() {
  command -v "$1" >/dev/null 2>&1
}

section "Environment"
printf 'DESKTOP_SESSION=%s\n' "${DESKTOP_SESSION:-}"
printf 'XDG_CURRENT_DESKTOP=%s\n' "${XDG_CURRENT_DESKTOP:-}"
printf 'XDG_SESSION_TYPE=%s\n' "${XDG_SESSION_TYPE:-}"
printf 'DISPLAY=%s\n' "${DISPLAY:-}"
printf 'WAYLAND_DISPLAY=%s\n' "${WAYLAND_DISPLAY:-}"
printf 'DBUS_SESSION_BUS_ADDRESS=%s\n' "${DBUS_SESSION_BUS_ADDRESS:-}"

if [[ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ]]; then
  fail "DBUS_SESSION_BUS_ADDRESS is empty. The user session bus is required."
else
  ok "Session bus address is present."
fi

section "Tooling"
for cmd in dbus-send sed awk; do
  if have "$cmd"; then
    ok "Found $cmd"
  else
    fail "Missing required command: $cmd"
  fi
done

if ! have dbus-send; then
  printf '\nProbe failed before D-Bus checks.\n'
  exit "$status"
fi

section "Session Bus -> org.a11y.Bus"
session_reply="$(dbus-send --session --dest=org.a11y.Bus --print-reply /org/a11y/bus org.a11y.Bus.GetAddress 2>&1)"
session_rc=$?
printf '%s\n' "$session_reply"

if [[ $session_rc -ne 0 ]]; then
  fail "Could not call org.a11y.Bus.GetAddress on the session bus."
  printf '\nFinal status: FAIL\n'
  exit "$status"
fi

atspi_addr="$(printf '%s\n' "$session_reply" | sed -n 's/.*string "\(.*\)"/\1/p' | head -n1)"
if [[ -z "$atspi_addr" ]]; then
  fail "AT-SPI bus address was not parsed from org.a11y.Bus.GetAddress."
  printf '\nFinal status: FAIL\n'
  exit "$status"
fi
ok "AT-SPI bus address: $atspi_addr"

section "AT-SPI Root Probe"
child_reply="$(dbus-send --bus="$atspi_addr" --dest=org.a11y.atspi.Registry --print-reply /org/a11y/atspi/accessible/root org.freedesktop.DBus.Properties.Get string:org.a11y.atspi.Accessible string:ChildCount 2>&1)"
child_rc=$?
printf '%s\n' "$child_reply"

if [[ $child_rc -ne 0 ]]; then
  fail "Could not read ChildCount from the AT-SPI root accessible."
  printf '\nFinal status: FAIL\n'
  exit "$status"
fi

child_count="$(printf '%s\n' "$child_reply" | sed -n 's/.*int32 \([0-9][0-9]*\).*/\1/p' | head -n1)"
if [[ -z "$child_count" ]]; then
  warn "ChildCount was not parsed, but the call succeeded."
else
  ok "AT-SPI root ChildCount=$child_count"
fi

section "Exposed Applications"
children_reply="$(dbus-send --bus="$atspi_addr" --dest=org.a11y.atspi.Registry --print-reply /org/a11y/atspi/accessible/root org.a11y.atspi.Accessible.GetChildren 2>&1)"
children_rc=$?
printf '%s\n' "$children_reply"

if [[ $children_rc -ne 0 ]]; then
  fail "Could not enumerate application children from the AT-SPI root."
  printf '\nFinal status: FAIL\n'
  exit "$status"
fi

mapfile -t services < <(printf '%s\n' "$children_reply" | sed -n 's/.*string "\(.*\)"/\1/p')
if [[ ${#services[@]} -eq 0 ]]; then
  warn "No child services were parsed from GetChildren."
else
  ok "Parsed ${#services[@]} exposed application object(s)."
fi

for service in "${services[@]}"; do
  name_reply="$(dbus-send --bus="$atspi_addr" --dest="$service" --print-reply /org/a11y/atspi/accessible/root org.freedesktop.DBus.Properties.Get string:org.a11y.atspi.Accessible string:Name 2>&1)"
  role_reply="$(dbus-send --bus="$atspi_addr" --dest="$service" --print-reply /org/a11y/atspi/accessible/root org.a11y.atspi.Accessible.GetRoleName 2>&1)"
  name="$(printf '%s\n' "$name_reply" | sed -n 's/.*string "\(.*\)"/\1/p' | head -n1)"
  role="$(printf '%s\n' "$role_reply" | sed -n 's/.*string "\(.*\)"/\1/p' | head -n1)"
  printf 'service=%s name=%s role=%s\n' "$service" "${name:-}" "${role:-}"
done

if [[ $status -eq 0 ]]; then
  printf '\nFinal status: OK\n'
else
  printf '\nFinal status: FAIL\n'
fi

exit "$status"
