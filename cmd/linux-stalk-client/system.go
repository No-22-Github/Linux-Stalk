package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type SystemInfo struct {
	Hostname          string
	PrettyOS          string
	Kernel            string
	Architecture      string
	DesktopSession    string
	CurrentDesktop    string
	SessionType       string
	Display           string
	WaylandDisplay    string
	SessionBusAddress string
	ATSPIBusAddress   string
	PowerSummary      string
	WiFiSSID          string
	BluetoothDevices  []string
	MediaSessions     []string
}

func collectSystemInfo() (SystemInfo, error) {
	info := SystemInfo{
		Hostname:          hostname(),
		PrettyOS:          readPrettyOS(),
		Kernel:            readKernel(),
		Architecture:      runtime.GOARCH,
		DesktopSession:    getenv("DESKTOP_SESSION"),
		CurrentDesktop:    getenv("XDG_CURRENT_DESKTOP"),
		SessionType:       getenv("XDG_SESSION_TYPE"),
		Display:           getenv("DISPLAY"),
		WaylandDisplay:    getenv("WAYLAND_DISPLAY"),
		SessionBusAddress: getenv("DBUS_SESSION_BUS_ADDRESS"),
	}

	enrichSystemInfoFromDBus(&info)
	enrichSessionInfoFromDBus(&info)

	return info, nil
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}

func getenv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func readPrettyOS() string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "PRETTY_NAME=") {
			continue
		}
		value := strings.TrimPrefix(line, "PRETTY_NAME=")
		return strings.Trim(value, `"`)
	}

	return runtime.GOOS
}

func readKernel() string {
	output, err := exec.Command("uname", "-sr").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(bytes.TrimSpace(output)))
}

func commandString(name string, args ...string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

func runCommand(name string, args ...string) (string, error) {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return "", fmt.Errorf("%s: %s", commandString(name, args...), text)
	}

	return strings.TrimSpace(string(output)), nil
}
