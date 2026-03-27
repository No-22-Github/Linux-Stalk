package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	dbusService        = "org.freedesktop.DBus"
	dbusPath           = dbus.ObjectPath("/org/freedesktop/DBus")
	dbusIface          = "org.freedesktop.DBus"
	mprisObjectPath    = dbus.ObjectPath("/org/mpris/MediaPlayer2")
	mprisRootIface     = "org.mpris.MediaPlayer2"
	mprisPlayerIface   = "org.mpris.MediaPlayer2.Player"
	mprisServicePrefix = "org.mpris.MediaPlayer2."
)

func enrichSessionInfoFromDBus(info *SystemInfo) {
	sessionAddress := getenv("DBUS_SESSION_BUS_ADDRESS")
	if sessionAddress == "" {
		return
	}

	conn, err := dbus.Connect(sessionAddress)
	if err != nil {
		return
	}
	defer conn.Close()

	if media, err := readMPRISMedia(conn); err == nil {
		info.MediaSessions = media
	}
}

func readMPRISMedia(conn *dbus.Conn) ([]string, error) {
	obj := conn.Object(dbusService, dbusPath)

	var names []string
	if err := obj.Call(dbusIface+".ListNames", 0).Store(&names); err != nil {
		return nil, err
	}

	merged := make(map[string]mediaSession)
	for _, name := range names {
		if !strings.HasPrefix(name, mprisServicePrefix) {
			continue
		}

		session, err := readMPRISPlayer(conn, name)
		if err != nil {
			continue
		}
		if session.summary == "" {
			continue
		}

		key := session.dedupeKey()
		if existing, ok := merged[key]; ok {
			merged[key] = betterMediaSession(existing, session)
			continue
		}
		merged[key] = session
	}

	sessions := make([]mediaSession, 0, len(merged))
	for _, session := range merged {
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].rank != sessions[j].rank {
			return sessions[i].rank < sessions[j].rank
		}
		return sessions[i].summary < sessions[j].summary
	})

	out := make([]string, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, session.summary)
	}
	return out, nil
}

type mediaSession struct {
	appKey   string
	status   string
	trackKey string
	summary  string
	rank     int
}

func readMPRISPlayer(conn *dbus.Conn, service string) (mediaSession, error) {
	identity, _ := getStringProperty(conn, service, mprisObjectPath, mprisRootIface, "Identity")
	status, _ := getStringProperty(conn, service, mprisObjectPath, mprisPlayerIface, "PlaybackStatus")
	metadata, _ := getVariantMapProperty(conn, service, mprisObjectPath, mprisPlayerIface, "Metadata")

	serviceName := strings.TrimPrefix(service, mprisServicePrefix)
	player := firstNonEmpty(identity, serviceName)
	title := variantMapString(metadata, "xesam:title")
	artists := variantMapStrings(metadata, "xesam:artist")
	album := variantMapString(metadata, "xesam:album")
	url := variantMapString(metadata, "xesam:url")

	var parts []string
	if player != "" {
		parts = append(parts, player)
	}
	if status != "" {
		parts = append(parts, status)
	}

	track := strings.TrimSpace(strings.Join([]string{
		title,
		strings.Join(artists, ", "),
	}, " - "))
	track = strings.Trim(track, " -")
	if track != "" {
		parts = append(parts, track)
	} else if album != "" {
		parts = append(parts, album)
	} else if url != "" {
		parts = append(parts, url)
	}

	if len(parts) == 0 {
		return mediaSession{}, nil
	}

	return mediaSession{
		appKey:   normalizeMediaApp(player, serviceName),
		status:   status,
		trackKey: normalizeTrackKey(title, artists, album, url),
		summary:  strings.Join(parts, " | "),
		rank:     mediaRank(status),
	}, nil
}

func (m mediaSession) dedupeKey() string {
	return strings.Join([]string{
		m.appKey,
		m.status,
		m.trackKey,
	}, "\x1f")
}

func betterMediaSession(a mediaSession, b mediaSession) mediaSession {
	if len(b.summary) > len(a.summary) {
		return b
	}
	return a
}

func mediaRank(status string) int {
	switch status {
	case "Playing":
		return 0
	case "Paused":
		return 1
	case "Stopped":
		return 2
	default:
		return 3
	}
}

func getVariantMapProperty(conn *dbus.Conn, service string, path dbus.ObjectPath, iface string, prop string) (map[string]dbus.Variant, error) {
	value, err := getDBusProperty(conn, service, path, iface, prop)
	if err != nil {
		return nil, err
	}

	out, ok := value.(map[string]dbus.Variant)
	if !ok {
		return nil, fmt.Errorf("%s.%s is %T, want map[string]dbus.Variant", iface, prop, value)
	}
	return out, nil
}

func variantMapString(values map[string]dbus.Variant, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	return variantString(value)
}

func variantMapStrings(values map[string]dbus.Variant, key string) []string {
	if values == nil {
		return nil
	}
	value, ok := values[key]
	if !ok {
		return nil
	}

	switch v := value.Value().(type) {
	case []string:
		return compactStrings(v)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeMediaApp(identity string, serviceName string) string {
	base := strings.ToLower(strings.TrimSpace(firstNonEmpty(identity, serviceName)))
	base = strings.TrimPrefix(base, "org.mpris.mediaplayer2.")
	base = strings.TrimPrefix(base, "google ")
	return base
}

func normalizeTrackKey(title string, artists []string, album string, url string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	album = strings.ToLower(strings.TrimSpace(album))
	url = strings.ToLower(strings.TrimSpace(url))

	artistKey := strings.ToLower(strings.Join(compactStrings(artists), ","))
	switch {
	case title != "" || artistKey != "":
		return title + "\x1f" + artistKey
	case url != "":
		return url
	case album != "":
		return album
	default:
		return ""
	}
}
