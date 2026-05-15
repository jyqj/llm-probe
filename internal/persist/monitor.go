package persist

import (
	"database/sql"
	"encoding/json"
	"time"

	"detector-service/internal/monitor"
)

// ── Targets ──

func SaveTarget(db *sql.DB, t *monitor.Target) error {
	if db == nil {
		return nil
	}
	payload, err := json.Marshal(t)
	if err != nil {
		return err
	}
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO monitor_targets
		(id, name, base_url, api_key, enabled, check_type, interval_ms, jitter_ms,
		 baseline_id, payload_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.BaseURL, t.APIKey, enabled, t.CheckType,
		t.Interval.Milliseconds(), t.Jitter.Milliseconds(), t.BaselineID,
		string(payload),
		t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339))
	return err
}

func DeleteTarget(db *sql.DB, id string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM monitor_targets WHERE id = ?`, id)
	return err
}

func LoadAllTargets(db *sql.DB) ([]*monitor.Target, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`SELECT api_key, payload_json FROM monitor_targets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*monitor.Target
	for rows.Next() {
		var apiKey, payload string
		if err := rows.Scan(&apiKey, &payload); err != nil {
			return nil, err
		}
		var t monitor.Target
		if err := json.Unmarshal([]byte(payload), &t); err != nil {
			continue
		}
		t.APIKey = apiKey
		out = append(out, &t)
	}
	return out, rows.Err()
}

// ── Health States ──

func SaveHealthState(db *sql.DB, hs *monitor.HealthState) error {
	if db == nil {
		return nil
	}
	ct := hs.CheckType
	if ct == "" {
		ct = "channel"
	}
	payload, err := json.Marshal(hs)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO health_states
		(target_id, model, check_type, status, score, grade, last_check, last_change,
		 consec_fails, consec_ok, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hs.TargetID, hs.Model, ct, string(hs.Status), hs.Score, hs.Grade,
		hs.LastCheck.Format(time.RFC3339), hs.LastChange.Format(time.RFC3339),
		hs.ConsecFails, hs.ConsecOK, string(payload))
	return err
}

func DeleteHealthStates(db *sql.DB, targetID string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM health_states WHERE target_id = ?`, targetID)
	return err
}

func DeleteHealthState(db *sql.DB, targetID, model, checkType string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM health_states WHERE target_id = ? AND model = ? AND check_type = ?`, targetID, model, checkType)
	return err
}

func LoadAllHealthStates(db *sql.DB) ([]*monitor.HealthState, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`SELECT payload_json FROM health_states`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*monitor.HealthState
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var hs monitor.HealthState
		if err := json.Unmarshal([]byte(payload), &hs); err != nil {
			continue
		}
		out = append(out, &hs)
	}
	return out, rows.Err()
}

// ── Runs ──

func SaveRun(db *sql.DB, run *monitor.MonitorRun) error {
	if db == nil {
		return nil
	}
	payload, err := json.Marshal(run)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO monitor_runs
		(id, target_id, model, check_type, status, score, grade,
		 started_at, elapsed_ms, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.TargetID, run.Model, run.CheckType,
		string(run.Status), run.Score, run.Grade,
		run.StartedAt.Format(time.RFC3339), run.ElapsedMs, string(payload))
	return err
}

func LoadAllRuns(db *sql.DB, maxRuns int) ([]*monitor.MonitorRun, error) {
	if db == nil {
		return nil, nil
	}
	if maxRuns <= 0 {
		maxRuns = 1000
	}
	rows, err := db.Query(`SELECT payload_json FROM monitor_runs
		ORDER BY started_at DESC LIMIT ?`, maxRuns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*monitor.MonitorRun
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var run monitor.MonitorRun
		if err := json.Unmarshal([]byte(payload), &run); err != nil {
			continue
		}
		out = append(out, &run)
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

func TrimRuns(db *sql.DB, maxRuns int) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM monitor_runs WHERE id NOT IN
		(SELECT id FROM monitor_runs ORDER BY started_at DESC LIMIT ?)`, maxRuns)
	return err
}
