package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"linux-stalk/internal/protocol"
)

type serverConfig struct {
	ListenAddr string   `json:"listen_addr"`
	DBPath     string   `json:"db_path"`
	APIKeys    []string `json:"api_keys"`
}

type eventRow struct {
	ReceivedAt time.Time              `json:"received_at"`
	Payload    protocol.IngestPayload `json:"payload"`
}

type deviceRow struct {
	DeviceID        string `json:"device_id"`
	EventCount      int64  `json:"event_count"`
	LatestEventTime string `json:"latest_event_time"`
	LatestSeenAt    string `json:"latest_seen_at"`
}

func main() {
	configPath := flag.String("config", "configs/server.json", "path to the server config file")
	flag.Parse()

	cfg, err := loadServerConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatalf("init db: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/ingest", ingestHandler(db, cfg))
	mux.HandleFunc("/devices", devicesHandler(db, cfg))
	mux.HandleFunc("/events/latest", latestEventHandler(db, cfg))
	mux.HandleFunc("/events", eventsHandler(db, cfg))

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s using db %s", cfg.ListenAddr, cfg.DBPath)
	log.Fatal(server.ListenAndServe())
}

func loadServerConfig(path string) (serverConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return serverConfig{}, err
	}

	var cfg serverConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return serverConfig{}, err
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "data/linux-stalk.db"
	}
	if len(cfg.APIKeys) == 0 {
		return serverConfig{}, fmt.Errorf("api_keys is required")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return serverConfig{}, err
	}
	return cfg, nil
}

func initDB(db *sql.DB) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`CREATE TABLE IF NOT EXISTS ingested_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			event_time TEXT NOT NULL,
			received_at TEXT NOT NULL,
			trigger TEXT NOT NULL,
			state_hash TEXT NOT NULL,
			payload_json TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_ingested_events_device_time ON ingested_events(device_id, event_time);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func ingestHandler(db *sql.DB, cfg serverConfig) http.HandlerFunc {
	keySet := buildKeySet(cfg.APIKeys)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(r, keySet) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := readBody(r)
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		var payload protocol.IngestPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if payload.DeviceID == "" || payload.Trigger == "" || payload.StateHash == "" {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}

		if err := insertPayload(db, payload, string(body)); err != nil {
			http.Error(w, "db insert failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func devicesHandler(db *sql.DB, cfg serverConfig) http.HandlerFunc {
	keySet := buildKeySet(cfg.APIKeys)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(r, keySet) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			SELECT device_id, COUNT(*), MAX(event_time), MAX(received_at)
			FROM ingested_events
			GROUP BY device_id
			ORDER BY MAX(event_time) DESC
		`)
		if err != nil {
			http.Error(w, "db query failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var out []deviceRow
		for rows.Next() {
			var row deviceRow
			if err := rows.Scan(&row.DeviceID, &row.EventCount, &row.LatestEventTime, &row.LatestSeenAt); err != nil {
				http.Error(w, "db scan failed", http.StatusInternalServerError)
				return
			}
			out = append(out, row)
		}

		writeJSON(w, http.StatusOK, out)
	}
}

func latestEventHandler(db *sql.DB, cfg serverConfig) http.HandlerFunc {
	keySet := buildKeySet(cfg.APIKeys)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(r, keySet) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
		if deviceID == "" {
			http.Error(w, "device_id is required", http.StatusBadRequest)
			return
		}

		row, err := queryLatestEvent(db, deviceID)
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "db query failed", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, row)
	}
}

func eventsHandler(db *sql.DB, cfg serverConfig) http.HandlerFunc {
	keySet := buildKeySet(cfg.APIKeys)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(r, keySet) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
		if deviceID == "" {
			http.Error(w, "device_id is required", http.StatusBadRequest)
			return
		}

		limit := 50
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value <= 0 || value > 500 {
				http.Error(w, "limit must be between 1 and 500", http.StatusBadRequest)
				return
			}
			limit = value
		}

		rows, err := queryEvents(db, deviceID, limit)
		if err != nil {
			http.Error(w, "db query failed", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, rows)
	}
}

func buildKeySet(keys []string) map[string]struct{} {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}
	return keySet
}

func authorized(r *http.Request, keySet map[string]struct{}) bool {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	key := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	_, ok := keySet[key]
	return ok
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func insertPayload(db *sql.DB, payload protocol.IngestPayload, raw string) error {
	_, err := db.Exec(
		`INSERT INTO ingested_events (device_id, event_time, received_at, trigger, state_hash, payload_json)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		payload.DeviceID,
		payload.EventTime.Format(time.RFC3339Nano),
		time.Now().UTC().Format(time.RFC3339Nano),
		payload.Trigger,
		payload.StateHash,
		raw,
	)
	return err
}

func queryLatestEvent(db *sql.DB, deviceID string) (eventRow, error) {
	var raw string
	var receivedAt string

	err := db.QueryRow(`
		SELECT received_at, payload_json
		FROM ingested_events
		WHERE device_id = ?
		ORDER BY event_time DESC, id DESC
		LIMIT 1
	`, deviceID).Scan(&receivedAt, &raw)
	if err != nil {
		return eventRow{}, err
	}

	return decodeEventRow(receivedAt, raw)
}

func queryEvents(db *sql.DB, deviceID string, limit int) ([]eventRow, error) {
	rows, err := db.Query(`
		SELECT received_at, payload_json
		FROM ingested_events
		WHERE device_id = ?
		ORDER BY event_time DESC, id DESC
		LIMIT ?
	`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []eventRow
	for rows.Next() {
		var receivedAt string
		var raw string
		if err := rows.Scan(&receivedAt, &raw); err != nil {
			return nil, err
		}

		row, err := decodeEventRow(receivedAt, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}

	return out, nil
}

func decodeEventRow(receivedAt string, raw string) (eventRow, error) {
	var payload protocol.IngestPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return eventRow{}, err
	}

	ts, err := time.Parse(time.RFC3339Nano, receivedAt)
	if err != nil {
		return eventRow{}, err
	}

	return eventRow{
		ReceivedAt: ts,
		Payload:    payload,
	}, nil
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
