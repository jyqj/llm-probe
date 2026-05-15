# Channel probe 文档索引

> 这些文档覆盖 `internal/channeltest/probe.go` 当前注册的 15 个 probe。每篇文档描述当前实现的请求形态和检测依据；如果代码变更，应同步更新本文档。

## Probe 总表

| 顺序 | Probe | Label | Tags | Est tokens | Checks | 文档 |
|---:|---|---|---|---:|---:|---|
| 1 | `precheck` | 基线流式探针 | required, monitor | 500 | 17 | [precheck.md](precheck.md) |
| 2 | `tag_replay` | 全量指纹探针 | required, thinking | 25000 | 22 | [tag_replay.md](tag_replay.md) |
| 3 | `mini_probe` | 极简非流式探针 | monitor | 100 | 6 | [mini_probe.md](mini_probe.md) |
| 4 | `identity_probe` | 身份识别探针 | full | 25000 | 14 | [identity_probe.md](identity_probe.md) |
| 5 | `self_intro` | 结构化自述探针 | full | 25000 | 6 | [self_intro.md](self_intro.md) |
| 6 | `tool_use` | 工具调用探针 | full | 1000 | 7 | [tool_use.md](tool_use.md) |
| 7 | `logic` | 逻辑推理探针 | heavy | 25000 | 1 | [logic.md](logic.md) |
| 8 | `hidden_prompt` | 隐藏 Prompt 检测 | monitor | 20 | 1 | [hidden_prompt.md](hidden_prompt.md) |
| 9 | `image_ocr` | 图片 OCR 探针 | heavy | 25000 | 1 | [image_ocr.md](image_ocr.md) |
| 10 | `pdf_extract` | PDF 提取探针 | heavy | 25000 | 1 | [pdf_extract.md](pdf_extract.md) |
| 11 | `magic_refusal` | 拒答字符串探针 | monitor | 50 | 1 | [magic_refusal.md](magic_refusal.md) |
| 12 | `effort_thinking` | Effort 级别思考探针 | thinking | 2000 | 6 | [effort_thinking.md](effort_thinking.md) |
| 13 | `signature_reject` | 空签名拒绝探针 | monitor, signature | 100 | 1 | [signature_reject.md](signature_reject.md) |
| 14 | `bash_tool` | Bash 工具探针 | full | 200 | 3 | [bash_tool.md](bash_tool.md) |
| 15 | `minimal_tokens` | 最小 Token 计费探针 | monitor | 20 | 2 | [minimal_tokens.md](minimal_tokens.md) |

## 写作约定

每个 probe 文档固定包含：

- 检测目的
- 请求形态
- 产出 checks
- 检测依据
- 误报/例外
- 本地优化建议

## 基线记录

实际运行结果存放在 [baselines/](baselines/) 目录，包含各渠道的逐 check 真实数据。

## 官方参考边界

- 官方文档可支撑公共 API 形态，例如 Messages、SSE、tool use、vision/PDF、extended thinking。
- ID/headers/usage 私有字段、字段顺序、Cloudflare 头、token 范围、magic refusal 等属于本项目经验型 fingerprint，应通过本地 clean/reverse 样本继续校准。
