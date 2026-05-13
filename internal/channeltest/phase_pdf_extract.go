package channeltest

import "detector-service/internal/channeltest/data"

var probePDFExtract = &Probe{
	ID: "pdf_extract", Label: "PDF 提取探针",
	Tags:      []string{"heavy"},
	EstTokens: 25000,
	Checks:    []string{"pdf_extract"},
	Run:       (*Runner).runPDFExtract,
}

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
	p.recordStreamResult(full)
	if full == nil {
		return []CheckResult{{Name: "pdf_extract", Pass: false, Detail: "parse failed"}}, nil
	}

	return []CheckResult{checkPDFExtract(full, pdfText)}, nil
}
