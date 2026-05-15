# Claude Console — claude-opus-4-5 基线

> 渠道 `Claude Console` 使用 `https://api.anthropic.com` 运行全部 probe 的真实记录。

## 测试元数据

| 字段 | 值 |
|---|---|
| 渠道 | Claude Console |
| 目标 | `https://api.anthropic.com` |
| 模型 | `claude-opus-4-5` |
| 时间 | 2026-05-15T01:23:31 |
| 总耗时 | 53465ms (53s) |
| 预估消耗 | input 153990 / output 30798 tokens, $1.5399 |
| 评分 | **D 58.09/100** |
| Checks | 69/91 passed |

## 分类得分

| 分类 | 得分 | 通过率 |
|---|---|---|
| LLM 指纹验证 | 20.588235294117645/25 | 14/17 |
| 结构完整性 | 20/25 | 28/35 |
| 签名校验 | 20/20 | 11/11 |
| 行为验证 | 7.5/20 | 3/8 |
| 多模态能力 | 0/10 | 0/2 |

## 逐 Probe 结果

### precheck — 基线流式探针 [PASS] (2139ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `headers` | PASS | 13/13 齐全 | all Anthropic ratelimit headers present |
| `request_id` | PASS | req_01 前缀 | Request-Id format OK: req_011Cb2r9QtrMNgs8... |
| `x_new_api_version` | PASS | 无 X-New-Api-Version 头 | no X-New-Api-Version header |
| `cf_headers` | PASS | 均存在 | Cloudflare-style headers present |
| `server_timing` | PASS | X-Envoy=1255 | X-Envoy-Upstream-Service-Time present: 1255 |
| `cf_ray_format` | PASS | 9fbb960ffc7eb74c-LAX | Cf-Ray format OK: 9fbb960ffc7eb74c-LAX |
| `cookie_domain` | PASS | anthropic.com 域 | cookie domain OK |
| `server_header` | PASS | cloudflare | Server=cloudflare OK |
| `sse_done` | PASS | no [DONE] sentinel | no [DONE] sentinel |
| `sse_event_order` | PASS | 7 events in correct order | 7 events, order OK |
| `sse_tailing` | PASS | double-newline endings (7) | double-newline endings (7) |
| `delta_usage_slim` | PASS | slim format (no bloat fields) | message_delta usage is slim format |
| `sse_ping_position` | PASS | ping after content_block_start | ping after content_block_start OK |
| `message_start_output_zero` | PASS | output_tokens=1 | output_tokens=1 OK |
| `container` | PASS | 无 container 字段 | no container field |
| `bedrock_state` | PASS | 无 bedrock_state 字段 | no bedrock_state |
| `cache_small_probe` | PASS | create=0 read=0 | cache values are zero for small probe |

### tag_replay — 全量指纹探针 [PASS] (5354ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `id_format` | PASS | msg_01 前缀 | msg_01 format OK |
| `model_name` | PASS | claude-opus-4-5-20251101 | model=claude-opus-4-5-20251101 (variant of requested) |
| `signature` | PASS | valid protobuf signature (external) | valid protobuf signature (external) |
| `signature_length` | PASS | 387 bytes | signature 387 bytes OK |
| `thinking_present` | PASS | found block type "thinking" | thinking block found |
| `usage_structure` | PASS | all present | usage structure OK |
| `field_order` | PASS | order correct | field order OK |
| `inference_geo` | PASS | not_available | inference_geo=not_available |
| `stop_details` | PASS | 存在 stop_details 字段 | stop_details present |
| `stop_details_structure` | PASS | stop_details 为 null（跳过） | stop_details null (skip) |
| `stop_reason` | PASS | end_turn | stop_reason=end_turn |
| `stop_sequence_null` | PASS | null | stop_sequence=null OK |
| `thinking_order` | PASS | thinking block at index 0 | thinking at index 0 |
| `thinking_display_omitted` | PASS | model=claude-opus-4-5 (not opus-4-7+) | not opus-4-7+ model (skip) |
| `tag_replay` | PASS | tag found in response | tag found in response: c9046c653438d0c3 |
| `cache_fake` | PASS | create=25085 read=0 | cache values non-zero: create=25085 read=0 |
| `message_start_usage` | PASS | input_tokens=579 | message_start usage OK: input_tokens=579 |
| `sse_ping_position` | PASS | ping after content_block_start | ping after content_block_start OK |
| `service_tier` | PASS | 存在 service_tier 字段 | service_tier present |
| `signature_type_leak` | PASS | 无 signature_type 字段 | no signature_type leak |
| `usage_fields_complete` | PASS | all 7 fields present | usage has all 7 fields |
| `cache_creation_complete` | PASS | 两者均存在 | both ephemeral fields present |

### mini_probe — 极简非流式探针 [PASS] (1955ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `backend_type` | PASS | msg_01 前缀 | official format (msg_01) |
| `small_output_tokens` | PASS | 1 | output_tokens=1 OK |
| `small_stop_reason` | PASS | max_tokens | stop_reason=max_tokens OK |
| `small_ephemeral_zero` | PASS | 5m=6295 1h=0 | ephemeral_5m=6295 ephemeral_1h=0 (cache_control used) |
| `small_cache_zero` | PASS | create=6295 read=0 | cache create=6295 read=0 (cache_control used) |
| `token_budget` | PASS | input_tokens=7 | input_tokens=7 within budget (≤80) |

### identity_probe — 身份识别探针 [PARTIAL 0/15] (754ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 400: invalid_request_error | adaptive thinking is not supported on this model |
| `nonstream_fields` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `nonstream_type` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `nonstream_role` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `field_order` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `body_key_order` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `id_format` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `identity_response` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `identity_no_leak` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `identity_platform` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `poison_answer` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `stop_sequence_null` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `service_tier` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `signature_type_leak` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |
| `usage_fields_complete` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |

### self_intro — 结构化自述探针 [PARTIAL 5/6] (6781ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `no_thinking_leak` | PASS | no thinking blocks found | no unexpected thinking block |
| `identity_response` | FAIL | neither Claude nor Anthropic found | response does not mention Claude or Anthropic |
| `structured_json_valid` | PASS | JSON 合法 | valid JSON response |
| `structured_schema_match` | PASS | JSON 字段与 schema 匹配 | schema match: {name, title, desc} OK |
| `structured_name_correct` | PASS | 包含 "David Petrov" | name correct: David Petrov |
| `structured_stop_reason` | PASS | end_turn | stop_reason=end_turn OK |

### tool_use — 工具调用探针 [PASS] (12397ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `tool_use_id` | PASS | srvtoolu_01 前缀 | srvtoolu_01 format OK |
| `tool_stop_reason` | PASS | end_turn | tool content present with stop_reason=end_turn |
| `tool_forced_compliance` | PASS | tool_use block present | tool_use block present |
| `web_search_result` | PASS | 结构完整 | web_search_tool_result structure OK |
| `server_tool_type` | PASS | server_tool_use | web_search uses server_tool_use |
| `citations_present` | PASS | citations 存在 | citations present in text blocks |
| `server_tool_usage` | PASS | map[web_fetch_requests:0 web_search_requests:1] | server_tool_use present: map[web_fetch_requests:0 web_search_requests:1] |

### logic — 逻辑推理探针 [PARTIAL 0/2] (500ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 400: invalid_request_error | adaptive thinking is not supported on this model |
| `logic_answer` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |

### hidden_prompt — 隐藏 Prompt 检测 [PASS] (2030ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `hidden_prompt` | PASS | input_tokens=8 | input_tokens=8 (clean, no hidden prompt) |

### image_ocr — 图片 OCR 探针 [PARTIAL 0/2] (441ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 400: invalid_request_error | adaptive thinking is not supported on this model |
| `image_ocr` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |

### pdf_extract — PDF 提取探针 [PARTIAL 0/2] (513ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 400: invalid_request_error | adaptive thinking is not supported on this model |
| `pdf_extract` | FAIL | 跳过 (HTTP 400) | skipped: invalid_request_error |

### magic_refusal — 拒答字符串探针 [PASS] (2442ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `magic_refusal` | PASS | end_turn | stop_reason=end_turn (部分渠道/tier 不触发 refusal) |

### effort_thinking — Effort 级别思考探针 [PASS] (13977ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `effort_high_thinking` | PASS | 2 content blocks | effort 请求被接受，响应含有效 content (含 thinking 块) |
| `effort_high_signature` | PASS | 1435 bytes | signature 有效 (1435 bytes) |
| `effort_medium_no_think` | PASS | thinking (40 字符) | content[0].type = "thinking" (40 字符 thinking) |
| `effort_low_no_think` | PASS | text | content[0].type = "text" (thinking 已跳过) |

### signature_reject — 空签名拒绝探针 [PASS] (286ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `signature_empty_rejected` | PASS | 400 | 状态码 400, 空 signature 被正确拒绝 |

### bash_tool — Bash 工具探针 [PASS] (2075ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `bash_stop_reason` | PASS | tool_use | stop_reason=tool_use OK |
| `bash_tool_name` | PASS | name="bash" | content 包含 name="bash" 的 tool_use 块 |
| `bash_tool_rejected` | PASS | 400 | 状态码 400, 非法 bash tool name 被正确拒绝 |

### minimal_tokens — 最小 Token 计费探针 [PASS] (1813ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `minimal_input_tokens` | PASS | 8 | input_tokens=8 (范围 8-16 OK) |
| `minimal_output_tokens` | PASS | 4 | output_tokens=4 (范围 1-4 OK) |

## 失败 Checks

| Check | 说明 |
|---|---|
| `api_error` | adaptive thinking is not supported on this model |
| `nonstream_fields` | skipped: invalid_request_error |
| `nonstream_type` | skipped: invalid_request_error |
| `nonstream_role` | skipped: invalid_request_error |
| `field_order` | skipped: invalid_request_error |
| `body_key_order` | skipped: invalid_request_error |
| `id_format` | skipped: invalid_request_error |
| `identity_response` | skipped: invalid_request_error |
| `identity_no_leak` | skipped: invalid_request_error |
| `identity_platform` | skipped: invalid_request_error |
| `poison_answer` | skipped: invalid_request_error |
| `stop_sequence_null` | skipped: invalid_request_error |
| `service_tier` | skipped: invalid_request_error |
| `signature_type_leak` | skipped: invalid_request_error |
| `usage_fields_complete` | skipped: invalid_request_error |
| `identity_response` | response does not mention Claude or Anthropic |
| `api_error` | adaptive thinking is not supported on this model |
| `logic_answer` | skipped: invalid_request_error |
| `api_error` | adaptive thinking is not supported on this model |
| `image_ocr` | skipped: invalid_request_error |
| `api_error` | adaptive thinking is not supported on this model |
| `pdf_extract` | skipped: invalid_request_error |

---

*生成时间: 2026-05-15T01:23:31*
