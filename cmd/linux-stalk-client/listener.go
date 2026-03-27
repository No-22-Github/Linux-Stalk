package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	atspiRegistryPath      = dbus.ObjectPath("/org/a11y/atspi/registry")
	ifaceEventFocus        = "org.a11y.atspi.Event.Focus"
	ifaceEventObject       = "org.a11y.atspi.Event.Object"
	ifaceEventWindow       = "org.a11y.atspi.Event.Window"
	ifaceEventMouse        = "org.a11y.atspi.Event.Mouse"
	ifaceEventKeyboard     = "org.a11y.atspi.Event.Keyboard"
	ifaceEventScreenReader = "org.a11y.atspi.Event.ScreenReader"
	focusEventName         = "focus:"
	focusedStateEvent      = "object:state-changed:focused"
)

var allEventRegistrations = []string{
	"focus:",
	"object:",
	"window:",
	"mouse:",
	"screen-reader:",
}

var allEventInterfaces = []string{
	ifaceEventFocus,
	ifaceEventObject,
	ifaceEventWindow,
	ifaceEventMouse,
	ifaceEventKeyboard,
	ifaceEventScreenReader,
}

type FocusEvent struct {
	Timestamp       time.Time
	Signal          string
	Sender          string
	Path            dbus.ObjectPath
	Interface       string
	Member          string
	AccessibleName  string
	AccessibleRole  string
	ApplicationName string
	BodySummary     string
}

func (p *atspiProbe) ListenAllEvents(ctx context.Context, onEvent func(FocusEvent)) error {
	return p.listenEvents(ctx, allEventRegistrations, allEventInterfaces, func(sig *dbus.Signal) bool {
		return strings.HasPrefix(sig.Name, "org.a11y.atspi.Event.")
	}, onEvent)
}

func (p *atspiProbe) ListenFocusEvents(ctx context.Context, onEvent func(FocusEvent)) error {
	return p.listenEvents(ctx, []string{focusEventName, focusedStateEvent}, []string{ifaceEventFocus, ifaceEventObject}, isFocusSignal, onEvent)
}

func (p *atspiProbe) listenEvents(ctx context.Context, registrations []string, interfaces []string, keep func(*dbus.Signal) bool, onEvent func(FocusEvent)) error {
	for _, event := range registrations {
		if err := p.registerEvent(event); err != nil {
			return err
		}
		defer p.deregisterEvent(event)
	}

	for _, iface := range interfaces {
		if err := p.atspiConn.AddMatchSignal(dbus.WithMatchInterface(iface)); err != nil {
			return fmt.Errorf("add %s match: %w", iface, err)
		}
		defer p.atspiConn.RemoveMatchSignal(dbus.WithMatchInterface(iface))
	}

	signals := make(chan *dbus.Signal, 64)
	p.atspiConn.Signal(signals)
	defer p.atspiConn.RemoveSignal(signals)

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
				return err
			}
			return nil
		case sig, ok := <-signals:
			if !ok || sig == nil {
				continue
			}
			if !keep(sig) {
				continue
			}

			event := p.describeFocusSignal(sig)
			onEvent(event)
		}
	}
}

func (p *atspiProbe) registerEvent(event string) error {
	obj := p.atspiConn.Object(atspiRegistryService, atspiRegistryPath)
	if err := obj.Call(ifaceRegistry+".RegisterEvent", 0, event, []string{}, "").Err; err != nil {
		return fmt.Errorf("register %s: %w", event, err)
	}
	return nil
}

func (p *atspiProbe) deregisterEvent(event string) {
	obj := p.atspiConn.Object(atspiRegistryService, atspiRegistryPath)
	_ = obj.Call(ifaceRegistry+".DeregisterEvent", 0, event).Err
}

func isFocusSignal(sig *dbus.Signal) bool {
	switch sig.Name {
	case ifaceEventFocus + ".Focus", ifaceEventFocus + ".focus":
		return true
	}

	if strings.HasPrefix(sig.Name, ifaceEventObject+".") && strings.Contains(strings.ToLower(sig.Name), "focused") {
		return true
	}

	if strings.HasPrefix(sig.Name, ifaceEventFocus+".") || strings.HasPrefix(sig.Name, ifaceEventObject+".") {
		return true
	}

	return false
}

func (p *atspiProbe) describeFocusSignal(sig *dbus.Signal) FocusEvent {
	event := FocusEvent{
		Timestamp:   time.Now(),
		Signal:      sig.Name,
		Sender:      sig.Sender,
		Path:        sig.Path,
		Interface:   signalInterface(sig.Name),
		Member:      signalMember(sig.Name),
		BodySummary: formatSignalBody(sig.Body),
	}

	if sig.Sender != "" && sig.Path.IsValid() {
		name, _ := p.getStringProperty(sig.Sender, sig.Path, ifaceAccessible, "Name")
		role, _ := p.getRoleName(sig.Sender, sig.Path)
		event.AccessibleName = name
		event.AccessibleRole = role

		if appRef, err := p.getApplication(sig.Sender, sig.Path); err == nil {
			appName, _ := p.getStringProperty(appRef.Service, appRef.Path, ifaceAccessible, "Name")
			event.ApplicationName = appName
		}
	}

	return event
}

func (p *atspiProbe) getApplication(service string, path dbus.ObjectPath) (ObjectRef, error) {
	obj := p.atspiConn.Object(service, path)

	var raw struct {
		Service string
		Path    dbus.ObjectPath
	}
	if err := obj.Call(ifaceAccessible+".GetApplication", 0).Store(&raw); err != nil {
		return ObjectRef{}, err
	}

	return ObjectRef{
		Service: raw.Service,
		Path:    raw.Path,
	}, nil
}

func signalInterface(name string) string {
	idx := strings.LastIndex(name, ".")
	if idx <= 0 {
		return name
	}
	return name[:idx]
}

func signalMember(name string) string {
	idx := strings.LastIndex(name, ".")
	if idx < 0 || idx == len(name)-1 {
		return ""
	}
	return name[idx+1:]
}

func formatSignalBody(body []interface{}) string {
	if len(body) == 0 {
		return "(empty)"
	}

	parts := make([]string, 0, len(body))
	for _, item := range body {
		parts = append(parts, fmt.Sprintf("%T=%v", item, item))
	}
	return strings.Join(parts, ", ")
}
