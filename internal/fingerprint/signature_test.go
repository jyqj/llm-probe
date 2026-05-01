package fingerprint

import (
	"encoding/base64"
	"testing"
)

func TestFakeSignature(t *testing.T) {
	sig := FakeSignature("claude-opus-4-6", 100, "test thinking content")

	// Should be valid base64
	raw, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("not valid base64: %v", err)
	}

	// Should start with protobuf field 2 (0x12)
	if len(raw) < 4 || raw[0] != 0x12 {
		t.Errorf("invalid protobuf structure: first byte = 0x%02x", raw[0])
	}

	// Should be verifiable
	ok, reason := VerifySignature(sig)
	if !ok {
		t.Errorf("FakeSignature should be verifiable, got: %s", reason)
	}
}

func TestVerifySignature(t *testing.T) {
	tests := []struct {
		name   string
		sig    string
		wantOK bool
	}{
		{"empty", "", false},
		{"invalid base64", "not-base64!!!", false},
		{"too short", "AAAA", false},
		{"valid fake", FakeSignature("claude-opus-4-6", 50, "some thinking"), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, _ := VerifySignature(tc.sig)
			if ok != tc.wantOK {
				t.Errorf("VerifySignature(%q) = %v, want %v", tc.sig[:min(20, len(tc.sig))], ok, tc.wantOK)
			}
		})
	}
}

func TestFakeEncryptedContent(t *testing.T) {
	ec := FakeEncryptedContent()

	raw, err := base64.StdEncoding.DecodeString(ec)
	if err != nil {
		t.Fatalf("not valid base64: %v", err)
	}

	if len(raw) < 100 {
		t.Errorf("encrypted_content too short: %d bytes", len(raw))
	}
}

func TestSetSigSecret(t *testing.T) {
	original := make([]byte, len(SigHMACSecret))
	copy(original, SigHMACSecret)

	SetSigSecret("test-secret-123")
	if string(SigHMACSecret) != "test-secret-123" {
		t.Errorf("SetSigSecret failed to update secret")
	}

	// Restore
	SigHMACSecret = original
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
