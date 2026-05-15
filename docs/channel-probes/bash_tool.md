# bash_tool：Bash 工具探针

- 代码：`internal/channeltest/phase_bash_tool.go`
- Tags：full
- EstTokens：200

## 检测目的

验证目标对 `bash` 工具类型的支持与校验：合法 `bash` tool 应产生 tool_use，非法 tool name 应被拒绝。该 probe 主要检查工具 schema 与错误处理路径。

## 请求形态

合法请求：

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 1024,
  "stream": false,
  "tools": [{"type":"bash_20250124", "name":"bash"}],
  "tool_choice": {"type":"tool", "name":"bash"},
  "messages": [{"role":"user", "content":"Run: echo hello"}]
}
```

非法请求：

```jsonc
POST {base_url}/v1/messages
headers: anthropic-beta: interleaved-thinking-2025-05-14
body: {
  "tools": [{"type":"bash_20250124", "name":"invalid_name"}],
  "messages": [{"role":"user", "content":"hello"}]
}
```

## 产出 checks

`bash_stop_reason`, `bash_tool_name`, `bash_tool_rejected`

## 检测依据

- 官方 tool use 文档可支撑工具定义、tool_choice 与 tool_use 响应的一般形态：<https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/overview>。
- `bash_20250124` 属于项目当前使用的特定工具类型；如需确认最新支持范围，应以官方工具文档/账号权限为准，不要仅凭本地代码推断。
- 非法 name 返回 400 是当前本地校验预期。

## 误报/例外

- bash/computer 类工具通常强依赖模型、beta、账号和权限；禁用不一定说明渠道伪造。
- 非法工具的错误码/错误体可能随官方版本变化。
- 该 probe 不适合默认 monitor，除非目标明确承诺支持 bash tool。

## 本地优化建议

- 把“工具不支持/未授权”和“结构伪造”分开打标签。
- 对 bash tool 增加 profile 或 capability gate，避免不支持账号被扣分过重。
- 保存合法与非法请求的原始 status/error body。
