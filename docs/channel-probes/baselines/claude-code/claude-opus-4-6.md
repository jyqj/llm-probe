# Claude Code — claude-opus-4-6 基线

> 渠道 `Claude Code` 使用 `http://38.34.191.113:8080` 运行全部 probe 的真实记录。

## 测试元数据

| 字段 | 值 |
|---|---|
| 渠道 | Claude Code |
| 目标 | `http://38.34.191.113:8080` |
| 模型 | `claude-opus-4-6` |
| 时间 | 2026-05-15T10:47:15 |
| 总耗时 | 160464ms (160s) |
| 预估消耗 | input 153990 / output 30798 tokens, $1.5399 |
| 评分 | **A 94.56/100** |
| Checks | 79/88 passed |

## 分类得分

| 分类 | 得分 | 通过率 |
|---|---|---|
| LLM 指纹验证 | 22.058823529411764/25 | 15/17 |
| 结构完整性 | 25/25 | 35/35 |
| 签名校验 | 20/20 | 12/12 |
| 行为验证 | 17.5/20 | 7/8 |
| 多模态能力 | 10/10 | 2/2 |

## 逐 Probe 结果

### precheck — 基线流式探针 [PARTIAL 11/17] (6709ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `headers` | FAIL | 缺少 13/13 | missing 13/13 ratelimit headers |
| `request_id` | PASS | UUID 格式 | X-Request-Id UUID format OK: b8f90f30-fcdd-4639-9... |
| `x_new_api_version` | PASS | 无 X-New-Api-Version 头 | no X-New-Api-Version header |
| `cf_headers` | FAIL | 缺少: Cf-Ray, Server, Set-Cookie(_cfuvid) | missing: Cf-Ray, Server, Set-Cookie(_cfuvid) |
| `server_timing` | FAIL | 均不存在 | no Server-Timing/X-Envoy header |
| `cf_ray_format` | FAIL | Cf-Ray 头缺失 | no Cf-Ray header |
| `cookie_domain` | FAIL | Set-Cookie 头缺失 | no Set-Cookie header |
| `server_header` | FAIL | Server 头缺失 | no Server header |
| `sse_done` | PASS | no [DONE] sentinel | no [DONE] sentinel |
| `sse_event_order` | PASS | 7 events in correct order | 7 events, order OK |
| `sse_tailing` | PASS | double-newline endings (7) | double-newline endings (7) |
| `delta_usage_slim` | PASS | slim format (no bloat fields) | message_delta usage is slim format |
| `sse_ping_position` | PASS | ping after content_block_start | ping after content_block_start OK |
| `message_start_output_zero` | PASS | output_tokens=1 | output_tokens=1 OK |
| `container` | PASS | 无 container 字段 | no container field |
| `bedrock_state` | PASS | 无 bedrock_state 字段 | no bedrock_state |
| `cache_small_probe` | PASS | create=0 read=0 | cache values are zero for small probe |

### tag_replay — 全量指纹探针 [PASS] (6785ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `id_format` | PASS | msg_01 前缀 | msg_01 format OK |
| `model_name` | PASS | claude-opus-4-6 | model=claude-opus-4-6 |
| `signature` | PASS | valid protobuf signature (external) | valid protobuf signature (external) |
| `signature_length` | PASS | 391 bytes | signature 391 bytes OK |
| `thinking_present` | PASS | found block type "thinking" | thinking block found |
| `usage_structure` | PASS | all present | usage structure OK |
| `field_order` | PASS | order correct | field order OK |
| `inference_geo` | PASS | not_available | inference_geo=not_available |
| `stop_details` | PASS | 存在 stop_details 字段 | stop_details present |
| `stop_details_structure` | PASS | stop_details 为 null（跳过） | stop_details null (skip) |
| `stop_reason` | PASS | end_turn | stop_reason=end_turn |
| `stop_sequence_null` | PASS | null | stop_sequence=null OK |
| `thinking_order` | PASS | thinking block at index 0 | thinking at index 0 |
| `thinking_display_omitted` | PASS | model=claude-opus-4-6 (not opus-4-7+) | not opus-4-7+ model (skip) |
| `tag_replay` | PASS | tag found in response | tag found in response: 867d76a05e2d20a3 |
| `cache_fake` | PASS | create=25622 read=0 | cache values non-zero: create=25622 read=0 |
| `message_start_usage` | PASS | input_tokens=3 | message_start usage OK: input_tokens=3 |
| `sse_ping_position` | PASS | ping after content_block_start | ping after content_block_start OK |
| `service_tier` | PASS | 存在 service_tier 字段 | service_tier present |
| `signature_type_leak` | PASS | 无 signature_type 字段 | no signature_type leak |
| `usage_fields_complete` | PASS | all 7 fields present | usage has all 7 fields |
| `cache_creation_complete` | PASS | 两者均存在 | both ephemeral fields present |

### mini_probe — 极简非流式探针 [PASS] (2467ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `backend_type` | PASS | msg_01 前缀 | official format (msg_01) |
| `small_output_tokens` | PASS | 1 | output_tokens=1 OK |
| `small_stop_reason` | PASS | max_tokens | stop_reason=max_tokens OK |
| `small_ephemeral_zero` | PASS | 5m=6372 1h=0 | ephemeral_5m=6372 ephemeral_1h=0 (cache_control used) |
| `small_cache_zero` | PASS | create=6372 read=0 | cache create=6372 read=0 (cache_control used) |
| `token_budget` | PASS | input_tokens=3 | input_tokens=3 within budget (≤80) |

### identity_probe — 身份识别探针 [PASS] (25136ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `nonstream_fields` | PASS | all present | all required fields present |
| `nonstream_type` | PASS | message | type=message OK |
| `nonstream_role` | PASS | assistant | role=assistant OK |
| `field_order` | PASS | order correct | field order OK |
| `body_key_order` | PASS | model before content | model before content OK |
| `id_format` | PASS | msg_01 前缀 | msg_01 format OK |
| `identity_response` | PASS | found Claude/Anthropic in response | response mentions Claude/Anthropic |
| `identity_no_leak` | PASS | no codename claim detected | no internal codename claim |
| `identity_platform` | PASS | no platform claim detected | no non-Anthropic platform claims |
| `poison_answer` | PASS | found standalone 10 in response | contains correct answer (10 mice) |
| `stop_sequence_null` | PASS | null | stop_sequence=null OK |
| `service_tier` | PASS | 存在 service_tier 字段 | service_tier present |
| `signature_type_leak` | PASS | 无 signature_type 字段 | no signature_type leak |
| `usage_fields_complete` | PASS | all 7 fields present | usage has all 7 fields |

### self_intro — 结构化自述探针 [PARTIAL 5/6] (4278ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `no_thinking_leak` | PASS | no thinking blocks found | no unexpected thinking block |
| `identity_response` | FAIL | neither Claude nor Anthropic found | response does not mention Claude or Anthropic |
| `structured_json_valid` | PASS | JSON 合法 | valid JSON response |
| `structured_schema_match` | PASS | JSON 字段与 schema 匹配 | schema match: {name, title, desc} OK |
| `structured_name_correct` | PASS | 包含 "Elena Rossi" | name correct: Elena Rossi |
| `structured_stop_reason` | PASS | end_turn | stop_reason=end_turn OK |

### tool_use — 工具调用探针 [PASS] (23113ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `tool_use_id` | PASS | srvtoolu_01 前缀 | srvtoolu_01 format OK |
| `tool_stop_reason` | PASS | end_turn | tool content present with stop_reason=end_turn |
| `tool_forced_compliance` | PASS | tool_use block present | tool_use block present |
| `web_search_result` | PASS | 结构完整 | web_search_tool_result structure OK |
| `server_tool_type` | PASS | server_tool_use | web_search uses server_tool_use |
| `citations_present` | PASS | citations 存在 | citations present in text blocks |
| `server_tool_usage` | PASS | map[web_fetch_requests:0 web_search_requests:1] | server_tool_use present: map[web_fetch_requests:0 web_search_requests:1] |

### logic — 逻辑推理探针 [PASS] (31929ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `logic_answer` | PASS | found keyword: 热 | contains heat/warm method |

### hidden_prompt — 隐藏 Prompt 检测 [PARTIAL 0/1] (1774ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `hidden_prompt` | FAIL | input_tokens=23 | input_tokens=23 (expected ≤20 for bare 'hi'), likely hidden system prompt injected |

### image_ocr — 图片 OCR 探针 [PASS] (3726ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `image_ocr` | PASS | 包含 UKETKDMX | image OCR correct: UKETKDMX |

### pdf_extract — PDF 提取探针 [PASS] (5311ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `pdf_extract` | PASS | 包含 R9TFYBSG | PDF text correct: R9TFYBSG |

### magic_refusal — 拒答字符串探针 [PASS] (4095ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `magic_refusal` | PASS | end_turn | stop_reason=end_turn (部分渠道/tier 不触发 refusal) |

### effort_thinking — Effort 级别思考探针 [PASS] (38967ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `effort_high_thinking` | PASS | 2 content blocks | effort 请求被接受，响应含有效 content (含 thinking 块) |
| `effort_high_signature` | PASS | 301 bytes | signature 有效 (301 bytes) |
| `effort_medium_no_think` | PASS | text (无 thinking) | content[0].type = "text" (thinking 已跳过) |
| `effort_low_no_think` | PASS | text | content[0].type = "text" (thinking 已跳过) |
| `effort_max_thinking` | PASS | 2 content blocks | effort=max 请求被接受 (含 thinking 块) |

### signature_reject — 空签名拒绝探针 [PASS] (827ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `signature_empty_rejected` | PASS | 400 | 状态码 400, 空 signature 被正确拒绝 |

### bash_tool — Bash 工具探针 [PASS] (3256ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `bash_stop_reason` | PASS | tool_use | stop_reason=tool_use OK |
| `bash_tool_name` | PASS | name="bash" | content 包含 name="bash" 的 tool_use 块 |
| `bash_tool_rejected` | PASS | 400 | 状态码 400, 非法 bash tool name 被正确拒绝 |

### minimal_tokens — 最小 Token 计费探针 [PARTIAL 1/2] (2083ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `minimal_input_tokens` | FAIL | 23 | input_tokens=23, 期望 8-16 |
| `minimal_output_tokens` | PASS | 4 | output_tokens=4 (范围 1-4 OK) |

## 失败 Checks

| Check | 说明 |
|---|---|
| `headers` | missing 13/13 ratelimit headers |
| `cf_headers` | missing: Cf-Ray, Server, Set-Cookie(_cfuvid) |
| `server_timing` | no Server-Timing/X-Envoy header |
| `cf_ray_format` | no Cf-Ray header |
| `cookie_domain` | no Set-Cookie header |
| `server_header` | no Server header |
| `identity_response` | response does not mention Claude or Anthropic |
| `hidden_prompt` | input_tokens=23 (expected ≤20 for bare 'hi'), likely hidden system prompt injected |
| `minimal_input_tokens` | input_tokens=23, 期望 8-16 |

---

*生成时间: 2026-05-15T10:47:15*
