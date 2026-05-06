package channeltest

import "detector-service/internal/channeltest/data"

// ════════════════════════════════════════════════════════════
//  Phase 2i: pdf_extract
//  cctest 07_pdf_extract: thinking=adaptive, 28 tools, full system,
//  max_tokens=1024, stream=true, document + text content
// ════════════════════════════════════════════════════════════

func (p *Runner) runPDFExtract(targetBase, targetKey, model string) ([]CheckResult, error) {
	pdfText := data.RandomOCRText(8)
	pdfB64 := data.GenTestPDFBase64(pdfText)

	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"stream":     true,
		"thinking":   map[string]any{"type": "adaptive"},
		"system":     fullSystem(),
		"tools":      data.Tools(),
		"metadata":   genMetadata(),
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "document",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "application/pdf",
							"data":       pdfB64,
						},
					},
					map[string]any{
						"type": "text",
						"text": "What text does this PDF contain? Reply with ONLY the exact text, nothing else. Do not use any tools.",
					},
				},
			},
		},
	})

	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sse, start, delta := readSSE(resp.Body)
	full := merge(start, delta, sse)
	if full == nil {
		return []CheckResult{{Name: "pdf_extract", Pass: false, Detail: "parse failed"}}, nil
	}

	return []CheckResult{checkPDFExtract(full, pdfText)}, nil
}
