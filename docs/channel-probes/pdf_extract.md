# pdf_extract：PDF 提取探针

- 代码：`internal/channeltest/phase_pdf_extract.go`
- Tags：`heavy`
- EstTokens：25000

## 检测目的

生成包含随机文本的 PDF，让模型只返回 PDF 中的文本，用于检查 document/PDF 输入支持与基础文本提取能力。

## 请求形态

```jsonc
POST {base_url}/v1/messages
body: {
  "model": "<model>",
  "max_tokens": 1024,
  "stream": true,
  "thinking": {"type":"adaptive"},
  "system": "<fullSystem>",
  "tools": "<data.Tools>",
  "metadata": "<genMetadata>",
  "messages": [{
    "role":"user",
    "content":[
      {"type":"document", "source":{"type":"base64", "media_type":"application/pdf", "data":"<generated pdf>"}},
      {"type":"text", "text":"What text does this PDF contain? ..."}
    ]
  }]
}
```

## 产出 checks

`pdf_extract`

## 检测依据

- 官方 PDF support 文档可支撑 document/PDF 输入的公共形态：<https://docs.anthropic.com/en/docs/build-with-claude/pdf-support>。
- 随机 PDF 文本是本地防模板 heuristic。

## 误报/例外

- PDF 支持可能依赖模型、账号权限和 beta/版本策略。
- PDF 渲染/解析失败可能来自生成器或中间代理限制，需保存原始 PDF 复现。
- 该 probe 高成本，不应作为默认长期检测面。

## 本地优化建议

- 保存 pdfText 和生成参数，给历史详情页展示。
- 增加“模型/渠道不支持 PDF”与“支持但提取错误”的分类。
- 低频专项跑即可，不进默认 monitor。
