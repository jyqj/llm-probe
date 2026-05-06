package data

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"strings"
	"testing"
)

func TestRandomOCRText(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		s := RandomOCRText(8)
		if len(s) != 8 {
			t.Fatalf("expected 8 chars, got %d: %q", len(s), s)
		}
		for _, c := range s {
			if !strings.ContainsRune(ocrCharset, c) {
				t.Fatalf("char %q not in charset", c)
			}
		}
		seen[s] = true
	}
	if len(seen) < 15 {
		t.Errorf("expected at least 15 unique strings in 20 runs, got %d", len(seen))
	}
}

func TestGenTestImageBase64(t *testing.T) {
	text := RandomOCRText(8)
	b64 := GenTestImageBase64(text)
	if b64 == "" {
		t.Fatal("empty base64")
	}

	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("invalid PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 288 || b.Dy() != 90 {
		t.Errorf("expected 288x90, got %dx%d", b.Dx(), b.Dy())
	}

	// Different text should produce different images
	b64b := GenTestImageBase64(RandomOCRText(8))
	if b64 == b64b {
		t.Error("two random images should differ")
	}
}

func TestGenTestPDFBase64(t *testing.T) {
	text := RandomOCRText(8)
	b64 := GenTestPDFBase64(text)
	if b64 == "" {
		t.Fatal("empty base64")
	}

	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}

	s := string(raw)
	if !strings.HasPrefix(s, "%PDF-1.0") {
		t.Error("missing PDF header")
	}
	if !strings.Contains(s, text) {
		t.Errorf("PDF does not contain expected text %q", text)
	}
	if !strings.HasSuffix(strings.TrimSpace(s), "%%EOF") {
		t.Error("missing EOF marker")
	}

	// Different text should produce different PDFs
	b64b := GenTestPDFBase64(RandomOCRText(8))
	if b64 == b64b {
		t.Error("two random PDFs should differ")
	}
}
