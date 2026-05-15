# Benchmark 官方基线对比

## 1. 目标

Benchmark / Intelligence 检测不追求“绝对满分”，而是回答：

```text
当前渠道在同一批静态题上的表现，是否接近官方 key 在同模型、同 effort、同参数下的表现？
```

当前静态 benchmark 已足够；后续优化重点应放在 baseline 录制、参数一致性、逐题偏差分析和 monitor 升级策略。

## 2. 当前实现入口

一次性运行：

- `POST /api/intelligence/datasets/{name}/run`
- `POST /api/intelligence/datasets/{name}/stream`

基线管理：

- `GET /api/monitor/baselines`
- `POST /api/monitor/baselines`
- `GET /api/monitor/baselines/{id}`
- `DELETE /api/monitor/baselines/{id}`

核心代码：

- `internal/intelligence/types.go`
- `internal/intelligence/runner.go`
- `internal/monitor/baseline.go`
- `internal/monitor/diff.go`
- `internal/api/handler_intelligence.go`
- `internal/api/handler_monitor.go`

## 3. 正确基线录制方式

录制官方基线时必须固定：

| 维度 | 要求 |
|---|---|
| dataset | 与目标检测完全一致，例如 `SWE-Atlas-QnA` |
| task ids | 同一批题；monitor sample 必须能和 baseline 重叠 |
| model | 同模型 ID；不要跨模型比较 |
| effort / thinking | 同 `effort`、`thinking_mode`、`max_tokens` |
| key | 使用被信任的官方 key / 官方兼容渠道 |
| profile | channel baseline 需记录 profile |
| 时间 | 官方模型版本更新后应重录 |

## 4. 对比输出

`RunReport.CompareToBaseline()` 输出：

- `baseline_score`
- `current_score`
- `relative_score`
- `overlapping_tasks`
- `consistent_tasks`
- `task_comparisons[]`

一致性规则当前为逐题 deviation 在 `[-0.1, 0.1]` 内算一致。

Monitor 侧 `DiffIntelligence()` 以 `TaskID` 为 key 做偏差；如果当前题执行出错且 baseline 有分数，当前分按 0 参与偏差，避免错误被隐藏。

## 5. 长期监控中的 benchmark

当前 `MonitorRunner.runIntelligence()`：

1. 选 dataset；默认取 registry 第一个。
2. 先跑 `ImportantFirst: true` 的小样本，默认 `limit=3`。
3. 如果有 baseline，并且任一重叠题 `current - baseline < -threshold`，触发升级。
4. 升级到 `intelligence_max_limit`；`0` 表示全量。
5. 记录 `IntelligenceSurface`：`important` / `expanded` / `full`。

## 6. 不建议做的事

- 不要用固定 pass rate 当健康阈值。
- 不要要求静态 benchmark 全部通过。
- 不要把不同 effort、不同模型、不同题集混在一起比较。
- 不要把 evaluator 的字符串匹配结果解释为完整能力评分。

## 7. 官方参考

Benchmark 题库和评分主要是本项目内部策略；官方文档只支撑 API 调用、thinking/effort 参数和 rate limit 约束：

- Messages API：<https://docs.anthropic.com/en/api/messages>
- Extended thinking：<https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking>
- Rate limits：<https://docs.anthropic.com/en/api/rate-limits>
