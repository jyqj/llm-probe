# Claude Console — claude-sonnet-4-6 基线

> 渠道 `Claude Console` 使用 `https://api.anthropic.com` 运行全部 probe 的真实记录。

## 测试元数据

| 字段 | 值 |
|---|---|
| 渠道 | Claude Console |
| 目标 | `https://api.anthropic.com` |
| 模型 | `claude-sonnet-4-6` |
| 时间 | 2026-05-15T01:19:15 |
| 总耗时 | 122816ms (122s) |
| 预估消耗 | input 153990 / output 30798 tokens, $0.9239 |
| 评分 | **A+ 97.5/100** |
| Checks | 87/88 passed |

## 分类得分

| 分类 | 得分 | 通过率 |
|---|---|---|
| LLM 指纹验证 | 25/25 | 17/17 |
| 结构完整性 | 25/25 | 35/35 |
| 签名校验 | 20/20 | 12/12 |
| 行为验证 | 17.5/20 | 7/8 |
| 多模态能力 | 10/10 | 2/2 |

## 逐 Probe 结果

### precheck — 基线流式探针 [PASS] (10963ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `headers` | PASS | 13/13 齐全 | all Anthropic ratelimit headers present |
| `request_id` | PASS | req_01 前缀 | Request-Id format OK: req_011Cb2qk7SEK25if... |
| `x_new_api_version` | PASS | 无 X-New-Api-Version 头 | no X-New-Api-Version header |
| `cf_headers` | PASS | 均存在 | Cloudflare-style headers present |
| `server_timing` | PASS | X-Envoy=837 | X-Envoy-Upstream-Service-Time present: 837 |
| `cf_ray_format` | PASS | 9fbb8e58c862999a-LAX | Cf-Ray format OK: 9fbb8e58c862999a-LAX |
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

### tag_replay — 全量指纹探针 [PASS] (4609ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `id_format` | PASS | msg_01 前缀 | msg_01 format OK |
| `model_name` | PASS | claude-sonnet-4-6 | model=claude-sonnet-4-6 |
| `signature` | PASS | valid protobuf signature (external) | valid protobuf signature (external) |
| `signature_length` | PASS | 617 bytes | signature 617 bytes OK |
| `thinking_present` | PASS | found block type "thinking" | thinking block found |
| `usage_structure` | PASS | all present | usage structure OK |
| `field_order` | PASS | order correct | field order OK |
| `inference_geo` | PASS | global | inference_geo=global |
| `stop_details` | PASS | 存在 stop_details 字段 | stop_details present |
| `stop_details_structure` | PASS | stop_details 为 null（跳过） | stop_details null (skip) |
| `stop_reason` | PASS | end_turn | stop_reason=end_turn |
| `stop_sequence_null` | PASS | null | stop_sequence=null OK |
| `thinking_order` | PASS | thinking block at index 0 | thinking at index 0 |
| `thinking_display_omitted` | PASS | model=claude-sonnet-4-6 (not opus-4-7+) | not opus-4-7+ model (skip) |
| `tag_replay` | PASS | tag found in response | tag found in response: 6f9435e3a42dd3ee |
| `cache_fake` | PASS | create=0 read=24909 | cache values non-zero: create=0 read=24909 |
| `message_start_usage` | PASS | input_tokens=402 | message_start usage OK: input_tokens=402 |
| `sse_ping_position` | PASS | ping after content_block_start | ping after content_block_start OK |
| `service_tier` | PASS | 存在 service_tier 字段 | service_tier present |
| `signature_type_leak` | PASS | 无 signature_type 字段 | no signature_type leak |
| `usage_fields_complete` | PASS | all 7 fields present | usage has all 7 fields |
| `cache_creation_complete` | PASS | 两者均存在 | both ephemeral fields present |

### mini_probe — 极简非流式探针 [PASS] (1374ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `backend_type` | PASS | msg_01 前缀 | official format (msg_01) |
| `small_output_tokens` | PASS | 1 | output_tokens=1 OK |
| `small_stop_reason` | PASS | max_tokens | stop_reason=max_tokens OK |
| `small_ephemeral_zero` | PASS | 5m=4 1h=0 | ephemeral_5m=4 ephemeral_1h=0 (cache_control used) |
| `small_cache_zero` | PASS | create=4 read=6296 | cache create=4 read=6296 (cache_control used) |
| `token_budget` | PASS | input_tokens=3 | input_tokens=3 within budget (≤80) |

### identity_probe — 身份识别探针 [PASS] (14139ms)

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

### self_intro — 结构化自述探针 [PARTIAL 5/6] (2257ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `no_thinking_leak` | PASS | no thinking blocks found | no unexpected thinking block |
| `identity_response` | FAIL | neither Claude nor Anthropic found | response does not mention Claude or Anthropic |
| `structured_json_valid` | PASS | JSON 合法 | valid JSON response |
| `structured_schema_match` | PASS | JSON 字段与 schema 匹配 | schema match: {name, title, desc} OK |
| `structured_name_correct` | PASS | 包含 "Alice Carter" | name correct: Alice Carter |
| `structured_stop_reason` | PASS | end_turn | stop_reason=end_turn OK |

### tool_use — 工具调用探针 [PASS] (24875ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `tool_use_id` | PASS | srvtoolu_01 前缀 | srvtoolu_01 format OK |
| `tool_stop_reason` | PASS | end_turn | tool content present with stop_reason=end_turn |
| `tool_forced_compliance` | PASS | tool_use block present | tool_use block present |
| `web_search_result` | PASS | 结构完整 | web_search_tool_result structure OK |
| `server_tool_type` | PASS | server_tool_use | web_search uses server_tool_use |
| `citations_present` | PASS | citations 存在 | citations present in text blocks |
| `server_tool_usage` | PASS | map[web_fetch_requests:0 web_search_requests:1] | server_tool_use present: map[web_fetch_requests:0 web_search_requests:1] |

### logic — 逻辑推理探针 [PASS] (19851ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `logic_answer` | PASS | found keyword: 热 | contains heat/warm method |

### hidden_prompt — 隐藏 Prompt 检测 [PASS] (725ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `hidden_prompt` | PASS | input_tokens=8 | input_tokens=8 (clean, no hidden prompt) |

### image_ocr — 图片 OCR 探针 [PASS] (1815ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `image_ocr` | PASS | 包含 99KELPA2 | image OCR correct: 99KELPA2 |

### pdf_extract — PDF 提取探针 [PASS] (1916ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `pdf_extract` | PASS | 包含 XAWFV9XT | PDF text correct: XAWFV9XT |

### magic_refusal — 拒答字符串探针 [PASS] (2752ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `magic_refusal` | PASS | end_turn | stop_reason=end_turn (部分渠道/tier 不触发 refusal) |

### effort_thinking — Effort 级别思考探针 [PASS] (34998ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `effort_high_thinking` | PASS | 2 content blocks | effort 请求被接受，响应含有效 content (含 thinking 块) |
| `effort_high_signature` | PASS | 1298 bytes | signature 有效 (1298 bytes) |
| `effort_medium_no_think` | PASS | thinking (1 字符) | content[0].type = "thinking" (1 字符 thinking) |
| `effort_low_no_think` | PASS | text | content[0].type = "text" (thinking 已跳过) |
| `effort_max_thinking` | PASS | 2 content blocks | effort=max 请求被接受 (含 thinking 块) |

### signature_reject — 空签名拒绝探针 [PASS] (263ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `signature_empty_rejected` | PASS | 400 | 状态码 400, 空 signature 被正确拒绝 |

### bash_tool — Bash 工具探针 [PASS] (1458ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `bash_stop_reason` | PASS | tool_use | stop_reason=tool_use OK |
| `bash_tool_name` | PASS | name="bash" | content 包含 name="bash" 的 tool_use 块 |
| `bash_tool_rejected` | PASS | 400 | 状态码 400, 非法 bash tool name 被正确拒绝 |

### minimal_tokens — 最小 Token 计费探针 [PASS] (812ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `minimal_input_tokens` | PASS | 8 | input_tokens=8 (范围 8-16 OK) |
| `minimal_output_tokens` | PASS | 4 | output_tokens=4 (范围 1-4 OK) |

## 失败 Checks

| Check | 说明 |
|---|---|
| `identity_response` | response does not mention Claude or Anthropic |

---

*生成时间: 2026-05-15T01:19:15*
