# image_ocr：图片 OCR 探针

- 代码：`internal/channeltest/phase_image_ocr.go`
- Tags：`heavy`
- EstTokens：25000

## 检测目的

生成带随机文本的 PNG 图片，让模型只返回图片中的文本，用于检查目标渠道是否真实支持 Claude vision 输入及基础 OCR 能力。

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
      {"type":"image", "source":{"type":"base64", "media_type":"image/png", "data":"<generated image>"}},
      {"type":"text", "text":"What does the text in the picture say? ..."}
    ]
  }]
}
```

## 产出 checks

`image_ocr`

## 检测依据

- 官方 vision 文档可支撑图片 content block 与 base64 source 的公共形态：<https://docs.anthropic.com/en/docs/build-with-claude/images-and-vision>。
- 随机 OCR 文本是本项目防模板/防缓存 heuristic。

## 误报/例外

- 小图渲染质量、字体、压缩、模型视觉能力都可能影响识别。
- 部分模型/账号/渠道可能不支持 vision 或被禁用，不应和文本渠道真实性混为一谈。
- 该 probe 成本高，不适合默认 monitor。

## 本地优化建议

- 对不同模型维护单独 OCR 难度与容错规则。
- 保存生成文本和图片参数，方便复现。
- 如果只是长期渠道可用性监控，不要启用该 heavy probe。
