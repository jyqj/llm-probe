# Claude Console 基线

> 使用 Anthropic 官方 API key 直连 `https://api.anthropic.com` 的基线记录。

## 模型索引

| 模型 | 日期 | 评分 | Checks | 耗时 | 文件 |
|---|---|---|---|---|---|
| claude-haiku-4-5 | 2026-05-15 | D 57.4 | 63/86 | 33s | [claude-haiku-4-5.md](claude-haiku-4-5.md) |
| claude-sonnet-4-6 | 2026-05-15 | A+ 97.5 | 87/88 | 123s | [claude-sonnet-4-6.md](claude-sonnet-4-6.md) |
| claude-opus-4-5 | 2026-05-15 | D 58.1 | 69/91 | 53s | [claude-opus-4-5.md](claude-opus-4-5.md) |
| claude-opus-4-6 | 2026-05-15 | A+ 97.5 | 87/88 | 136s | [claude-opus-4-6.md](claude-opus-4-6.md) |
| claude-opus-4-7 | 2026-05-15 | A+ 97.5 | 88/89 | 131s | [claude-opus-4-7.md](claude-opus-4-7.md) |

## 结果概览

**支持 adaptive thinking 的模型（sonnet-4-6, opus-4-6, opus-4-7）**：全部 A+ 97.5 分，唯一失败是 `identity_response`（self_intro 探针的结构化输出中模型按 schema 输出虚构人物 JSON，不涉及自我介绍）。

**仅支持 enabled thinking 的模型（haiku-4-5, opus-4-5）**：D 级（57-58 分），因多个 probe 硬编码了 `thinking.type=adaptive`，API 返回 400。失败是 probe 兼容性问题，不代表模型本身异常。
