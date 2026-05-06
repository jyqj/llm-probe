package channeltest

import "strings"

// checkImageOCR verifies the model can read text from a dynamically generated image.
func checkImageOCR(body map[string]any, expected string) CheckResult {
	text := strings.TrimSpace(strings.ToUpper(collectResponseText(body)))
	if text == "" {
		return CheckResult{Name: "image_ocr", Pass: false, Detail: "no text content in response"}
	}
	expected = strings.ToUpper(expected)
	if strings.Contains(text, expected) {
		return CheckResult{Name: "image_ocr", Pass: true, Detail: "image OCR correct: " + expected}
	}
	return CheckResult{Name: "image_ocr", Pass: false, Detail: "expected " + expected + ", got: " + truncate(text, 40)}
}

// checkPDFExtract verifies the model can extract text from a dynamically generated PDF.
func checkPDFExtract(body map[string]any, expected string) CheckResult {
	text := strings.TrimSpace(strings.ToUpper(collectResponseText(body)))
	if text == "" {
		return CheckResult{Name: "pdf_extract", Pass: false, Detail: "no text content in response"}
	}
	expected = strings.ToUpper(expected)
	if strings.Contains(text, expected) {
		return CheckResult{Name: "pdf_extract", Pass: true, Detail: "PDF text correct: " + expected}
	}
	return CheckResult{Name: "pdf_extract", Pass: false, Detail: "expected " + expected + ", got: " + truncate(text, 40)}
}
