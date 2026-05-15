# self_intro：结构化自述探针

- 代码：`internal/channeltest/phase_self_intro.go`
- Tags：full
- EstTokens：25000

## 检测目的

在不请求 thinking 的情况下，让模型按 JSON schema 输出结构化自述，检查 no-thinking leak、结构化 JSON 有效性、schema 匹配、随机姓名回填和 stop_reason。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 1024,
  "stream": true,
  "system": "<fullSystem>",
  "tools": "<data.Tools>",
  "metadata": "<genMetadata>",
  "messages": [{"role":"user", "content":"Describe a person named <random> ..."}],
  "output_config": {
    "format": {"type":"json_schema", "schema":"name/title/desc object"}
  }
  // no thinking field
}
```

## 产出 checks

`no_thinking_leak`, `structured_json_valid`, `structured_schema_match`, `structured_name_correct`, `structured_stop_reason`

## 检测依据

- Messages API 支撑标准 message 请求/响应和 streaming：<https://docs.anthropic.com/en/api/messages>、<https://docs.anthropic.com/en/docs/build-with-claude/streaming>。
- `output_config.format.json_schema` 是当前项目使用的结构化输出请求形态；若官方结构化输出接口有版本变动，应以官方最新文档为准。
- `no_thinking_leak` 来自 extended thinking 的边界：未请求 thinking 时不应出现 thinking 内容。参考：<https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking>。

## 误报/例外

- 结构化输出接口字段如果随官方版本变化，本 probe 会整体失效。
- 模型可能合法输出等价 JSON，但字段顺序/文本风格不同；schema check 应只校验结构和必要值。

## 本地优化建议

- 为结构化输出单独维护官方文档链接和接口版本注释。
- 保留随机姓名，避免代理层固定模板通过。
