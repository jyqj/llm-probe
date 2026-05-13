package persist

const schemaSQL = `
CREATE TABLE IF NOT EXISTS monitor_targets (
  id TEXT PRIMARY KEY,
  name TEXT,
  base_url TEXT NOT NULL,
  api_key TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  check_type TEXT,
  interval_ms INTEGER,
  jitter_ms INTEGER,
  baseline_id TEXT,
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS monitor_runs (
  id TEXT PRIMARY KEY,
  target_id TEXT NOT NULL,
  model TEXT NOT NULL,
  check_type TEXT,
  status TEXT,
  score REAL,
  grade TEXT,
  started_at TEXT NOT NULL,
  elapsed_ms INTEGER,
  payload_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_monitor_runs_target_started
  ON monitor_runs(target_id, started_at DESC);

CREATE TABLE IF NOT EXISTS health_states (
  target_id TEXT NOT NULL,
  model TEXT NOT NULL,
  status TEXT,
  score REAL,
  grade TEXT,
  last_check TEXT,
  last_change TEXT,
  consec_fails INTEGER,
  consec_ok INTEGER,
  payload_json TEXT NOT NULL,
  PRIMARY KEY(target_id, model)
);

CREATE TABLE IF NOT EXISTS baselines (
  id TEXT PRIMARY KEY,
  name TEXT,
  model TEXT,
  effort TEXT,
  thinking_mode TEXT,
  max_tokens INTEGER,
  created_at TEXT NOT NULL,
  payload_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS alert_events (
  id TEXT PRIMARY KEY,
  rule_name TEXT,
  severity TEXT,
  status TEXT,
  target_id TEXT,
  model TEXT,
  fired_at TEXT,
  resolved_at TEXT,
  notified INTEGER,
  payload_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alert_events_status_fired
  ON alert_events(status, fired_at DESC);

CREATE TABLE IF NOT EXISTS channel_history (
  id TEXT PRIMARY KEY,
  channel_name TEXT,
  target TEXT,
  model TEXT,
  timestamp TEXT,
  score REAL,
  payload_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS intelligence_history (
  id TEXT PRIMARY KEY,
  dataset_name TEXT,
  model TEXT,
  started_at TEXT,
  score_total REAL,
  pass_rate REAL,
  payload_json TEXT NOT NULL
);
`
