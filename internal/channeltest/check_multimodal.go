package channeltest

import "strings"

func checkImageOCR(body map[string]any, expected string) CheckResult {
	text := strings.TrimSpace(strings.ToUpper(collectResponseText(body)))
	expected = strings.ToUpper(expected)
	if text == "" {
		return CheckResult{Name: "image_ocr", Pass: false,
			Expected: "包含 " + expected, Actual: "无文本输出",
			Detail: "no text content in response"}
	}
	if strings.Contains(text, expected) {
		return CheckResult{Name: "image_ocr", Pass: true,
			Expected: "包含 " + expected, Actual: "包含 " + expected,
			Detail: "image OCR correct: " + expected}
	}
	return CheckResult{Name: "image_ocr", Pass: false,
		Expected: "包含 " + expected, Actual: truncate(text, 40),
		Detail: "expected " + expected + ", got: " + truncate(text, 40)}
}

func checkPDFExtract(body map[string]any, expected string) CheckResult {
	text := strings.TrimSpace(strings.ToUpper(collectResponseText(body)))
	expected = strings.ToUpper(expected)
	if text == "" {
		return CheckResult{Name: "pdf_extract", Pass: false,
			Expected: "包含 " + expected, Actual: "无文本输出",
			Detail: "no text content in response"}
	}
	if strings.Contains(text, expected) {
		return CheckResult{Name: "pdf_extract", Pass: true,
			Expected: "包含 " + expected, Actual: "包含 " + expected,
			Detail: "PDF text correct: " + expected}
	}
	return CheckResult{Name: "pdf_extract", Pass: false,
		Expected: "包含 " + expected, Actual: truncate(text, 40),
		Detail: "expected " + expected + ", got: " + truncate(text, 40)}
}
