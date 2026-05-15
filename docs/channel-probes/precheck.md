# precheck：基线流式探针

- 代码：`internal/channeltest/phase_precheck.go`
- Tags：`required`, `monitor`
- EstTokens：500

## 检测目的

用一个低成本流式请求快速确认目标是否具备 Claude Messages API 的基本流式形态，并收集 headers、SSE 事件顺序、message_start usage、cache/上游泄漏等结构指纹。

这是长期 channel monitor 的第一层轻量面，也是 full run 的必跑基线。

## 请求形态

```jsonc
POST {base_url}/v1/messages
headers: x-api-key, anthropic-version: 2023-06-01
body: {
  "model": "<model>",
  "messages": [{"role":"user", "content":"say ok"}],
  "max_tokens": 20,
  "stream": true
}
```

## 产出 checks

`headers`, `request_id`, `x_new_api_version`, `cf_headers`, `server_timing`, `cf_ray_format`, `cookie_domain`, `server_header`, `sse_done`, `sse_event_order`, `sse_tailing`, `delta_usage_slim`, `sse_ping_position`, `message_start_output_zero`, `container`, `bedrock_state`, `cache_small_probe`

## 检测依据

- 官方 Messages API 和 streaming 文档说明了 `/v1/messages`、SSE 流式响应、headers、usage 等公共形态：<https://docs.anthropic.com/en/api/messages>、<https://docs.anthropic.com/en/docs/build-with-claude/streaming>。
- `request_id`、Cloudflare 头、`container`、`bedrock_state`、`cache_small_probe` 属于本项目本地 clean/reverse 对照 fingerprint，不应写成官方公开保证。
- `sse_done` 用来识别 OpenAI 风格 `[DONE]` 哨兵泄漏；Claude streaming 以事件流为主，本项目把 `[DONE]` 视为兼容层痕迹。

## 误报/例外

- Bedrock / Vertex / Max profile 可能天然缺少或改变部分头部，应通过 profile 降权，不应直接判死。
- 代理/CDN 可能改写 `server`、`server-timing`、`cf-*` 头。
- 小请求的 cache/usage 细节可能随官方实现变化，需要定期用官方 key 复测。

## 本地优化建议

- 长期监控保留该 probe，但不要把头部类 check 作为唯一健康依据。
- 若发现 clean 渠道稳定变化，优先在 `profile.go` 或 check 期望中做 profile-specific 调整。
- 为本 probe 保存典型 clean/reverse raw SSE 片段，后续调参时先对照样本而不是凭经验改阈值。
