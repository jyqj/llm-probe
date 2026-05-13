package persist

import (
	"database/sql"
	"encoding/json"
	"time"

	"detector-service/internal/alert"
)

func SaveAlertEvent(db *sql.DB, ev *alert.Event) error {
	if db == nil {
		return nil
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	var resolvedAt sql.NullString
	if ev.ResolvedAt != nil {
		resolvedAt = sql.NullString{String: ev.ResolvedAt.Format(time.RFC3339), Valid: true}
	}
	notified := 0
	if ev.Notified {
		notified = 1
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO alert_events
		(id, rule_name, severity, status, target_id, model, fired_at, resolved_at, notified, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.RuleName, string(ev.Severity), string(ev.Status),
		ev.TargetID, ev.Model, ev.FiredAt.Format(time.RFC3339),
		resolvedAt, notified, string(payload))
	return err
}

func UpdateAlertEvent(db *sql.DB, ev *alert.Event) error {
	return SaveAlertEvent(db, ev)
}

func LoadAllAlertEvents(db *sql.DB, maxEvents int) ([]*alert.Event, error) {
	if db == nil {
		return nil, nil
	}
	if maxEvents <= 0 {
		maxEvents = 500
	}
	rows, err := db.Query(`SELECT payload_json FROM alert_events ORDER BY fired_at DESC LIMIT ?`, maxEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*alert.Event
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var ev alert.Event
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		out = append(out, &ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// reverse to chronological order (store expects oldest-first in slice)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func TrimAlertEvents(db *sql.DB, maxEvents int) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM alert_events WHERE id NOT IN
		(SELECT id FROM alert_events ORDER BY fired_at DESC LIMIT ?)`, maxEvents)
	return err
}
