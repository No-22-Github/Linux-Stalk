package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

func main() {
	cfg := parseConfig()

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type config struct {
	listenFocus    bool
	listenAll      bool
	listenRelevant bool
	push           bool
	snapshot       bool
	filterLog      string
	configPath     string
	timeout        time.Duration
}

func parseConfig() config {
	cfg := config{}
	flag.BoolVar(&cfg.listenFocus, "listen-focus", false, "listen for AT-SPI focus-related events")
	flag.BoolVar(&cfg.listenAll, "listen-all", false, "listen for all AT-SPI event categories")
	flag.BoolVar(&cfg.listenRelevant, "listen-relevant", false, "listen only for filtered foreground and focused-control AT-SPI events")
	flag.BoolVar(&cfg.push, "push", false, "listen for window switches and push state snapshots to the configured server")
	flag.BoolVar(&cfg.snapshot, "snapshot", false, "print a one-shot system and accessibility snapshot instead of listening")
	flag.StringVar(&cfg.filterLog, "filter-log", "", "filter an existing atspi listener log file and print only relevant events")
	flag.StringVar(&cfg.configPath, "config", "configs/client.json", "path to the client config file")
	flag.DurationVar(&cfg.timeout, "timeout", 0, "optional timeout for listen mode, for example 15s")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  linux-stalk [--listen-relevant] [--timeout 15s]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  linux-stalk --push [--config configs/client.json]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  linux-stalk --listen-all [--timeout 30s]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  linux-stalk --listen-focus [--timeout 30s]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  linux-stalk --snapshot\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  linux-stalk --filter-log atspi-all.txt\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if cfg.filterLog == "" && !cfg.listenFocus && !cfg.listenAll && !cfg.listenRelevant && !cfg.push && !cfg.snapshot {
		cfg.listenRelevant = true
	}

	return cfg
}

func run(cfg config) error {
	if cfg.filterLog != "" {
		return filterLogFile(cfg.filterLog)
	}

	sysInfo, err := collectSystemInfo()
	if err != nil {
		return err
	}

	probe, err := newATSPIProbe()
	if err != nil {
		return err
	}
	defer probe.Close()
	sysInfo.ATSPIBusAddress = probe.BusAddress()

	if cfg.push {
		return runPush(probe, cfg)
	}

	if cfg.snapshot {
		apps, err := probe.ListApplications()
		if err != nil {
			return err
		}

		focus, err := probe.FindFocusedApplication(apps)
		if err != nil {
			return err
		}

		printSystemInfo(sysInfo)
		fmt.Println()
		printApplications(apps)
		fmt.Println()
		printFocus(focus)
		return nil
	}

	if cfg.listenRelevant {
		printSystemInfo(sysInfo)
		fmt.Println()
		return listenRelevant(probe, cfg.timeout)
	}

	if cfg.listenAll {
		printSystemInfo(sysInfo)
		fmt.Println()
		return listenAll(probe, cfg.timeout)
	}

	if cfg.listenFocus {
		printSystemInfo(sysInfo)
		fmt.Println()
		return listenFocus(probe, cfg.timeout)
	}

	return fmt.Errorf("no mode selected")
}

func listenRelevant(probe *atspiProbe, timeout time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	fmt.Println("== Relevant AT-SPI Event Listener ==")
	fmt.Println("Kept events: Window.Activate, StateChanged(focused|active=1) on meaningful roles, noisy shell events dropped")
	fmt.Println("Listening. Stop with Ctrl+C.")

	lastPrinted := ""
	return probe.ListenAllEvents(ctx, func(event FocusEvent) {
		if !isRelevantEvent(event) {
			return
		}
		if fp := eventFingerprint(event); fp == lastPrinted {
			return
		} else {
			lastPrinted = fp
		}
		fmt.Printf("[%s] %s\n", event.Timestamp.Format(time.RFC3339), event.Signal)
		fmt.Printf("  Sender: %s\n", noneIfEmpty(event.Sender))
		fmt.Printf("  Path: %s\n", event.Path)
		fmt.Printf("  Interface: %s\n", noneIfEmpty(event.Interface))
		fmt.Printf("  Member: %s\n", noneIfEmpty(event.Member))
		fmt.Printf("  Accessible Name: %s\n", noneIfEmpty(event.AccessibleName))
		fmt.Printf("  Accessible Role: %s\n", noneIfEmpty(event.AccessibleRole))
		fmt.Printf("  Application: %s\n", noneIfEmpty(event.ApplicationName))
		fmt.Printf("  Body: %s\n", event.BodySummary)
		fmt.Println()
	})
}

func listenAll(probe *atspiProbe, timeout time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	fmt.Println("== AT-SPI Event Listener ==")
	fmt.Println("Registered events: focus:, object:, window:, mouse:, screen-reader:")
	fmt.Println("Listening. Stop with Ctrl+C.")

	return probe.ListenAllEvents(ctx, func(event FocusEvent) {
		fmt.Printf("[%s] %s\n", event.Timestamp.Format(time.RFC3339), event.Signal)
		fmt.Printf("  Sender: %s\n", noneIfEmpty(event.Sender))
		fmt.Printf("  Path: %s\n", event.Path)
		fmt.Printf("  Interface: %s\n", noneIfEmpty(event.Interface))
		fmt.Printf("  Member: %s\n", noneIfEmpty(event.Member))
		fmt.Printf("  Accessible Name: %s\n", noneIfEmpty(event.AccessibleName))
		fmt.Printf("  Accessible Role: %s\n", noneIfEmpty(event.AccessibleRole))
		fmt.Printf("  Application: %s\n", noneIfEmpty(event.ApplicationName))
		fmt.Printf("  Body: %s\n", event.BodySummary)
		fmt.Println()
	})
}

func listenFocus(probe *atspiProbe, timeout time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	fmt.Println("== Focus Event Listener ==")
	fmt.Println("Registered events: focus:, object:state-changed:focused")
	fmt.Println("Listening. Stop with Ctrl+C.")

	return probe.ListenFocusEvents(ctx, func(event FocusEvent) {
		fmt.Printf("[%s] %s\n", event.Timestamp.Format(time.RFC3339), event.Signal)
		fmt.Printf("  Sender: %s\n", noneIfEmpty(event.Sender))
		fmt.Printf("  Path: %s\n", event.Path)
		fmt.Printf("  Interface: %s\n", noneIfEmpty(event.Interface))
		fmt.Printf("  Member: %s\n", noneIfEmpty(event.Member))
		fmt.Printf("  Accessible Name: %s\n", noneIfEmpty(event.AccessibleName))
		fmt.Printf("  Accessible Role: %s\n", noneIfEmpty(event.AccessibleRole))
		fmt.Printf("  Application: %s\n", noneIfEmpty(event.ApplicationName))
		fmt.Printf("  Body: %s\n", event.BodySummary)
		fmt.Println()
	})
}

func printSystemInfo(info SystemInfo) {
	fmt.Println("== System Info ==")
	fmt.Printf("Hostname: %s\n", noneIfEmpty(info.Hostname))
	fmt.Printf("Pretty OS: %s\n", noneIfEmpty(info.PrettyOS))
	fmt.Printf("Kernel: %s\n", noneIfEmpty(info.Kernel))
	fmt.Printf("Architecture: %s\n", noneIfEmpty(info.Architecture))
	fmt.Printf("Desktop Session: %s\n", noneIfEmpty(info.DesktopSession))
	fmt.Printf("Current Desktop: %s\n", noneIfEmpty(info.CurrentDesktop))
	fmt.Printf("Session Type: %s\n", noneIfEmpty(info.SessionType))
	fmt.Printf("Display: %s\n", noneIfEmpty(info.Display))
	fmt.Printf("Wayland Display: %s\n", noneIfEmpty(info.WaylandDisplay))
	fmt.Printf("DBus Session Bus: %s\n", noneIfEmpty(info.SessionBusAddress))
	fmt.Printf("AT-SPI Bus: %s\n", noneIfEmpty(info.ATSPIBusAddress))
	fmt.Printf("Power: %s\n", noneIfEmpty(info.PowerSummary))
	fmt.Printf("Wi-Fi SSID: %s\n", noneIfEmpty(info.WiFiSSID))
	fmt.Printf("Bluetooth Devices: %s\n", joinOrNone(info.BluetoothDevices))
	fmt.Printf("Media: %s\n", joinOrNone(info.MediaSessions))
}

func printApplications(apps []ApplicationInfo) {
	fmt.Println("== Accessibility Applications ==")
	if len(apps) == 0 {
		fmt.Println("No applications exposed via AT-SPI.")
		return
	}

	for idx, app := range apps {
		fmt.Printf("%d. %s\n", idx+1, noneIfEmpty(app.Name))
		fmt.Printf("   Service: %s\n", app.Ref.Service)
		fmt.Printf("   Path: %s\n", app.Ref.Path)
		fmt.Printf("   Role: %s\n", noneIfEmpty(app.RoleName))
		fmt.Printf("   Child Count: %d\n", app.ChildCount)
		fmt.Printf("   Interfaces: %s\n", joinOrNone(app.Interfaces))
		fmt.Printf("   States: %s\n", joinOrNone(app.States))
	}
}

func printFocus(focus *FocusInfo) {
	fmt.Println("== Focused Application ==")
	if focus == nil {
		fmt.Println("No focused or active accessible object was found.")
		return
	}

	fmt.Printf("Application: %s\n", noneIfEmpty(focus.Application.Name))
	fmt.Printf("App Service: %s\n", focus.Application.Ref.Service)
	fmt.Printf("App Path: %s\n", focus.Application.Ref.Path)
	fmt.Printf("App States: %s\n", joinOrNone(focus.Application.States))

	if focus.Object == nil {
		fmt.Println("Focused object details: unavailable")
		return
	}

	fmt.Printf("Focused Object Name: %s\n", noneIfEmpty(focus.Object.Name))
	fmt.Printf("Focused Object Role: %s\n", noneIfEmpty(focus.Object.RoleName))
	fmt.Printf("Focused Object Service: %s\n", focus.Object.Ref.Service)
	fmt.Printf("Focused Object Path: %s\n", focus.Object.Ref.Path)
	fmt.Printf("Focused Object States: %s\n", joinOrNone(focus.Object.States))
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}

	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return strings.Join(copyValues, ", ")
}

func noneIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(unknown)"
	}

	return value
}

var errNoSessionBus = errors.New("DBUS_SESSION_BUS_ADDRESS is not set")
