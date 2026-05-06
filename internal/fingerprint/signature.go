package fingerprint

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
)

// SigHMACSecret is the HMAC key used by VerifySignature to check thinking.signature blobs.
var SigHMACSecret []byte

func init() {
	SigHMACSecret = make([]byte, 32)
	rand.Read(SigHMACSecret)
}

// SetSigSecret replaces the HMAC secret at runtime.
func SetSigSecret(secret string) {
	if secret != "" {
		SigHMACSecret = []byte(secret)
	}
}

// VerifySignature checks if a base64 signature is a valid protobuf blob with correct HMAC.
func VerifySignature(b64 string) (bool, string) {
	if b64 == "" {
		return false, "empty"
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(b64)
		if err != nil {
			return false, "b64 decode error"
		}
	}
	if len(raw) < 4 {
		return false, "too short"
	}

	// Parse outer: field 2 LEN + field 3 VAR
	if raw[0] != 0x12 {
		return false, "outer tag mismatch"
	}
	innerLen, off := decodeVarint(raw, 1)
	if off+innerLen > len(raw) {
		return false, "outer len overflow"
	}
	inner := raw[off : off+innerLen]
	after := raw[off+innerLen:]
	if len(after) == 0 || after[0] != 0x18 {
		return false, "missing outer field3"
	}
	outerF3, _ := decodeVarint(after, 1)

	// Parse inner fields
	fields := make(map[int][]byte)
	i := 0
	for i < len(inner) {
		start := i
		tag := inner[i]
		i++
		fNum := int(tag >> 3)
		wire := tag & 7
		if wire != 2 {
			return false, "unexpected wire type"
		}
		ln, ni := decodeVarint(inner, i)
		i = ni
		if i+ln > len(inner) {
			return false, "inner len overflow"
		}
		end := i + ln
		fields[fNum] = inner[start:end]
		i = end
	}

	for _, f := range []int{1, 2, 3, 4, 5} {
		if _, ok := fields[f]; !ok {
			return false, "missing inner field"
		}
	}

	// Extract HMAC from field 2
	f2 := fields[2]
	hmacLen, hi := decodeVarint(f2, 1)
	if hmacLen != 12 {
		return false, "hmac len mismatch"
	}
	if hi+hmacLen > len(f2) {
		return false, "hmac data overflow"
	}
	hmacVal := f2[hi : hi+hmacLen]

	macInput := append([]byte{byte(outerF3)}, fields[1]...)
	macInput = append(macInput, fields[3]...)
	macInput = append(macInput, fields[4]...)
	macInput = append(macInput, fields[5]...)

	mac := hmac.New(sha256.New, SigHMACSecret)
	mac.Write(macInput)
	expected := mac.Sum(nil)[:12]

	if !hmac.Equal(hmacVal, expected) {
		return false, "hmac mismatch"
	}
	return true, "ok"
}

func decodeVarint(b []byte, i int) (int, int) {
	n := 0
	s := 0
	for i < len(b) {
		x := b[i]
		i++
		n |= int(x&0x7F) << s
		if x&0x80 == 0 {
			return n, i
		}
		s += 7
	}
	return n, i
}

// IntVal extracts an integer from a map[string]any, supporting float64, int, and json.Number.
func IntVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}
