package persist

import (
	"database/sql"
	"encoding/json"
	"time"

	"detector-service/internal/channeltest"
)

func SaveKeyword(db *sql.DB, kw *channeltest.CustomKeyword) error {
	if db == nil {
		return nil
	}
	scopes, _ := json.Marshal(kw.Scopes)
	_, err := db.Exec(`INSERT OR REPLACE INTO channel_keywords
		(id, pattern, channel, scopes, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		kw.ID, kw.Pattern, kw.Channel, string(scopes), kw.Enabled,
		kw.CreatedAt.Format(time.RFC3339))
	return err
}

func DeleteKeyword(db *sql.DB, id string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM channel_keywords WHERE id = ?`, id)
	return err
}

func LoadAllKeywords(db *sql.DB) ([]*channeltest.CustomKeyword, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`SELECT id, pattern, channel, scopes, enabled, created_at FROM channel_keywords ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*channeltest.CustomKeyword
	for rows.Next() {
		var kw channeltest.CustomKeyword
		var scopesJSON, createdAt string
		var enabled int
		if err := rows.Scan(&kw.ID, &kw.Pattern, &kw.Channel, &scopesJSON, &enabled, &createdAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(scopesJSON), &kw.Scopes)
		kw.Enabled = enabled != 0
		kw.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		out = append(out, &kw)
	}
	return out, rows.Err()
}
