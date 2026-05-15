# minimal_tokens：最小 Token 计费探针

- 代码：`internal/channeltest/phase_minimal_tokens.go`
- Tags：`monitor`
- EstTokens：20

## 检测目的

发送最小文本请求，检查 usage 中 input/output token 是否落在项目当前经验范围，用于发现 OpenAI 兼容层 token 计数、假 usage 或异常网关改写。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 4,
  "stream": false,
  "messages": [{"role":"user", "content":"hi"}]
}
```

## 产出 checks

`minimal_input_tokens`, `minimal_output_tokens`

当前实现期望：

- input tokens：8-16
- output tokens：1-4

## 检测依据

- Messages API 响应包含 usage/token 统计；公开接口只支撑字段存在，不保证本项目硬编码范围：<https://docs.anthropic.com/en/api/messages>。
- token 范围是本地官方 key 样本经验，需要随 tokenizer、模型和接口版本复测。

## 误报/例外

- token 计数可能因模型、系统隐式文本、tokenizer 更新而变化。
- `max_tokens=4` 时输出长度和 stop_reason 可能不稳定。
- 兼容官方渠道可能使用不同 usage 扩展字段。

## 本地优化建议

- 保留为 monitor 弱信号。
- 将硬编码范围升级为 per-model/per-profile baseline 范围。
- 失败时触发 full run，不单独判定渠道异常。
