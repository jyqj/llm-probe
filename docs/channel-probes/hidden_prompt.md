# hidden_prompt：隐藏 Prompt 检测

- 代码：`internal/channeltest/phase_hidden_prompt.go`
- Tags：`monitor`
- EstTokens：20

## 检测目的

用最小请求检查响应体是否泄漏隐藏 system prompt、代理层模板、内部配置或不应出现的上游内容。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 1,
  "messages": [{"role":"user", "content":"hi"}]
  // intentionally no system, no tools, no metadata
}
```

## 产出 checks

`hidden_prompt`

## 检测依据

- 官方 Messages API 只说明用户可提交 messages/system 等字段；本 probe 的检测核心是本地泄漏规则，不是官方公开字段要求：<https://docs.anthropic.com/en/api/messages>。
- 空 system/tools/metadata 下出现内部 prompt 或代理层文本，通常说明网关注入或模型上下文污染。

## 误报/例外

- `max_tokens=1` 可能导致内容过短，无法稳定暴露泄漏。
- 某些渠道会自动注入安全或身份系统提示，但未必代表逆向；需结合词库和 profile 判断。

## 本地优化建议

- 保留为 monitor 低成本探针。
- 维护可配置泄漏关键词库，区分 fatal / warning / info。
- 一旦失败，应触发 full run 并保存 raw response。
