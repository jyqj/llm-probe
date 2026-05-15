# Claude Console — Intelligence Benchmark 基线

> SWE-Atlas-QnA 124 题，按模型 × 思考力度 (effort) 全量测试记录。
> 渠道: Claude Console (`https://api.anthropic.com`)

## 总览

| 模型 | Effort | 得分 | 通过率 | 完成 | 错误 | 耗时 |
|---|---|---|---|---|---|---|
| claude-haiku-4-5 | default | 3.0 | 13.8% | 124/124 | 1 | 81m |
| claude-sonnet-4-6 | default | 2.8 | 14.6% | 124/124 | 76 | 311m |

## 模型 × Effort 对比

### claude-haiku-4-5

| Effort | 得分 | 通过率 | 评估数 | 通过数 | 错误数 | 耗时 |
|---|---|---|---|---|---|---|
| default | 3.0 | 13.8% | 123 | 17 | 1 | 81m |

### claude-sonnet-4-6

| Effort | 得分 | 通过率 | 评估数 | 通过数 | 错误数 | 耗时 |
|---|---|---|---|---|---|---|
| default | 2.8 | 14.6% | 48 | 7 | 76 | 311m |

## 逐题得分矩阵

| # | Task ID | Lang | Category | haiku45_def | sonnet46_def |
|---|---|---|---|---|---|
| 1 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 2 | `6905333b74f2` | ts | Code Onboarding | 0.0 | 0.0 |
| 3 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | ERR |
| 4 | `6905333b74f2` | go | Security | 0.0 | ERR |
| 5 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 6 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 7 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 8 | `6905333b74f2` | c | Root-cause analysis | 0.0 | 0.0 |
| 9 | `6905333b74f2` | go | Architecture & syste | 0.20 | 0.0 |
| 10 | `6905333b74f2` | python | Code Onboarding | 0.0 | 0.11 |
| 11 | `6905333b74f2` | go | Architecture & syste | 0.0 | ERR |
| 12 | `6905333b74f2` | go | Root-cause analysis | 0.0 | 0.0 |
| 13 | `6905333b74f2` | ts | Architecture & syste | 0.29 | 0.43 |
| 14 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 15 | `6905333b74f2` | c | Architecture & syste | 0.05 | ERR |
| 16 | `6905333b74f2` | python | Architecture & syste | 0.0 | 0.0 |
| 17 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 18 | `6905333b74f2` | go | Security | 0.08 | ERR |
| 19 | `6905333b74f2` | go | Code Onboarding | 0.0 | 0.0 |
| 20 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 21 | `6905333b74f2` | go | Architecture & syste | 0.17 | ERR |
| 22 | `6905333b74f2` | go | Code Onboarding | 0.33 | 0.22 |
| 23 | `6905333b74f2` | python | Code Onboarding | 0.0 | ERR |
| 24 | `6905333b74f2` | ts | Architecture & syste | 0.0 | ERR |
| 25 | `6905333b74f2` | python | Root-cause analysis | 0.0 | 0.0 |
| 26 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 27 | `6905333b74f2` | python | Architecture & syste | 0.0 | ERR |
| 28 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | ERR |
| 29 | `6905333b74f2` | python | Architecture & syste | 0.0 | 0.14 |
| 30 | `6905333b74f2` | python | API & library usage  | 0.0 | 0.0 |
| 31 | `6905333b74f2` | ts | Code Onboarding | 0.12 | 0.0 |
| 32 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 33 | `6905333b74f2` | go | Architecture & syste | ERR | 0.0 |
| 34 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | 0.0 |
| 35 | `6905333b74f2` | go | Code Onboarding | 0.0 | ERR |
| 36 | `6905333b74f2` | c | Architecture & syste | 0.0 | 0.0 |
| 37 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | ERR |
| 38 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 39 | `6905333b74f2` | ts | Code Onboarding | 0.0 | ERR |
| 40 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 41 | `6905333b74f2` | python | Architecture & syste | 0.0 | 0.0 |
| 42 | `6905333b74f2` | python | Architecture & syste | 0.0 | 0.0 |
| 43 | `6905333b74f2` | python | Architecture & syste | 0.0 | ERR |
| 44 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | 0.0 |
| 45 | `6905333b74f2` | c | Architecture & syste | 0.0 | 0.0 |
| 46 | `6905333b74f2` | go | Architecture & syste | 0.0 | ERR |
| 47 | `6905333b74f2` | go | Code Onboarding | 0.0 | 0.0 |
| 48 | `6905333b74f2` | python | Architecture & syste | 0.20 | ERR |
| 49 | `6905333b74f2` | go | Code Onboarding | 0.0 | 0.0 |
| 50 | `6905333b74f2` | python | Architecture & syste | 0.0 | 0.0 |
| 51 | `6905333b74f2` | go | Security | 0.50 | ERR |
| 52 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 53 | `6905333b74f2` | ts | Code Onboarding | 0.0 | 0.0 |
| 54 | `6905333b74f2` | python | Root-cause analysis | 0.0 | ERR |
| 55 | `6905333b74f2` | ts | Code Onboarding | 0.0 | 0.0 |
| 56 | `6905333b74f2` | go | Security | 0.0 | 0.0 |
| 57 | `6905333b74f2` | python | Architecture & syste | 0.0 | ERR |
| 58 | `6905333b74f2` | go | Root-cause analysis | 0.14 | ERR |
| 59 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 60 | `6905333b74f2` | python | API & library usage  | 0.0 | 0.0 |
| 61 | `6905333b74f2` | ts | Architecture & syste | 0.0 | 0.0 |
| 62 | `6905333b74f2` | go | Code Onboarding | 0.0 | 0.0 |
| 63 | `6905333b74f2` | python | Root-cause analysis | 0.0 | ERR |
| 64 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 65 | `6905333b74f2` | go | Root-cause analysis | 0.0 | 0.0 |
| 66 | `6905333b74f2` | python | Architecture & syste | 0.0 | ERR |
| 67 | `6905333b74f2` | python | Root-cause analysis | 0.0 | ERR |
| 68 | `6905333b74f2` | ts | Code Onboarding | 0.12 | ERR |
| 69 | `6905333b74f2` | c | Code Onboarding | 0.0 | 0.0 |
| 70 | `6905333b74f2` | go | Code Onboarding | 0.0 | 0.0 |
| 71 | `6905333b74f2` | python | Code Onboarding | 0.0 | ERR |
| 72 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | ERR |
| 73 | `6905333b74f2` | python | API & library usage  | 0.0 | ERR |
| 74 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 75 | `6905333b74f2` | ts | Code Onboarding | 0.0 | 0.0 |
| 76 | `6905333b74f2` | python | Architecture & syste | 0.0 | ERR |
| 77 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 78 | `6905333b74f2` | go | Code Onboarding | 0.0 | ERR |
| 79 | `6905333b74f2` | ts | Root-cause analysis | 0.43 | 0.17 |
| 80 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 81 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 82 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | ERR |
| 83 | `6905333b74f2` | python | Architecture & syste | 0.0 | ERR |
| 84 | `6905333b74f2` | ts | Root-cause analysis | 0.07 | 0.0 |
| 85 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 86 | `6905333b74f2` | python | API & library usage  | 0.0 | 0.0 |
| 87 | `6905333b74f2` | go | Root-cause analysis | 0.0 | 0.0 |
| 88 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 89 | `6905333b74f2` | python | Code Onboarding | 0.0 | 0.0 |
| 90 | `6905333b74f2` | c | Architecture & syste | 0.0 | 0.0 |
| 91 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 92 | `6905333b74f2` | python | Root-cause analysis | 0.0 | ERR |
| 93 | `6905333b74f2` | c | Root-cause analysis | 0.0 | ERR |
| 94 | `6905333b74f2` | ts | Code Onboarding | 0.23 | 0.0 |
| 95 | `6905333b74f2` | go | Code Onboarding | 0.0 | ERR |
| 96 | `6905333b74f2` | ts | Code Onboarding | 0.07 | 0.07 |
| 97 | `6905333b74f2` | python | Architecture & syste | 0.0 | ERR |
| 98 | `6905333b74f2` | ts | Architecture & syste | 0.0 | ERR |
| 99 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | 0.0 |
| 100 | `6905333b74f2` | ts | Root-cause analysis | 0.0 | ERR |
| 101 | `6905333b74f2` | go | Security | 0.0 | ERR |
| 102 | `6905333b74f2` | ts | Code Onboarding | 0.0 | ERR |
| 103 | `6905333b74f2` | ts | Code Onboarding | 0.0 | 0.0 |
| 104 | `6905333b74f2` | python | Architecture & syste | 0.0 | ERR |
| 105 | `6905333b74f2` | go | Code Onboarding | 0.0 | 0.0 |
| 106 | `6905333b74f2` | ts | Security | 0.0 | 0.0 |
| 107 | `6905333b74f2` | c | Architecture & syste | 0.14 | 0.21 |
| 108 | `6905333b74f2` | c | Root-cause analysis | 0.0 | ERR |
| 109 | `6905333b74f2` | go | Security | 0.0 | ERR |
| 110 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 111 | `6905333b74f2` | go | Security | 0.0 | 0.0 |
| 112 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 113 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 114 | `6905333b74f2` | go | Security | 0.0 | ERR |
| 115 | `6905333b74f2` | c | Code Onboarding | 0.0 | 0.0 |
| 116 | `6905333b74f2` | python | Root-cause analysis | 0.0 | ERR |
| 117 | `6905333b74f2` | ts | Security | 0.0 | ERR |
| 118 | `6905333b74f2` | go | Root-cause analysis | 0.0 | ERR |
| 119 | `6905333b74f2` | ts | Security | 0.0 | ERR |
| 120 | `6905333b74f2` | c | Architecture & syste | 0.0 | ERR |
| 121 | `6905333b74f2` | ts | Code Onboarding | 0.62 | ERR |
| 122 | `6905333b74f2` | ts | Architecture & syste | 0.0 | ERR |
| 123 | `6905333b74f2` | python | Root-cause analysis | 0.0 | ERR |
| 124 | `6905333b74f2` | c | Root-cause analysis | 0.0 | ERR |

---

*共 2 组运行，124 道题*
