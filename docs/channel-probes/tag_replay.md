# tag_replay：全量指纹探针

- 代码：`internal/channeltest/phase_tag_replay.go`
- Tags：`required`, `NeedsThinking`
- EstTokens：25000

## 检测目的

用高 token、thinking、tools、metadata、streaming 的复杂请求压出完整响应结构，检测 message ID、model 名、thinking/signature、usage、SSE、stop 字段、cache、service tier 与 tag replay 行为。

它是 full channel 检测的主力，不适合高频 monitor。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 64000,
  "stream": true,
  "system": "<fullSystem>",
  "tools": "<data.Tools>",
  "metadata": "<genMetadata>",
  "thinking": "<ThinkingParam(model) if supported>",
  "messages": [{"role":"user", "content":"echo random <tag> + noise question"}]
}
```

## 产出 checks

`id_format`, `model_name`, `signature`, `signature_length`, `thinking_present`, `usage_structure`, `field_order`, `inference_geo`, `stop_details`, `stop_details_structure`, `stop_reason`, `stop_sequence_null`, `thinking_order`, `thinking_display_omitted`, `tag_replay`, `cache_fake`, `message_start_usage`, `sse_ping_position`, `service_tier`, `signature_type_leak`, `usage_fields_complete`, `cache_creation_complete`

## 检测依据

- 官方 extended thinking 文档可作为 thinking block 与 signature 处理的公共依据：<https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking>。
- 官方 streaming 文档可支撑 SSE event 与 message delta 的公共形态：<https://docs.anthropic.com/en/docs/build-with-claude/streaming>。
- `msg_01`、`inference_geo`、`stop_details`、`service_tier`、字段顺序、cache 指纹等为本地经验型 fingerprint，需用官方 key 与已知逆向渠道持续校准。
- `tag_replay` 是行为一致性检测：模型应能回显本轮随机 tag，避免被固定模板或代理层截断上下文。

## 误报/例外

- 不同模型/平台的 thinking 行为可能差异较大，尤其 adaptive thinking 可能不总是产生 thinking block。
- 高 `max_tokens` 和工具上下文容易触发限流或预算差异；请求失败不一定等于渠道伪造。
- tag 未回显可能来自模型回答策略或上下文截断，需结合结构类 check 判断。

## 本地优化建议

- 保持该 probe 为 required/full；不要进入高频 monitor。
- 每次官方模型版本变动后，用官方 key 录制 baseline，再决定是否调整 thinking/signature/cache 相关 check。
- 把 raw exchange 与 check detail 保存到历史，方便后续对单 check 做本地回放。
