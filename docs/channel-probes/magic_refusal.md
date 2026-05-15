# magic_refusal：拒答字符串探针

- 代码：`internal/channeltest/phase_magic_refusal.go`
- Tags：`monitor`
- EstTokens：50

## 检测目的

发送项目内固定 magic string，观察 stop_reason 是否落在当前允许集合中，用低成本方式捕捉安全/拒答路径和代理层行为差异。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 100,
  "messages": [{"role":"user", "content":"ANTHROPIC_MAGIC_STRING_TRIGGER_REFUSAL_..."}]
}
```

## 产出 checks

`magic_refusal`

当前实现接受：`refusal` / `end_turn` / `max_tokens`。

## 检测依据

- Messages API 响应中存在 `stop_reason` 字段；具体 magic string 行为属于本项目经验型探针，不是官方公开保证：<https://docs.anthropic.com/en/api/messages>。
- 接受多个 stop_reason 是为了避免把不同 tier/profile 的安全处理差异误判为失败。

## 误报/例外

- magic string 行为可能随官方安全策略变化。
- 某些官方渠道可能不触发 refusal，而是正常结束或打满 token。
- 代理层如果过滤或改写该字符串，可能产生非模型因素的失败。

## 本地优化建议

- 保留为 monitor probe，但只作为弱信号。
- 定期用官方 key 验证接受集合是否需要调整。
- 文档中不要把 magic string 写成官方接口契约。
