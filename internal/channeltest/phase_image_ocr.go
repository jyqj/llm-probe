package channeltest

import "detector-service/internal/channeltest/data"

var probeImageOCR = &Probe{
	ID: "image_ocr", Label: "图片 OCR 探针",
	Tags:       []string{"heavy"},
	EstTokens:  25000,
	Checks:     []string{"image_ocr"},
	OnlyModels: []string{"claude-sonnet", "claude-opus"},
	Run:        (*Runner).runImageOCR,
}

func (p *Runner) runImageOCR(targetBase, targetKey, model string) ([]CheckResult, error) {
	ocrText := data.RandomOCRText(8)
	imgB64 := data.GenTestImageBase64(ocrText)

	req := map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"stream":     true,
		"system":     fullSystem(),
		"tools":      data.Tools(),
		"metadata":   genMetadata(),
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "image/png",
							"data":       imgB64,
						},
					},
					map[string]any{
						"type": "text",
						"text": "What does the text in the picture say? Reply with ONLY the text, nothing else. Do not use any tools.",
					},
				},
			},
		},
	}
	if tp := ThinkingParam(model); tp != nil {
		req["thinking"] = tp
	}
	body := toJSON(req)

	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sse, start, delta := readSSE(resp.Body)
	full := merge(start, delta, sse)
	p.recordStreamResult(full)
	if full == nil {
		return []CheckResult{{Name: "image_ocr", Pass: false, Detail: "parse failed"}}, nil
	}

	return []CheckResult{checkImageOCR(full, ocrText)}, nil
}
