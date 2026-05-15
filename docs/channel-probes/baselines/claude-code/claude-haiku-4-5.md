# Claude Code — claude-haiku-4-5 基线

> 渠道 `Claude Code` 使用 `http://38.34.191.113:8080` 运行全部 probe 的真实记录。

## 测试元数据

| 字段 | 值 |
|---|---|
| 渠道 | Claude Code |
| 目标 | `http://38.34.191.113:8080` |
| 模型 | `claude-haiku-4-5` |
| 时间 | 2026-05-15T10:44:33 |
| 总耗时 | 5729ms (5s) |
| 预估消耗 | input 153890 / output 30778 tokens, $0.3078 |
| 评分 | **F 0/100** |
| Checks | 0/94 passed |
| 跳过 Probes | signature_reject |

## 分类得分

| 分类 | 得分 | 通过率 |
|---|---|---|
| LLM 指纹验证 | 0/25 | 0/17 |
| 结构完整性 | 0/25 | 0/35 |
| 签名校验 | 0/20 | 0/6 |
| 行为验证 | 0/20 | 0/8 |
| 多模态能力 | 0/10 | 0/2 |

## 逐 Probe 结果

### precheck — 基线流式探针 [PARTIAL 0/18] (745ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `headers` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `request_id` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `x_new_api_version` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `cf_headers` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `server_timing` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `cf_ray_format` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `cookie_domain` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `server_header` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `sse_done` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `sse_event_order` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `sse_tailing` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `delta_usage_slim` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `sse_ping_position` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `message_start_output_zero` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `container` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `bedrock_state` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `cache_small_probe` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### tag_replay — 全量指纹探针 [PARTIAL 0/23] (778ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `id_format` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `model_name` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `signature` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `signature_length` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `thinking_present` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `usage_structure` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `field_order` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `inference_geo` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `stop_details` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `stop_details_structure` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `stop_reason` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `stop_sequence_null` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `thinking_order` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `thinking_display_omitted` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `tag_replay` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `cache_fake` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `message_start_usage` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `sse_ping_position` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `service_tier` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `signature_type_leak` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `usage_fields_complete` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `cache_creation_complete` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### mini_probe — 极简非流式探针 [PARTIAL 0/7] (350ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `backend_type` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `small_output_tokens` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `small_stop_reason` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `small_ephemeral_zero` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `small_cache_zero` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `token_budget` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### identity_probe — 身份识别探针 [PARTIAL 0/15] (494ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `nonstream_fields` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `nonstream_type` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `nonstream_role` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `field_order` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `body_key_order` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `id_format` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `identity_response` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `identity_no_leak` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `identity_platform` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `poison_answer` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `stop_sequence_null` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `service_tier` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `signature_type_leak` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `usage_fields_complete` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### self_intro — 结构化自述探针 [PARTIAL 0/7] (458ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `no_thinking_leak` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `identity_response` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `structured_json_valid` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `structured_schema_match` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `structured_name_correct` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `structured_stop_reason` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### tool_use — 工具调用探针 [PARTIAL 0/8] (292ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `tool_use_id` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `tool_stop_reason` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `tool_forced_compliance` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `web_search_result` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `server_tool_type` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `citations_present` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `server_tool_usage` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### logic — 逻辑推理探针 [PARTIAL 0/2] (375ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `logic_answer` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### hidden_prompt — 隐藏 Prompt 检测 [PARTIAL 0/2] (299ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `hidden_prompt` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### image_ocr — 图片 OCR 探针 [PARTIAL 0/2] (392ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `image_ocr` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### pdf_extract — PDF 提取探针 [PARTIAL 0/2] (377ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `pdf_extract` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### magic_refusal — 拒答字符串探针 [PARTIAL 0/2] (294ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `magic_refusal` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

### effort_thinking — Effort 级别思考探针 [PASS] (0ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|

### bash_tool — Bash 工具探针 [PARTIAL 0/3] (581ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `bash_stop_reason` | FAIL | 请求失败 | HTTP 503 (api_error): No available accounts: no available accounts |
| `bash_tool_name` | FAIL | 请求失败 | HTTP 503 (api_error): No available accounts: no available accounts |
| `bash_tool_rejected` | FAIL | 503 | 状态码 503, 期望 400 |

### minimal_tokens — 最小 Token 计费探针 [PARTIAL 0/3] (289ms)

| Check | 状态 | 实际值 | 详情 |
|---|---|---|---|
| `api_error` | FAIL | HTTP 503: api_error | No available accounts: no available accounts |
| `minimal_input_tokens` | FAIL | 跳过 (HTTP 503) | skipped: api_error |
| `minimal_output_tokens` | FAIL | 跳过 (HTTP 503) | skipped: api_error |

## 失败 Checks

| Check | 说明 |
|---|---|
| `api_error` | No available accounts: no available accounts |
| `headers` | skipped: api_error |
| `request_id` | skipped: api_error |
| `x_new_api_version` | skipped: api_error |
| `cf_headers` | skipped: api_error |
| `server_timing` | skipped: api_error |
| `cf_ray_format` | skipped: api_error |
| `cookie_domain` | skipped: api_error |
| `server_header` | skipped: api_error |
| `sse_done` | skipped: api_error |
| `sse_event_order` | skipped: api_error |
| `sse_tailing` | skipped: api_error |
| `delta_usage_slim` | skipped: api_error |
| `sse_ping_position` | skipped: api_error |
| `message_start_output_zero` | skipped: api_error |
| `container` | skipped: api_error |
| `bedrock_state` | skipped: api_error |
| `cache_small_probe` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `id_format` | skipped: api_error |
| `model_name` | skipped: api_error |
| `signature` | skipped: api_error |
| `signature_length` | skipped: api_error |
| `thinking_present` | skipped: api_error |
| `usage_structure` | skipped: api_error |
| `field_order` | skipped: api_error |
| `inference_geo` | skipped: api_error |
| `stop_details` | skipped: api_error |
| `stop_details_structure` | skipped: api_error |
| `stop_reason` | skipped: api_error |
| `stop_sequence_null` | skipped: api_error |
| `thinking_order` | skipped: api_error |
| `thinking_display_omitted` | skipped: api_error |
| `tag_replay` | skipped: api_error |
| `cache_fake` | skipped: api_error |
| `message_start_usage` | skipped: api_error |
| `sse_ping_position` | skipped: api_error |
| `service_tier` | skipped: api_error |
| `signature_type_leak` | skipped: api_error |
| `usage_fields_complete` | skipped: api_error |
| `cache_creation_complete` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `backend_type` | skipped: api_error |
| `small_output_tokens` | skipped: api_error |
| `small_stop_reason` | skipped: api_error |
| `small_ephemeral_zero` | skipped: api_error |
| `small_cache_zero` | skipped: api_error |
| `token_budget` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `nonstream_fields` | skipped: api_error |
| `nonstream_type` | skipped: api_error |
| `nonstream_role` | skipped: api_error |
| `field_order` | skipped: api_error |
| `body_key_order` | skipped: api_error |
| `id_format` | skipped: api_error |
| `identity_response` | skipped: api_error |
| `identity_no_leak` | skipped: api_error |
| `identity_platform` | skipped: api_error |
| `poison_answer` | skipped: api_error |
| `stop_sequence_null` | skipped: api_error |
| `service_tier` | skipped: api_error |
| `signature_type_leak` | skipped: api_error |
| `usage_fields_complete` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `no_thinking_leak` | skipped: api_error |
| `identity_response` | skipped: api_error |
| `structured_json_valid` | skipped: api_error |
| `structured_schema_match` | skipped: api_error |
| `structured_name_correct` | skipped: api_error |
| `structured_stop_reason` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `tool_use_id` | skipped: api_error |
| `tool_stop_reason` | skipped: api_error |
| `tool_forced_compliance` | skipped: api_error |
| `web_search_result` | skipped: api_error |
| `server_tool_type` | skipped: api_error |
| `citations_present` | skipped: api_error |
| `server_tool_usage` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `logic_answer` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `hidden_prompt` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `image_ocr` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `pdf_extract` | skipped: api_error |
| `api_error` | No available accounts: no available accounts |
| `magic_refusal` | skipped: api_error |
| `bash_stop_reason` | HTTP 503 (api_error): No available accounts: no available accounts |
| `bash_tool_name` | HTTP 503 (api_error): No available accounts: no available accounts |
| `bash_tool_rejected` | 状态码 503, 期望 400 |
| `api_error` | No available accounts: no available accounts |
| `minimal_input_tokens` | skipped: api_error |
| `minimal_output_tokens` | skipped: api_error |

---

*生成时间: 2026-05-15T10:44:33*
