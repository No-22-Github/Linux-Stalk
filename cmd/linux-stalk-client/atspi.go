package main

import (
	"fmt"
	"sort"

	"github.com/godbus/dbus/v5"
)

const (
	sessionBusObjectPath = dbus.ObjectPath("/org/a11y/bus")
	sessionBusService    = "org.a11y.Bus"

	atspiRegistryService = "org.a11y.atspi.Registry"
	atspiRootPath        = dbus.ObjectPath("/org/a11y/atspi/accessible/root")

	ifaceA11yBus        = "org.a11y.Bus"
	ifaceRegistry       = "org.a11y.atspi.Registry"
	ifaceAccessible     = "org.a11y.atspi.Accessible"
	ifaceDBusProperties = "org.freedesktop.DBus.Properties"
)

type ObjectRef struct {
	Service string
	Path    dbus.ObjectPath
}

type ApplicationInfo struct {
	Ref        ObjectRef
	Name       string
	RoleName   string
	Interfaces []string
	States     []string
	ChildCount int32
}

type AccessibleNode struct {
	Ref      ObjectRef
	Name     string
	RoleName string
	States   []string
}

type FocusInfo struct {
	Application ApplicationInfo
	Object      *AccessibleNode
}

type atspiProbe struct {
	sessionConn *dbus.Conn
	atspiConn   *dbus.Conn
	busAddress  string
}

func newATSPIProbe() (*atspiProbe, error) {
	sessionAddress := getenv("DBUS_SESSION_BUS_ADDRESS")
	if sessionAddress == "" {
		return nil, errNoSessionBus
	}

	sessionConn, err := dbus.Connect(sessionAddress)
	if err != nil {
		return nil, fmt.Errorf("connect session bus: %w", err)
	}

	busAddress, err := getATSPIBusAddress(sessionConn)
	if err != nil {
		sessionConn.Close()
		return nil, err
	}

	atspiConn, err := dbus.Connect(busAddress)
	if err != nil {
		sessionConn.Close()
		return nil, fmt.Errorf("connect at-spi bus: %w", err)
	}

	return &atspiProbe{
		sessionConn: sessionConn,
		atspiConn:   atspiConn,
		busAddress:  busAddress,
	}, nil
}

func (p *atspiProbe) Close() {
	if p.atspiConn != nil {
		p.atspiConn.Close()
	}
	if p.sessionConn != nil {
		p.sessionConn.Close()
	}
}

func (p *atspiProbe) BusAddress() string {
	return p.busAddress
}

func getATSPIBusAddress(conn *dbus.Conn) (string, error) {
	obj := conn.Object(sessionBusService, sessionBusObjectPath)

	var address string
	if err := obj.Call(ifaceA11yBus+".GetAddress", 0).Store(&address); err != nil {
		return "", fmt.Errorf("org.a11y.Bus.GetAddress: %w", err)
	}

	return address, nil
}

func (p *atspiProbe) ListApplications() ([]ApplicationInfo, error) {
	refs, err := p.getChildren(atspiRegistryService, atspiRootPath)
	if err != nil {
		return nil, fmt.Errorf("list root applications: %w", err)
	}

	apps := make([]ApplicationInfo, 0, len(refs))
	for _, ref := range refs {
		name, _ := p.getStringProperty(ref.Service, ref.Path, ifaceAccessible, "Name")
		childCount, _ := p.getInt32Property(ref.Service, ref.Path, ifaceAccessible, "ChildCount")
		roleName, _ := p.getRoleName(ref.Service, ref.Path)
		ifaces, _ := p.getInterfaces(ref.Service, ref.Path)
		states, _ := p.getStates(ref.Service, ref.Path)

		apps = append(apps, ApplicationInfo{
			Ref:        ref,
			Name:       name,
			RoleName:   roleName,
			Interfaces: dedupeAndSort(ifaces),
			States:     dedupeAndSort(states),
			ChildCount: childCount,
		})
	}

	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Name == apps[j].Name {
			return apps[i].Ref.Service < apps[j].Ref.Service
		}
		return apps[i].Name < apps[j].Name
	})

	return apps, nil
}

func (p *atspiProbe) FindFocusedApplication(apps []ApplicationInfo) (*FocusInfo, error) {
	for _, app := range apps {
		node, err := p.findFocusedNode(app.Ref, 0, 48)
		if err != nil {
			return nil, err
		}
		if node != nil {
			appCopy := app
			return &FocusInfo{
				Application: appCopy,
				Object:      node,
			}, nil
		}
	}

	for _, app := range apps {
		if hasState(app.States, "active") || hasState(app.States, "focused") {
			appCopy := app
			return &FocusInfo{Application: appCopy}, nil
		}
	}

	return nil, nil
}

func (p *atspiProbe) findFocusedNode(ref ObjectRef, depth int, limit int) (*AccessibleNode, error) {
	if depth > limit {
		return nil, nil
	}

	name, _ := p.getStringProperty(ref.Service, ref.Path, ifaceAccessible, "Name")
	roleName, _ := p.getRoleName(ref.Service, ref.Path)
	states, _ := p.getStates(ref.Service, ref.Path)

	if hasState(states, "focused") || hasState(states, "active") {
		return &AccessibleNode{
			Ref:      ref,
			Name:     name,
			RoleName: roleName,
			States:   dedupeAndSort(states),
		}, nil
	}

	children, err := p.getChildren(ref.Service, ref.Path)
	if err != nil {
		return nil, nil
	}

	for _, child := range children {
		node, err := p.findFocusedNode(child, depth+1, limit)
		if err != nil {
			return nil, err
		}
		if node != nil {
			return node, nil
		}
	}

	return nil, nil
}

func (p *atspiProbe) getChildren(service string, path dbus.ObjectPath) ([]ObjectRef, error) {
	var raw []struct {
		Service string
		Path    dbus.ObjectPath
	}

	obj := p.atspiConn.Object(service, path)
	if err := obj.Call(ifaceAccessible+".GetChildren", 0).Store(&raw); err != nil {
		return nil, err
	}

	refs := make([]ObjectRef, 0, len(raw))
	for _, item := range raw {
		if item.Service == "" || !item.Path.IsValid() {
			continue
		}
		refs = append(refs, ObjectRef{
			Service: item.Service,
			Path:    item.Path,
		})
	}

	return refs, nil
}

func (p *atspiProbe) getRoleName(service string, path dbus.ObjectPath) (string, error) {
	obj := p.atspiConn.Object(service, path)
	var roleName string
	if err := obj.Call(ifaceAccessible+".GetRoleName", 0).Store(&roleName); err != nil {
		return "", err
	}
	return roleName, nil
}

func (p *atspiProbe) getInterfaces(service string, path dbus.ObjectPath) ([]string, error) {
	obj := p.atspiConn.Object(service, path)
	var ifaces []string
	if err := obj.Call(ifaceAccessible+".GetInterfaces", 0).Store(&ifaces); err != nil {
		return nil, err
	}
	return ifaces, nil
}

func (p *atspiProbe) getStates(service string, path dbus.ObjectPath) ([]string, error) {
	obj := p.atspiConn.Object(service, path)
	var words []uint32
	if err := obj.Call(ifaceAccessible+".GetState", 0).Store(&words); err != nil {
		return nil, err
	}

	return decodeStates(words), nil
}

func (p *atspiProbe) getStringProperty(service string, path dbus.ObjectPath, iface string, prop string) (string, error) {
	value, err := p.getProperty(service, path, iface, prop)
	if err != nil {
		return "", err
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s.%s is %T, want string", iface, prop, value)
	}
	return str, nil
}

func (p *atspiProbe) getInt32Property(service string, path dbus.ObjectPath, iface string, prop string) (int32, error) {
	value, err := p.getProperty(service, path, iface, prop)
	if err != nil {
		return 0, err
	}

	number, ok := value.(int32)
	if !ok {
		return 0, fmt.Errorf("%s.%s is %T, want int32", iface, prop, value)
	}
	return number, nil
}

func (p *atspiProbe) getProperty(service string, path dbus.ObjectPath, iface string, prop string) (any, error) {
	obj := p.atspiConn.Object(service, path)

	var value dbus.Variant
	if err := obj.Call(ifaceDBusProperties+".Get", 0, iface, prop).Store(&value); err != nil {
		return nil, err
	}

	return value.Value(), nil
}

func hasState(states []string, want string) bool {
	for _, state := range states {
		if state == want {
			return true
		}
	}
	return false
}

func dedupeAndSort(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}

	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func atspiStateName(code uint32) (string, bool) {
	name, ok := atspiStates[code]
	return name, ok
}

func decodeStates(words []uint32) []string {
	if len(words) == 0 {
		return nil
	}

	states := make([]string, 0, 16)
	for wordIndex, word := range words {
		if word == 0 {
			continue
		}

		for bit := uint32(0); bit < 32; bit++ {
			if word&(1<<bit) == 0 {
				continue
			}

			code := uint32(wordIndex)*32 + bit
			if name, ok := atspiStateName(code); ok {
				states = append(states, name)
				continue
			}
			states = append(states, fmt.Sprintf("unknown(%d)", code))
		}
	}

	return states
}

var atspiStates = map[uint32]string{
	0:  "invalid",
	1:  "active",
	2:  "armed",
	3:  "busy",
	4:  "checked",
	5:  "collapsed",
	6:  "defunct",
	7:  "editable",
	8:  "enabled",
	9:  "expandable",
	10: "expanded",
	11: "focusable",
	12: "focused",
	13: "horizontal",
	14: "iconified",
	15: "modal",
	16: "multi_line",
	17: "multiselectable",
	18: "opaque",
	19: "pressed",
	20: "resizable",
	21: "selectable",
	22: "selected",
	23: "sensitive",
	24: "showing",
	25: "single_line",
	26: "stale",
	27: "transient",
	28: "vertical",
	29: "visible",
	30: "manages_descendants",
	31: "indeterminate",
	32: "required",
	33: "truncated",
	34: "animated",
	35: "invalid_entry",
	36: "supports_autocompletion",
	37: "selectable_text",
	38: "is_default",
	39: "visited",
	40: "checkable",
	41: "has_popup",
	42: "read_only",
}
