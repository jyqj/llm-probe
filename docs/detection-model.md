# 检测模型与评分语义

## 1. 两条主检测线

### Channel 检测

Channel 检测回答的问题是：目标 API 在协议、响应结构、流式事件、thinking/signature、工具、多模态、行为自述等维度上，是否接近真实 Claude/Anthropic 渠道。

它不负责判断模型智商高低；即使模型回答聪明，只要响应结构、SSE、headers、signature 或工具形态不一致，也可能被判为渠道风险。

### Benchmark / Intelligence 检测

Benchmark 检测回答的问题是：目标渠道在固定题库上的输出质量，是否接近官方 key 在相同条件下的表现。

当前结论：静态 benchmark 已足够。不要用“必须全对”作为健康条件，应使用官方 key baseline 的相对表现。

## 2. Channel check 分类

`check_registry.go` 把 check 分为五类：

| Category | 权重 | 语义 |
|---|---:|---|
| `fingerprint` | 25 | ID、地理、上游泄漏、token/字段指纹 |
| `structural` | 25 | JSON body、headers、SSE、cache、工具结构 |
| `signature` | 20 | thinking block、signature、effort 行为 |
| `behavioral` | 20 | tag replay、身份、拒答、逻辑行为 |
| `multimodal` | 10 | image / PDF 能力 |

评分由 `CalculateScore(checks, surface, profile)` 完成。profile 可把部分 check 变成 info-only，避免 Bedrock / Vertex / Max 这类官方兼容渠道因为天然差异被误伤。

## 3. Probe surface

| Surface | 目的 | 当前来源 |
|---|---|---|
| full | 一次性深度检查 | `Runner.Run` / `RunStream`，跑 `ProbesForModel(model)` |
| monitor | 长期轻量检查 | `Runner.RunMonitorCtx`，只跑带 `monitor` tag 的 probe |
| heavy | 高成本能力验证 | probe tag 标记为 `heavy`，默认不适合作为高频监控 |

当前 monitor probe：

- `precheck`
- `mini_probe`
- `hidden_prompt`
- `magic_refusal`
- `signature_reject`（仅签名模型）
- `minimal_tokens`

当前 heavy probe：

- `logic`
- `image_ocr`
- `pdf_extract`

## 4. Profile 语义

当前内置 profile：

- `console`：Anthropic Console/API 直连，默认全量评分。
- `bedrock`：AWS Bedrock 官方渠道，部分头部/ID/字段 check 变成 info-only 或 expected-fail。
- `vertex`：Google Vertex AI 官方渠道，部分头部/ID check 降权。
- `max`：Anthropic Max 订阅，头部和若干边缘字段按信息化处理。

Profile 不是“证明官方”的机制，只是告诉评分器：某些差异在这个渠道类型下不应扣分。

## 5. Benchmark baseline 语义

Benchmark 的核心健康定义：

```text
same dataset + same task ids + same model + same effort/thinking + same max_tokens
current channel score ~= official-key baseline score
```

`RunReport.CompareToBaseline()` 会输出：

- `baseline_score`
- `current_score`
- `relative_score`
- `overlapping_tasks`
- `consistent_tasks`
- `task_comparisons[]`

Monitor 中 `DiffIntelligence()` 会按 `TaskID` 做逐题偏差；本轮错误题在有 baseline 分数时按 0 分参与偏差计算，避免运行错误被“跳过”掉。

## 6. 检测依据分层

后续新增或优化 probe 时，必须把依据分三层写清：

1. **官方公开接口依据**：Messages API、SSE、tool use、vision/PDF、extended thinking 等。
2. **本地 clean/reverse 对照经验**：ID 前缀、header 形态、字段泄漏、cache/token 指纹等。
3. **项目内产品策略**：哪些 probe 可进入 monitor，哪些只能 full/heavy，哪些 profile 应降权。

如果一个检测点只来自第 2 或第 3 层，不要在文档中写成官方保证。

## 7. 元数据单一真相源

后端已提供只读元数据 API：

- `GET /api/meta/models`：模型能力、thinking/effort、价格元数据。
- `GET /api/channel/probes`：probe 顺序、标签、成本估算、checks、文档路径。
- `GET /api/channel/checks`：check 分类、显示名、默认修复动作、文档路径。

前端应优先消费这些接口；本地 JS 模型表只作为接口不可用时的 fallback。新增 probe/check/model 时，必须同步：

1. Go 侧注册表或模型能力表。
2. `docs/channel-probes/*.md`。
3. 元数据 API 返回结果的 smoke 校验。

## 8. 官方参考

- Messages API：<https://docs.anthropic.com/en/api/messages>
- Streaming Messages：<https://docs.anthropic.com/en/docs/build-with-claude/streaming>
- Extended thinking：<https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking>
- Web search tool：<https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/web-search-tool>
- Rate limits：<https://docs.anthropic.com/en/api/rate-limits>
