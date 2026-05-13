package persist

import (
	"database/sql"
	"encoding/json"
	"time"

	"detector-service/internal/monitor"
)

func SaveBaseline(db *sql.DB, b *monitor.Baseline) error {
	if db == nil {
		return nil
	}
	payload, err := json.Marshal(b)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO baselines
		(id, name, model, effort, thinking_mode, max_tokens, created_at, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.Name, b.Model, b.Effort, b.ThinkingMode, b.MaxTokens,
		b.CreatedAt.Format(time.RFC3339), string(payload))
	return err
}

func DeleteBaseline(db *sql.DB, id string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM baselines WHERE id = ?`, id)
	return err
}

func LoadAllBaselines(db *sql.DB) ([]*monitor.Baseline, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`SELECT payload_json FROM baselines ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*monitor.Baseline
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var b monitor.Baseline
		if err := json.Unmarshal([]byte(payload), &b); err != nil {
			continue
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}
