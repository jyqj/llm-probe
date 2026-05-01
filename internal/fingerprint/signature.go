package fingerprint

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
)

// SigHMACSecret is the HMAC key for signing thinking.signature blobs.
// Override via config.yaml disguise.sig_secret or SetSigSecret().
// IMPORTANT: Always set a custom secret in production via config or env var.
var SigHMACSecret []byte

func init() {
	// Generate random default secret on startup if not configured
	SigHMACSecret = make([]byte, 32)
	rand.Read(SigHMACSecret)
}

// SetSigSecret replaces the HMAC secret at runtime.
func SetSigSecret(secret string) {
	if secret != "" {
		SigHMACSecret = []byte(secret)
	}
}

// varint encodes an unsigned integer as a protobuf varint.
func varint(n uint64) []byte {
	var out []byte
	for n > 0x7F {
		out = append(out, byte(n&0x7F)|0x80)
		n >>= 7
	}
	out = append(out, byte(n&0x7F))
	return out
}

// pbVarint encodes a protobuf field with wire type 0 (varint).
func pbVarint(tag int, value uint64) []byte {
	out := []byte{byte(tag<<3) | 0}
	out = append(out, varint(value)...)
	return out
}

// pbLen encodes a protobuf field with wire type 2 (length-delimited).
func pbLen(tag int, payload []byte) []byte {
	out := []byte{byte(tag<<3) | 2}
	out = append(out, varint(uint64(len(payload)))...)
	out = append(out, payload...)
	return out
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func randIntRange(lo, hi int) int {
	if lo >= hi {
		return lo
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(hi-lo)))
	return lo + int(n.Int64())
}

// buildAnthropicBlob constructs a protobuf blob matching official signature schema.
func buildAnthropicBlob(inner1Sub1 int, identitySubTag int, identityValue []byte,
	outerField3 int, ctLo, ctHi int) []byte {

	f1Content := append([]byte{}, pbVarint(1, uint64(inner1Sub1))...)
	f1Content = append(f1Content, pbVarint(3, 2)...)
	f1Content = append(f1Content, pbLen(5, randomBytes(64))...)
	f1Content = append(f1Content, pbLen(identitySubTag, identityValue)...)
	f1Content = append(f1Content, pbVarint(7, 0)...)

	f1 := pbLen(1, f1Content)
	f3 := pbLen(3, randomBytes(12))
	f4 := pbLen(4, randomBytes(48))
	f5 := pbLen(5, randomBytes(randIntRange(ctLo, ctHi)))

	macInput := append([]byte{byte(outerField3)}, f1...)
	macInput = append(macInput, f3...)
	macInput = append(macInput, f4...)
	macInput = append(macInput, f5...)

	mac := hmac.New(sha256.New, SigHMACSecret)
	mac.Write(macInput)
	hmac12 := mac.Sum(nil)[:12]

	f2 := pbLen(2, hmac12)

	inner := append([]byte{}, f1...)
	inner = append(inner, f2...)
	inner = append(inner, f3...)
	inner = append(inner, f4...)
	inner = append(inner, f5...)

	outer := append(pbLen(2, inner), pbVarint(3, uint64(outerField3))...)
	return outer
}

// FakeSignature generates a thinking signature blob.
// thinkingText binds the actual thinking content into the HMAC so that
// altering the thinking block invalidates the signature.
func FakeSignature(model string, thinkingLen int, thinkingText string) string {
	ctLo, ctHi := 30, 100
	if thinkingLen > 0 {
		ctLo = max(17, thinkingLen/8)
		ctHi = max(60, thinkingLen/4)
	}
	identityPayload := []byte(model)
	if thinkingText != "" {
		h := sha256.Sum256([]byte(thinkingText))
		identityPayload = append(identityPayload, h[:]...)
	}
	blob := buildAnthropicBlob(12, 6, identityPayload, 1, ctLo, ctHi)
	return base64.StdEncoding.EncodeToString(blob)
}

// FakeEncryptedContent generates an encrypted_content blob for web_search results.
func FakeEncryptedContent() string {
	uid := []byte(newUUID())
	blob := buildAnthropicBlob(14, 4, uid, 3, 800, 2200)
	return base64.StdEncoding.EncodeToString(blob)
}

// FakeEncryptedIndex generates an encrypted_index blob for citations.
func FakeEncryptedIndex() string {
	uid := []byte(newUUID())
	blob := buildAnthropicBlob(14, 4, uid, 3, 30, 80)
	return base64.StdEncoding.EncodeToString(blob)
}

// VerifySignature checks if a base64 signature was issued by this gateway.
func VerifySignature(b64 string) (bool, string) {
	if b64 == "" {
		return false, "empty"
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		// Try URL-safe or raw
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
	// f2 = [tag_byte, length_varint, hmac_12_bytes...]
	f2 := fields[2]
	hmacLen, hi := decodeVarint(f2, 1) // skip tag byte, parse length varint
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

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0F) | 0x40
	b[8] = (b[8] & 0x3F) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
