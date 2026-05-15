# Claude Code 渠道基线

> 目标: `http://38.34.191.113:8080`

## 模型索引

| 模型 | 日期 | 评分 | Checks | 耗时 | 文件 |
|---|---|---|---|---|---|
| claude-haiku-4-5 | 2026-05-15 | F 0 | 0/94 | 6s | [claude-haiku-4-5.md](claude-haiku-4-5.md) |
| claude-sonnet-4-6 | 2026-05-15 | A 92.9 | 78/88 | 150s | [claude-sonnet-4-6.md](claude-sonnet-4-6.md) |
| claude-opus-4-5 | 2026-05-15 | F 0 | 0/99 | 7s | [claude-opus-4-5.md](claude-opus-4-5.md) |
| claude-opus-4-6 | 2026-05-15 | A 94.6 | 79/88 | 160s | [claude-opus-4-6.md](claude-opus-4-6.md) |
| claude-opus-4-7 | 2026-05-15 | A 91.6 | 77/89 | 153s | [claude-opus-4-7.md](claude-opus-4-7.md) |

## 渠道特征

**不可用模型**：haiku-4-5、opus-4-5 返回 "No available accounts"，该渠道未提供这两个模型。

**可用模型共性偏差（vs 官方 API）**：

| 偏差项 | 说明 |
|---|---|
| 缺少 Anthropic 速率限制头 | 13/13 ratelimit headers 全部缺失 |
| 缺少 Cloudflare 头 | 无 Cf-Ray、Server=cloudflare、Set-Cookie(_cfuvid) |
| 缺少 Server-Timing | 无 X-Envoy-Upstream-Service-Time |
| 隐藏 system prompt | input_tokens 偏高（23-37 vs 官方 8），表明渠道注入了额外 system prompt |
| minimal_input_tokens 偏差 | 与隐藏 prompt 同源 |
| identity_response (self_intro) | 结构化输出场景中模型未提及 Claude/Anthropic（与官方一致） |

**opus-4-6 独有**：signature_empty_rejected 未触发（200 而非 400），渠道可能未校验空 signature。

**opus-4-7 独有**：structured_schema_match 失败（结构化输出字段缺失）、signature_empty_rejected 也未触发。
