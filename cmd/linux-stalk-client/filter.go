package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
)

type parsedBody struct {
	kind    string
	detail1 string
}

func isRelevantEvent(event FocusEvent) bool {
	app := strings.TrimSpace(event.ApplicationName)
	role := strings.ToLower(strings.TrimSpace(event.AccessibleRole))

	if app == "" || app == "(unknown)" {
		return false
	}
	if isNoisyApp(app) {
		return false
	}

	switch event.Signal {
	case ifaceEventWindow + ".Activate":
		return role == "window" || role == "dialog" || role == "alert dialog" || role == "(unknown)"
	case ifaceEventObject + ".StateChanged":
		body := parseBodySummary(event.BodySummary)
		if body.kind == "focused" && body.detail1 == "1" {
			return isRelevantFocusRole(role)
		}
	}

	return false
}

func isNoisyApp(app string) bool {
	switch app {
	case "gnome-shell", "org.gnome.Shell.Extensions":
		return true
	default:
		return false
	}
}

func isRelevantFocusRole(role string) bool {
	switch role {
	case "dialog", "alert dialog", "terminal", "text", "text box", "button", "toggle button", "row", "list", "list item", "tree grid", "menu item", "tab panel", "application":
		return true
	default:
		return false
	}
}

func parseBodySummary(summary string) parsedBody {
	result := parsedBody{}

	parts := strings.Split(summary, ", ")
	if len(parts) > 0 && strings.HasPrefix(parts[0], "string=") {
		result.kind = strings.TrimPrefix(parts[0], "string=")
	}
	if len(parts) > 1 && strings.HasPrefix(parts[1], "int32=") {
		result.detail1 = strings.TrimPrefix(parts[1], "int32=")
	}

	return result
}

func filterLogFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var block []string
	var lastPrinted string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if len(block) > 0 {
				if event, ok := parseLoggedEvent(block); ok && isRelevantEvent(event) && eventFingerprint(event) != lastPrinted {
					fmt.Println(strings.Join(block, "\n"))
					fmt.Println()
					lastPrinted = eventFingerprint(event)
				}
				block = nil
			}
			continue
		}

		if strings.HasPrefix(line, "[") {
			if len(block) > 0 {
				if event, ok := parseLoggedEvent(block); ok && isRelevantEvent(event) && eventFingerprint(event) != lastPrinted {
					fmt.Println(strings.Join(block, "\n"))
					fmt.Println()
					lastPrinted = eventFingerprint(event)
				}
				block = nil
			}
		}

		if len(block) > 0 || strings.HasPrefix(line, "[") {
			block = append(block, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(block) > 0 {
		if event, ok := parseLoggedEvent(block); ok && isRelevantEvent(event) && eventFingerprint(event) != lastPrinted {
			fmt.Println(strings.Join(block, "\n"))
			fmt.Println()
		}
	}

	return nil
}

func parseLoggedEvent(lines []string) (FocusEvent, bool) {
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "[") {
		return FocusEvent{}, false
	}

	event := FocusEvent{
		Signal: extractSignalLine(lines[0]),
	}

	for _, line := range lines[1:] {
		switch {
		case strings.HasPrefix(line, "  Sender: "):
			event.Sender = strings.TrimPrefix(line, "  Sender: ")
		case strings.HasPrefix(line, "  Path: "):
			event.Path = dbus.ObjectPath(strings.TrimPrefix(line, "  Path: "))
		case strings.HasPrefix(line, "  Interface: "):
			event.Interface = strings.TrimPrefix(line, "  Interface: ")
		case strings.HasPrefix(line, "  Member: "):
			event.Member = strings.TrimPrefix(line, "  Member: ")
		case strings.HasPrefix(line, "  Accessible Name: "):
			event.AccessibleName = strings.TrimPrefix(line, "  Accessible Name: ")
		case strings.HasPrefix(line, "  Accessible Role: "):
			event.AccessibleRole = strings.TrimPrefix(line, "  Accessible Role: ")
		case strings.HasPrefix(line, "  Application: "):
			event.ApplicationName = strings.TrimPrefix(line, "  Application: ")
		case strings.HasPrefix(line, "  Body: "):
			event.BodySummary = strings.TrimPrefix(line, "  Body: ")
		}
	}

	return event, true
}

func extractSignalLine(line string) string {
	end := strings.Index(line, "] ")
	if end < 0 || end+2 >= len(line) {
		return ""
	}
	return line[end+2:]
}

func eventFingerprint(event FocusEvent) string {
	return strings.Join([]string{
		event.Signal,
		event.Sender,
		string(event.Path),
		event.AccessibleName,
		event.AccessibleRole,
		event.ApplicationName,
		event.BodySummary,
	}, "\x1f")
}
