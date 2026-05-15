# logic：逻辑推理探针

- 代码：`internal/channeltest/phase_logic.go`
- Tags：`heavy`
- EstTokens：25000

## 检测目的

用开关-灯泡经典题检查模型是否给出“热/温度/触摸”相关方法。该 probe 是 channel full 的行为 sanity check，不是主要智商 benchmark。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 64000,
  "stream": true,
  "thinking": {"type":"adaptive"},
  "system": "<fullSystem>",
  "tools": "<data.Tools>",
  "metadata": "<genMetadata>",
  "messages": [{"role":"user", "content":"3 switches / 3 bulbs, then 4 switches / 4 bulbs"}]
}
```

## 产出 checks

`logic_answer`

## 检测依据

- 这是本项目行为 heuristic：正确解法通常涉及灯泡余温/触摸/温度。
- 官方 Messages/streaming 文档只支撑请求与响应形态，不支撑该题本身：<https://docs.anthropic.com/en/api/messages>。

## 误报/例外

- 关键词法可能漏掉等价表述，也可能被无关“热”字误中。
- 该题容易被模型训练数据记忆，不适合作为完整智商指标。
- 由于 `heavy` 且 `max_tokens` 高，不适合长期高频检测。

## 本地优化建议

- 保留为 full/heavy sanity check。
- 真正的智商趋势检测放在 `intelligence` 静态 benchmark，并与官方 key baseline 对比。
- 如果优化本 probe，优先改成多关键词/正则/小 rubric，而不是加入更多大题。
