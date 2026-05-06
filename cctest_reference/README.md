# cctest 标准参考数据

完整的 Claude Code 探测测试集，包含请求 prompt 和各渠道的实际响应。

## 目录结构

```
cctest_reference/
├── README.md
├── prompts/
│   ├── opus-4-6/            # claude-opus-4-6 测试请求 body（11 个）
│   ├── opus-4-7/            # claude-opus-4-7 测试请求 body（11 个）
│   └── sonnet-4-6/          # claude-sonnet-4-6 测试请求 body（11 个）
│       ├── 00_precheck.json           # 连通性预检 (say ok, stream=true)
│       ├── 01_tag_replay.json         # Tag 复读 (thinking=adaptive, 28 tools, stream=true)
│       ├── 02_logic_reasoning.json    # 逻辑推理 (thinking=adaptive, stream=true)
│       ├── 03_identity_probe.json     # 身份探测 + 毒药题 (thinking=adaptive, non-stream)
│       ├── 04_self_intro.json         # 自我介绍 (output_config json_schema, stream=true)
│       ├── 05_tool_use.json           # Web Search 工具调用 (tool_choice forced, stream=true)
│       ├── 06_image_ocr.json          # 图片 OCR (base64 image, stream=true)
│       ├── 07_pdf_extract.json        # PDF 文本提取 (base64 pdf, stream=true)
│       ├── 08_mini_probe.json         # 精确探测 (max_tokens=1, non-stream)
│       ├── 09_hidden_prompt.json      # 隐藏 prompt 检测 (裸请求无 system, non-stream)
│       └── 10_magic_refusal.json      # Magic refusal 检测 (non-stream)
└── responses/
    ├── claude-console/      # 官方 Console API 响应（3 模型 x 11 测试 = 33 个）
    │   ├── opus-4-6/
    │   ├── opus-4-7/
    │   └── sonnet-4-6/
    ├── azure/               # Azure 渠道（经一层号池中转）
    │   ├── opus-4-6/        # 11 个
    │   ├── opus-4-7/        # 11 个
    │   └── sonnet-4-6/      # 3 个（号池不可用）
    ├── claude-console/      # 官方 Console API（直连）
    │   ├── opus-4-6/        # 11 个
    │   ├── opus-4-7/        # 11 个
    │   └── sonnet-4-6/      # 11 个
    ├── kiro/                # Kiro 渠道（经一层号池中转）
    │   ├── opus-4-6/        # 11 个
    │   ├── opus-4-7/        # 11 个
    │   └── sonnet-4-6/      # 5 个（号池不可用）
    └── windsurf/            # Windsurf 逆向渠道
        ├── opus-4-6/        # 11 个
        ├── opus-4-7/        # 11 个
        └── sonnet-4-6/      # 11 个
            └── *.json       # 统一 JSON 格式: {headers, body}
```

## 测试项详情

| # | 文件 | 类型 | 流式 | 检测维度 |
|---|------|------|------|----------|
| 00 | precheck | say ok | stream | 连接 + headers + SSE 结构 |
| 01 | tag_replay | tag 回显 | stream | LLM 指纹 + 签名 + usage |
| 02 | logic_reasoning | 推理题 | stream | 行为验证 |
| 03 | identity_probe | 身份问答 | non-stream | 身份验证 + 结构完整性 |
| 04 | self_intro | 结构化输出 | stream | JSON schema 验证 |
| 05 | tool_use | web_search | stream | 工具调用 + stop_reason |
| 06 | image_ocr | 图片识别 | stream | 多模态 |
| 07 | pdf_extract | PDF 提取 | stream | 多模态 |
| 08 | mini_probe | max_tokens=1 | non-stream | 后端类型 + 精确 token 验证 |
| 09 | hidden_prompt | 裸请求 | non-stream | 隐藏 system prompt 检测 |
| 10 | magic_refusal | 魔术字符串 | non-stream | 官方 refusal 行为 |

## 用法

```bash
# 非流式请求
curl -X POST https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "anthropic-beta: interleaved-thinking-2025-05-14" \
  -d @prompts/opus-4-6/09_hidden_prompt.json

# 流式请求（加 -N 禁止缓冲）
curl -N -X POST https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "anthropic-beta: interleaved-thinking-2025-05-14" \
  -d @prompts/opus-4-6/00_precheck.json
```

## Azure 渠道特征摘要

基于实际测试结果，Azure（经一层号池中转）的关键差异：

### 三模型共性差异（vs 官方 API）

| 检测项 | 官方预期 | Azure 实际 |
|--------|---------|-----------|
| Headers | `Anthropic-Ratelimit-*`, `Cf-Ray`, `Request-Id`, `Set-Cookie` | `X-Ratelimit-*`, 无 Cf 相关 |
| stop_details | `{type: "end_turn"}` | `null` |
| inference_geo | `us` / `eu` | `not_available` |
| magic_refusal | stop_reason="refusal" | stop_reason="end_turn" (正常回复) |
| SSE 尾部 | `\n\n\n` (三换行) | `\n\n` (双换行) |
| delta usage | slim (仅 4 字段) | 包含 input_tokens + cache 等完整字段 |

### 模型间差异

| 检测项 | opus-4-6 | opus-4-7 |
|--------|----------|----------|
| message_start output_tokens | 1 (非零) | 0 或 1 (不稳定) |
| thinking 内容 | 有完整 thinking 文本 | `thinking: ""` (display=omitted 模式，仅有 signature) |
| identity_probe 模型自述 | 声称 `claude-sonnet-4-6` | 声称 `Claude Sonnet 4.6` |
| hidden_prompt input_tokens | 8 (干净) | 13 (略高，可能有额外 overhead) |
| 08_mini_probe content | `[{type:"text", text:"Hi"}]` | `[]` (空 content，max_tokens=1 未输出) |

### sonnet-4-6

两个渠道（Azure/Kiro）号池均 **不可用** (`No available accounts`)，待补充。

---

## Kiro 渠道特征摘要

Kiro 渠道与 Azure 渠道经过同一号池中转，响应结构 **高度一致**：

- Headers 格式完全相同（`X-Ratelimit-*`，UUID 格式 `X-Request-Id`）
- `stop_details`: null，`inference_geo`: "not_available"
- `magic_refusal`: 未触发，正常回复
- thinking 模式与 Azure 一致（opus-4-6 有内容，opus-4-7 display=omitted）

**结论：Azure 和 Kiro 使用相同的上游号池，响应指纹一致，无需区分处理。**

---

## Claude Console vs Azure 对比

基于 33 + 22 个实测响应的完整对比：

### Headers 差异

| Header | Console (官方) | Azure (中转) |
|--------|---------------|-------------|
| `Anthropic-Ratelimit-*` | 有（全 13 项） | 无（用 `X-Ratelimit-*`） |
| `Request-Id` | `req_01...` 格式 | UUID 格式 |
| `Cf-Ray` | 有（`hex16-IATA`） | 无 |
| `Set-Cookie` | `_cfuvid...anthropic.com` | 无 |
| `Server` | `cloudflare` | 无 |
| `x-envoy-upstream-service-time` | 有 | 无 |
| `Anthropic-Organization-Id` | 有 | 无 |

### Body 差异

| 字段 | Console | Azure |
|------|---------|-------|
| `inference_geo` | `global` | `not_available` |
| `stop_details` | `null`（两者一致） | `null` |
| `magic_refusal` | `end_turn`/`max_tokens`（未触发 refusal） | `end_turn`（同样未触发） |

### 重要发现

- **stop_details 两端都为 null** — 这不是 Azure 独有的问题，Console API 也返回 null。检测代码中对 `stop_details` 的预期可能需要更新。
- **magic_refusal 两端都未触发** — Console 的 opus-4-6 返回 `end_turn` 正常回复，opus-4-7 返回 `max_tokens`。此行为可能与 key 的 tier 或模型版本有关。
- **inference_geo 有差异** — Console 返回 `global`，Azure 返回 `not_available`。
- **Headers 差异是最可靠的检测手段** — Anthropic 官方的 header 命名、格式、字段数量与 Azure 中转完全不同。
