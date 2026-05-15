# effort_thinking：Effort 级别思考探针

- 代码：`internal/channeltest/phase_effort_thinking.go`
- Tags：`NeedsThinking`
- EstTokens：2000

## 检测目的

按模型能力表测试 `low`、`medium`、`high`、`max`、`xhigh` effort 级别下的 thinking/content/signature 行为，确认目标渠道对 effort/output_config 与 thinking 参数的支持是否接近官方渠道。

## 请求形态

该 probe 会按支持的 effort 发送多次非流式请求：

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 256 或 16000,
  "stream": false,
  "messages": [{"role":"user", "content":"short or reasoning question"}],
  "thinking": "<ThinkingParam(model) if supported>",
  "output_config": {"effort":"low|medium|high|max|xhigh"}
}
```

## 产出 checks

`effort_high_thinking`, `effort_high_signature`, `effort_medium_no_think`, `effort_low_no_think`, `effort_max_thinking`, `effort_xhigh_thinking`

## 检测依据

- 官方 extended thinking 文档可作为 thinking block、signature、interleaved thinking 等公共依据：<https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking>。
- `output_config.effort` 是当前项目采用的 effort 请求形态；不同模型能力由 `model_caps.go` 本地维护，需跟随官方版本更新。
- 当前实现已承认 adaptive thinking 下 high/max 不一定稳定产生 thinking block：有有效 content 时可通过；若有 thinking block，则校验 signature base64。

## 误报/例外

- effort 支持强依赖模型版本，`model_caps.go` 过期会造成误判。
- adaptive thinking 可能不产生 thinking block，不能简单按“没有 thinking = 失败”。
- `xhigh` 仅对声明支持的模型运行。

## 本地优化建议

- 保持 `model_caps.go` 作为后端单一真相源，并通过 `/api/meta/models` 供前端消费；前端本地模型表只作为 fallback。
- 对每个 effort 保存官方 key baseline，避免只靠绝对检查。
- 对 signature 只在 thinking block 存在时严格校验，避免 adaptive 模式误伤。
