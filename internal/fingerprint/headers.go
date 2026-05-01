package fingerprint

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

// BuildResponseHeaders creates fake Anthropic-style response headers.
func BuildResponseHeaders(model, clientKey string, streaming bool, rl *RateLimitState) http.Header {
	now := time.Now().UTC()

	if rl == nil {
		rl = RateLimitTick(model, 500, 500)
	}

	envoyMs := 600 + randIntn(1400)
	baseSec := 30 + randIntn(25)
	reqReset := now.Add(time.Duration(baseSec) * time.Second)
	tokenReset := reqReset.Add(time.Duration(1+randIntn(3)) * time.Second)
	outputReset := tokenReset
	inputReset := tokenReset.Add(time.Duration(1+randIntn(3)) * time.Second)

	isoFmt := func(t time.Time) string { return t.Format("2006-01-02T15:04:05Z") }

	// Generate cookie
	cookieToken := make([]byte, 32)
	rand.Read(cookieToken)
	cookieToken2 := make([]byte, 32)
	rand.Read(cookieToken2)

	h := http.Header{}
	h.Set("Date", now.Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	if streaming {
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
	} else {
		h.Set("Content-Type", "application/json")
	}
	h.Set("Connection", "keep-alive")
	h.Set("Anthropic-Organization-Id", rl.OrgID)

	h.Set("Anthropic-Ratelimit-Input-Tokens-Limit", strconv.Itoa(rl.InputLimit))
	h.Set("Anthropic-Ratelimit-Input-Tokens-Remaining", strconv.Itoa(rl.InputRemain))
	h.Set("Anthropic-Ratelimit-Input-Tokens-Reset", isoFmt(inputReset))
	h.Set("Anthropic-Ratelimit-Output-Tokens-Limit", strconv.Itoa(rl.OutputLimit))
	h.Set("Anthropic-Ratelimit-Output-Tokens-Remaining", strconv.Itoa(rl.OutputRemain))
	h.Set("Anthropic-Ratelimit-Output-Tokens-Reset", isoFmt(outputReset))
	h.Set("Anthropic-Ratelimit-Requests-Limit", strconv.Itoa(rl.ReqLimit))
	h.Set("Anthropic-Ratelimit-Requests-Remaining", strconv.Itoa(rl.ReqRemain))
	h.Set("Anthropic-Ratelimit-Requests-Reset", isoFmt(reqReset))
	h.Set("Anthropic-Ratelimit-Tokens-Limit", strconv.Itoa(rl.TokensLimit))
	h.Set("Anthropic-Ratelimit-Tokens-Remaining", strconv.Itoa(rl.TokensRemain))
	h.Set("Anthropic-Ratelimit-Tokens-Reset", isoFmt(tokenReset))

	cfRayID, _ := rand.Int(rand.Reader, new(big.Int).SetUint64(^uint64(0)))
	h.Set("Cf-Cache-Status", "DYNAMIC")
	h.Set("Cf-Ray", fmt.Sprintf("%016x-YYZ", cfRayID.Uint64()))
	h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	h.Set("Request-Id", NewRequestID())
	h.Set("Server-Timing", fmt.Sprintf("x-originResponse;dur=%d", envoyMs))
	h.Set("Set-Cookie", fmt.Sprintf("_cfuvid=%s-%d.0-1.0.1.1-%s; HttpOnly; SameSite=None; Secure; Path=/; Domain=api.anthropic.com",
		base64.RawURLEncoding.EncodeToString(cookieToken),
		now.Unix(),
		base64.RawURLEncoding.EncodeToString(cookieToken2)))
	h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	h.Set("Vary", "Accept-Encoding")
	h.Set("X-Envoy-Upstream-Service-Time", strconv.Itoa(envoyMs))
	h.Set("X-Robots-Tag", "none")
	// Server header — mimic cloudflare
	h.Set("Server", "cloudflare")

	return h
}

// ApplyHeaders writes all headers from src to the ResponseWriter.
func ApplyHeaders(w http.ResponseWriter, h http.Header) {
	for k, vals := range h {
		for _, v := range vals {
			w.Header().Set(k, v)
		}
	}
}
