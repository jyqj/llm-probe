# Channel Probe 基线记录

> 本目录存放各渠道的真实 probe 运行结果。每份记录使用官方定义的 15 个 probe 顺序执行，记录所有 check 的通过状态和实际观察值。

## 渠道索引

| 渠道 | 模型数 | 最高评分 | 目录 |
|---|---|---|---|
| Claude Console | 5 | A+ 97.5 | [claude-console/](claude-console/) |
| Claude Code | 5 | A 94.6 | [claude-code/](claude-code/) |

## 目录结构

```
baselines/
  <channel-name>/
    README.md                    # 渠道汇总 + 模型索引
    <model>.md                   # 逐 check 可读文档
    <model>.json                 # 原始 JSON 完整数据
```

## 添加新渠道

1. 在 `baselines/` 下建渠道文件夹
2. 运行 `go run scripts/run_probes.go <api-key> <model> [target-url]` 获取 JSON
3. 运行 `python3 scripts/gen_baseline_doc.py <json> <channel-name> <output.md>` 生成文档
4. 更新渠道 README 和本索引
