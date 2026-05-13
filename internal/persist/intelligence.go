package persist

import (
	"database/sql"
	"encoding/json"
	"time"

	"detector-service/internal/intelligence"
)

func SaveIntelligenceHistory(db *sql.DB, r *intelligence.RunReport) error {
	if db == nil {
		return nil
	}
	payload, err := json.Marshal(r)
	if err != nil {
		return err
	}
	var scoreTotal, passRate sql.NullFloat64
	if r.ScoreTotal != nil {
		scoreTotal = sql.NullFloat64{Float64: *r.ScoreTotal, Valid: true}
	}
	if r.PassRate != nil {
		passRate = sql.NullFloat64{Float64: *r.PassRate, Valid: true}
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO intelligence_history
		(id, dataset_name, model, started_at, score_total, pass_rate, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.DatasetName, r.Model, r.StartedAt.Format(time.RFC3339),
		scoreTotal, passRate, string(payload))
	return err
}

func DeleteIntelligenceHistory(db *sql.DB, id string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM intelligence_history WHERE id = ?`, id)
	return err
}

func LoadAllIntelligenceHistory(db *sql.DB) ([]*intelligence.RunReport, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`SELECT payload_json FROM intelligence_history ORDER BY started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*intelligence.RunReport
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var r intelligence.RunReport
		if err := json.Unmarshal([]byte(payload), &r); err != nil {
			continue
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}
