# identity_probe：身份识别探针

- 代码：`internal/channeltest/phase_identity.go`
- Tags：full
- EstTokens：25000

## 检测目的

通过身份追问、内部代号诱导和经典毒药题，检查非流式响应结构、模型自述、平台泄漏、内部 codename 泄漏、推理能力和若干 body 字段指纹。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 64000,
  "thinking": {"type":"adaptive"},
  "system": "<fullSystem>",
  "tools": "<data.Tools>",
  "metadata": "<genMetadata>",
  "messages": [{"role":"user", "content":"identity question + poison bottle puzzle"}]
  // stream omitted, non-streaming
}
```

## 产出 checks

`nonstream_fields`, `nonstream_type`, `nonstream_role`, `field_order`, `body_key_order`, `id_format`, `identity_response`, `identity_no_leak`, `identity_platform`, `poison_answer`, `stop_sequence_null`, `service_tier`, `signature_type_leak`, `usage_fields_complete`

## 检测依据

- 官方 Messages API 可支撑非流式 message body、role、type、content、usage、stop_reason 的基本结构：<https://docs.anthropic.com/en/api/messages>。
- 身份自述、平台声明、内部代号和毒药题属于行为 heuristic，用来发现代理层 system prompt 泄漏、套壳平台泄漏或低质量模型替换。
- 字段顺序、`service_tier`、`signature_type_leak` 等为本地 fingerprint，不是官方公开契约。

## 误报/例外

- 身份回答可能因系统提示、地区、模型版本而变化；应避免把“没有精确措辞”当失败。
- 毒药题只做轻量 sanity check，不等价于 benchmark 智商检测。
- Bedrock / Vertex / Max profile 可能在 ID、headers 或平台相关字段上天然不同。

## 本地优化建议

- 把身份/平台泄漏 check 做成可配置关键词库，避免写死所有候选平台。
- 单独保存用户 prompt 与模型回答，用于排查误报。
- 若要加强智商检测，不在这里加题，应去 benchmark/static dataset 侧维护。
