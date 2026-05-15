# tool_use：工具调用探针

- 代码：`internal/channeltest/phase_tool_use.go`
- Tags：full
- EstTokens：1000

## 检测目的

强制调用 `web_search` server tool，检查工具调用 ID、stop_reason、强制工具遵循、web search 结果结构、server_tool_use 类型、citation 和 server tool usage 统计。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 16000,
  "stream": true,
  "temperature": 1,
  "system": ["billing block", "Claude Code text", "web search instruction"],
  "tools": [{"type":"web_search_20250305", "name":"web_search", "max_uses":1}],
  "tool_choice": {"type":"tool", "name":"web_search"},
  "metadata": "<genMetadata>",
  "messages": [{"role":"user", "content":"Perform a web search ..."}]
}
```

## 产出 checks

`tool_use_id`, `tool_stop_reason`, `tool_forced_compliance`, `web_search_result`, `server_tool_type`, `citations_present`, `server_tool_usage`

## 检测依据

- 官方 tool use 与 web search 文档可支撑工具定义、tool_choice、server tool result/citation/usage 的公共形态：<https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/overview>、<https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/web-search-tool>。
- `toolu_01` / `srvtoolu_01` ID 前缀属于本地 fingerprint，应通过官方 key 样本校准。

## 误报/例外

- web search 可能受账号权限、地区、模型、配额、时间和联网可用性影响。
- 搜索结果内容本身不稳定；应检查结构，不应依赖具体新闻文本。
- 某些官方兼容渠道可能禁用 server tools，需要在 target/profile 文档中标注，不宜纳入高频 monitor。

## 本地优化建议

- 将 web search 权限失败与结构失败分开记录。
- 为工具调用保存 raw content blocks，便于校准 `server_tool_use` / citations check。
- 如果长期需要工具健康监控，应做低频专项 monitor，而不是放进默认 monitor surface。
