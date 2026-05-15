#!/usr/bin/env bash
set -euo pipefail

db_path="${1:-data/detector.db}"

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "sqlite3 not found" >&2
  exit 127
fi

if [[ ! -f "$db_path" ]]; then
  echo "database not found: $db_path" >&2
  exit 1
fi

required_tables=(
  monitor_targets
  monitor_runs
  health_states
  baselines
  alert_events
  channel_history
  intelligence_history
  channel_keywords
)

for table in "${required_tables[@]}"; do
  exists="$(sqlite3 "$db_path" "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='$table';")"
  if [[ "$exists" != "1" ]]; then
    echo "missing table: $table" >&2
    exit 1
  fi
done

require_column() {
  local table="$1"
  local column="$2"
  local count
  count="$(sqlite3 "$db_path" "SELECT count(*) FROM pragma_table_info('$table') WHERE name='$column';")"
  if [[ "$count" != "1" ]]; then
    echo "missing column: $table.$column" >&2
    exit 1
  fi
}

require_column monitor_runs check_type
require_column health_states check_type
require_column monitor_targets payload_json
require_column baselines payload_json

sqlite3 "$db_path" <<'SQL'
.headers on
.mode column
SELECT 'monitor_targets' AS table_name, count(*) AS rows FROM monitor_targets
UNION ALL SELECT 'monitor_runs', count(*) FROM monitor_runs
UNION ALL SELECT 'health_states', count(*) FROM health_states
UNION ALL SELECT 'baselines', count(*) FROM baselines
UNION ALL SELECT 'alert_events', count(*) FROM alert_events
UNION ALL SELECT 'channel_history', count(*) FROM channel_history
UNION ALL SELECT 'intelligence_history', count(*) FROM intelligence_history
UNION ALL SELECT 'channel_keywords', count(*) FROM channel_keywords;
SQL

echo "DB SMOKE OK: $db_path"
