package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	systemIfaceProps       = "org.freedesktop.DBus.Properties"
	systemBusRoot          = "/"
	upowerService          = "org.freedesktop.UPower"
	upowerRootPath         = dbus.ObjectPath("/org/freedesktop/UPower")
	upowerDisplayDevice    = dbus.ObjectPath("/org/freedesktop/UPower/devices/DisplayDevice")
	upowerIfaceRoot        = "org.freedesktop.UPower"
	upowerIfaceDevice      = "org.freedesktop.UPower.Device"
	nmService              = "org.freedesktop.NetworkManager"
	nmRootPath             = dbus.ObjectPath("/org/freedesktop/NetworkManager")
	nmIfaceRoot            = "org.freedesktop.NetworkManager"
	nmIfaceActiveConn      = "org.freedesktop.NetworkManager.Connection.Active"
	nmIfaceAccessPoint     = "org.freedesktop.NetworkManager.AccessPoint"
	bluezService           = "org.bluez"
	dbusObjectManagerIface = "org.freedesktop.DBus.ObjectManager"
	bluezIfaceDevice       = "org.bluez.Device1"
)

func enrichSystemInfoFromDBus(info *SystemInfo) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return
	}

	if summary, err := readPowerSummary(conn); err == nil {
		info.PowerSummary = summary
	}
	if ssid, err := readWiFiSSID(conn); err == nil {
		info.WiFiSSID = ssid
	}
	if devices, err := readConnectedBluetoothDevices(conn); err == nil {
		info.BluetoothDevices = devices
	}
}

func readPowerSummary(conn *dbus.Conn) (string, error) {
	onBattery, err := getBoolProperty(conn, upowerService, upowerRootPath, upowerIfaceRoot, "OnBattery")
	if err != nil {
		return "", err
	}

	isPresent, err := getBoolProperty(conn, upowerService, upowerDisplayDevice, upowerIfaceDevice, "IsPresent")
	if err != nil {
		return "", err
	}

	if !isPresent {
		if onBattery {
			return "on battery", nil
		}
		return "AC power (no battery present)", nil
	}

	percentage, _ := getFloat64Property(conn, upowerService, upowerDisplayDevice, upowerIfaceDevice, "Percentage")
	stateCode, _ := getUint32Property(conn, upowerService, upowerDisplayDevice, upowerIfaceDevice, "State")
	state := upowerStateName(stateCode)

	summary := strings.TrimSpace(fmt.Sprintf("%s %.0f%%", state, percentage))
	if summary == "" {
		if onBattery {
			return "on battery", nil
		}
		return "AC power", nil
	}
	return summary, nil
}

func readWiFiSSID(conn *dbus.Conn) (string, error) {
	paths, err := getObjectPathSliceProperty(conn, nmService, nmRootPath, nmIfaceRoot, "ActiveConnections")
	if err != nil {
		return "", err
	}

	for _, path := range paths {
		connType, err := getStringProperty(conn, nmService, path, nmIfaceActiveConn, "Type")
		if err != nil || connType != "802-11-wireless" {
			continue
		}

		apPath, _ := getObjectPathProperty(conn, nmService, path, nmIfaceActiveConn, "SpecificObject")
		if apPath == "/" || !apPath.IsValid() {
			continue
		}

		ssidBytes, err := getByteSliceProperty(conn, nmService, apPath, nmIfaceAccessPoint, "Ssid")
		if err != nil {
			continue
		}

		ssid := strings.TrimSpace(string(ssidBytes))
		if ssid != "" {
			return ssid, nil
		}
	}

	return "not connected", nil
}

func readConnectedBluetoothDevices(conn *dbus.Conn) ([]string, error) {
	obj := conn.Object(bluezService, dbus.ObjectPath(systemBusRoot))

	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := obj.Call(dbusObjectManagerIface+".GetManagedObjects", 0).Store(&objects); err != nil {
		return nil, err
	}

	var devices []string
	for _, ifaces := range objects {
		props, ok := ifaces[bluezIfaceDevice]
		if !ok {
			continue
		}

		connected, ok := variantBool(props["Connected"])
		if !ok || !connected {
			continue
		}

		name := variantString(props["Alias"])
		if name == "" {
			name = variantString(props["Name"])
		}
		if name == "" {
			name = variantString(props["Address"])
		}
		if name == "" {
			continue
		}

		devices = append(devices, name)
	}

	sort.Strings(devices)
	return devices, nil
}

func getStringProperty(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) (string, error) {
	value, err := getDBusProperty(conn, service, path, iface, prop)
	if err != nil {
		return "", err
	}
	out, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s.%s is %T, want string", iface, prop, value)
	}
	return out, nil
}

func getBoolProperty(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) (bool, error) {
	value, err := getDBusProperty(conn, service, path, iface, prop)
	if err != nil {
		return false, err
	}
	out, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s.%s is %T, want bool", iface, prop, value)
	}
	return out, nil
}

func getFloat64Property(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) (float64, error) {
	value, err := getDBusProperty(conn, service, path, iface, prop)
	if err != nil {
		return 0, err
	}
	out, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("%s.%s is %T, want float64", iface, prop, value)
	}
	return out, nil
}

func getUint32Property(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) (uint32, error) {
	value, err := getDBusProperty(conn, service, path, iface, prop)
	if err != nil {
		return 0, err
	}
	out, ok := value.(uint32)
	if !ok {
		return 0, fmt.Errorf("%s.%s is %T, want uint32", iface, prop, value)
	}
	return out, nil
}

func getObjectPathProperty(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) (dbus.ObjectPath, error) {
	value, err := getDBusProperty(conn, service, path, iface, prop)
	if err != nil {
		return "", err
	}
	out, ok := value.(dbus.ObjectPath)
	if !ok {
		return "", fmt.Errorf("%s.%s is %T, want object path", iface, prop, value)
	}
	return out, nil
}

func getObjectPathSliceProperty(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) ([]dbus.ObjectPath, error) {
	value, err := getDBusProperty(conn, service, path, iface, prop)
	if err != nil {
		return nil, err
	}
	out, ok := value.([]dbus.ObjectPath)
	if !ok {
		return nil, fmt.Errorf("%s.%s is %T, want []object path", iface, prop, value)
	}
	return out, nil
}

func getByteSliceProperty(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) ([]byte, error) {
	value, err := getDBusProperty(conn, service, path, iface, prop)
	if err != nil {
		return nil, err
	}
	out, ok := value.([]byte)
	if !ok {
		return nil, fmt.Errorf("%s.%s is %T, want []byte", iface, prop, value)
	}
	return out, nil
}

func getDBusProperty(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) (interface{}, error) {
	obj := conn.Object(service, path)

	var value dbus.Variant
	if err := obj.Call(systemIfaceProps+".Get", 0, iface, prop).Store(&value); err != nil {
		return nil, err
	}

	return value.Value(), nil
}

func variantString(v dbus.Variant) string {
	value, ok := v.Value().(string)
	if !ok {
		return ""
	}
	return value
}

func variantBool(v dbus.Variant) (bool, bool) {
	value, ok := v.Value().(bool)
	return value, ok
}

func upowerStateName(code uint32) string {
	switch code {
	case 1:
		return "charging"
	case 2:
		return "discharging"
	case 3:
		return "empty"
	case 4:
		return "fully charged"
	case 5:
		return "pending charge"
	case 6:
		return "pending discharge"
	default:
		return "unknown"
	}
}
