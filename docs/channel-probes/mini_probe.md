# mini_probe：极简非流式探针

- 代码：`internal/channeltest/phase_mini_probe.go`
- Tags：`monitor`
- EstTokens：100

## 检测目的

用一个极简非流式请求检查后端类型、最小输出 token、stop_reason、小请求 cache 归零和 token budget。它用于长期监控中的低成本健康采样。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 1,
  "stream": false,
  "system": "<fullSystem>",
  "metadata": "<genMetadata>",
  "messages": [{"role":"user", "content":"hi"}]
}
```

## 产出 checks

`backend_type`, `small_output_tokens`, `small_stop_reason`, `small_ephemeral_zero`, `small_cache_zero`, `token_budget`

## 检测依据

- 非流式 Messages API 响应应返回 message body、content、usage、stop_reason 等公共字段：<https://docs.anthropic.com/en/api/messages>。
- `backend_type`、小请求 token/cache 范围和 token budget 属于本项目本地 fingerprint，用来发现 OpenAI 兼容层、缓存伪造或模型能力表不一致。

## 误报/例外

- `max_tokens=1` 的 stop_reason 和 output token 可能受模型行为、空白输出或官方边缘变化影响。
- 缓存字段可能随 prompt cache 相关实现变化，需结合官方 key baseline。
- 一些官方兼容渠道可能不暴露相同 usage 扩展字段，应结合 profile 降权。

## 本地优化建议

- 保留为 monitor 默认 probe。
- 如果该 probe 偶发失败，不要直接全局判 critical；应触发一次 full channel run 复核。
- 将小请求 token 范围从硬编码阈值逐步改为按模型/profile/baseline 的相对范围。
