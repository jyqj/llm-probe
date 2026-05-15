# SQLite schema 与持久化边界

> 当前项目未上线，SQLite 采用“索引列 + payload_json”的轻量方案。本文档只描述当前 `internal/persist/schema.go` 的 DDL 和使用边界，不引入复杂 migration 体系。

## 1. 持久化原则

- Store 层仍以内存结构为主，SQLite 作为 write-through 与启动恢复。
- 每张表保留少量查询索引列，完整对象写入 `payload_json`。
- 当前没有 schema version；开发期若 schema 变更，应补 smoke 检查而不是假设本地 DB 总是最新。
- `monitor_targets.api_key` 当前明文存储，仅适合本地/内网开发环境。

## 2. 表结构概览

| 表 | 作用 | 主要索引列 |
|---|---|---|
| `monitor_targets` | 长期监控目标 | `id`, `base_url`, `enabled`, `check_type`, `baseline_id` |
| `monitor_runs` | 每次 monitor run | `target_id`, `model`, `check_type`, `status`, `score`, `started_at` |
| `health_states` | 当前健康状态 | `target_id`, `model`, `check_type`, `status`, `score` |
| `baselines` | 官方基线 | `id`, `name`, `model`, `effort`, `thinking_mode`, `max_tokens` |
| `alert_events` | 告警事件 | `rule_name`, `severity`, `status`, `target_id`, `model`；payload 内含 `check_type` |
| `channel_history` | 一次性 channel 历史 | `channel_name`, `target`, `model`, `timestamp`, `score` |
| `intelligence_history` | benchmark 历史 | `dataset_name`, `model`, `effort`, `thinking_mode`, `score_total`, `pass_rate` |
| `channel_keywords` | channel 识别关键词 | `pattern`, `channel`, `scopes`, `enabled` |

## 3. 关键字段

### `monitor_targets`

- `api_key`：明文密钥。不要上传 DB，不要用于公网多租户。
- `payload_json`：完整 `Target`，包括模型、interval、jitter、intelligence 参数等。
- `check_type`：`channel` / `intelligence` / `both`。

### `monitor_runs`

- `payload_json`：完整 `MonitorRun`，包含 channel report、intelligence report、baseline diff、escalation 信息。
- `check_type`：用于区分 channel 与 intelligence；调度和 health state 都按该维度隔离。

### `health_states`

主键：`(target_id, model, check_type)`。

用于 scheduler adaptive interval、status 页面和 alert 判断。

### `baselines`

`payload_json` 保存完整 `Baseline`：

- `ChannelReport`
- `IntelligenceReport`
- `Model`
- `Profile`
- `ThinkingEffort`
- `Effort`
- `ThinkingMode`
- `MaxTokens`

注意：baseline 必须记录足够参数，才能判断当前 run 是否可比。

## 4. 当前技术债

1. **无 schema version**：本地 DB 如果已有旧表，可能与代码 drift。
2. **payload_json 难统计**：后续想分析“某个 check 历史失败率”会很难。
3. **API key 明文**：只能作为本地/内网开发方案。
4. **history 与 monitor run 分散**：channel/intelligence 一次性历史和 monitor_runs 都保存 report，后续要统一查询模型。
5. **alert_events 索引列缺少 check_type**：当前 `check_type` 已进入事件 payload 和内存 dedup key，但表级索引列尚未展开。

## 5. 推荐后续最小补强

在不引入完整 migration 框架的前提下，建议补：

```text
scripts/db-smoke.sh
/api/debug/db-status 或本地 CLI
```

Smoke 至少检查：

- 表是否存在。
- `health_states` 是否含 `check_type`。
- `monitor_runs` 是否含 `check_type`。
- `baselines.payload_json` 是否可反序列化。
- `monitor_targets.api_key` 是否不会被 API 响应泄漏。

## 6. 后续统计型 schema 规划

如果要做更强本地优化，建议再拆只读分析表：

```text
channel_probe_results(report_id, probe_id, latency_ms, status)
channel_check_results(report_id, probe_id, check_name, pass, category, actual, expected)
intelligence_task_results(report_id, task_id, score, pass, error)
baseline_task_scores(baseline_id, dataset, task_id, score)
```

这些表不必现在实现，但文档和后续设计应按这个方向避免 payload_json 成为唯一数据源。
