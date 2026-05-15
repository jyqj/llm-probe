# signature_reject：空签名拒绝探针

- 代码：`internal/channeltest/phase_signature_reject.go`
- Tags：`monitor`, `NeedsSignature`
- EstTokens：100

## 检测目的

构造一段带空 `signature` 的历史 assistant thinking block，并继续追问，验证目标是否按预期拒绝无效 thinking signature。它用于捕捉 signature 校验缺失或兼容层伪造。

## 请求形态

```jsonc
POST {base_url}/v1/messages
headers: anthropic-beta: interleaved-thinking-2025-05-14
body: {
  "model": "<model>",
  "max_tokens": 1024,
  "stream": false,
  "thinking": {"type":"enabled", "budget_tokens":5000},
  "messages": [
    {"role":"user", "content":"What is 2+2?"},
    {"role":"assistant", "content":[
      {"type":"thinking", "thinking":"Let me think about this.", "signature":""},
      {"type":"text", "text":"4"}
    ]},
    {"role":"user", "content":"Are you sure?"}
  ]
}
```

## 产出 checks

`signature_empty_rejected`

当前实现期望 HTTP `400`。

## 检测依据

- Extended thinking 文档说明 thinking block/signature 在多轮对话和工具场景中的重要性：<https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking>。
- “空 signature 应拒绝”是本项目根据当前官方行为和本地样本形成的校验点；具体状态码应随官方 key 回归确认。

## 误报/例外

- beta header 或 thinking signature 语义可能随官方版本变更。
- 不支持 signature 的模型不会运行该 probe。
- 某些官方兼容渠道可能在网关层返回不同错误码或错误体。

## 本地优化建议

- 保留 monitor，但只对 `NeedsSignature` 模型启用。
- 除状态码外记录 error type/message，后续可按官方变化调整。
- 若官方 key 行为变化，先更新文档和 baseline，再改 check。
