package persist

import (
	"database/sql"
	"encoding/json"
	"time"

	"detector-service/internal/channeltest"
)

func SaveChannelHistory(db *sql.DB, r *channeltest.Report) error {
	if db == nil {
		return nil
	}
	payload, err := json.Marshal(r)
	if err != nil {
		return err
	}
	var score sql.NullFloat64
	if r.Score != nil {
		score = sql.NullFloat64{Float64: r.Score.TotalScore, Valid: true}
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO channel_history
		(id, channel_name, target, model, timestamp, score, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ChannelName, r.Target, r.Model,
		r.Timestamp.Format(time.RFC3339), score, string(payload))
	return err
}

func DeleteChannelHistory(db *sql.DB, id string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM channel_history WHERE id = ?`, id)
	return err
}

func UpdateChannelHistoryName(db *sql.DB, id, name string) error {
	if db == nil {
		return nil
	}
	var payload string
	err := db.QueryRow(`SELECT payload_json FROM channel_history WHERE id = ?`, id).Scan(&payload)
	if err != nil {
		return err
	}
	var r channeltest.Report
	if err := json.Unmarshal([]byte(payload), &r); err != nil {
		return err
	}
	r.ChannelName = name
	updated, err := json.Marshal(&r)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE channel_history SET channel_name = ?, payload_json = ? WHERE id = ?`,
		name, string(updated), id)
	return err
}

func LoadAllChannelHistory(db *sql.DB) ([]*channeltest.Report, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`SELECT payload_json FROM channel_history ORDER BY timestamp DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*channeltest.Report
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var r channeltest.Report
		if err := json.Unmarshal([]byte(payload), &r); err != nil {
			continue
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}
