package fingerprint

import (
	"crypto/rand"
	"math/big"
	"regexp"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func randB62(n int) string {
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(62))
		b[i] = base62[idx.Int64()]
	}
	return string(b)
}

func NewMsgID() string      { return "msg_01" + randB62(22) }
func NewToolUseID() string  { return "toolu_01" + randB62(22) }
func NewSrvToolID() string  { return "srvtoolu_01" + randB62(22) }
func NewRequestID() string  { return "req_011Ca9" + randB62(18) }

var (
	msgRe      = regexp.MustCompile(`^msg_01[0-9A-Za-z]{22}$`)
	toolRe     = regexp.MustCompile(`^toolu_01[0-9A-Za-z]{22}$`)
	srvToolRe  = regexp.MustCompile(`^srvtoolu_01[0-9A-Za-z]{22}$`)
)

// RewriteID rewrites non-standard IDs to official format.
func RewriteID(old string, tool, serverTool bool) string {
	if old == "" {
		if serverTool {
			return NewSrvToolID()
		}
		if tool {
			return NewToolUseID()
		}
		return NewMsgID()
	}
	if serverTool || len(old) > 9 && old[:9] == "srvtoolu_" {
		if srvToolRe.MatchString(old) {
			return old
		}
		return NewSrvToolID()
	}
	if tool || len(old) > 6 && old[:6] == "toolu_" {
		if toolRe.MatchString(old) {
			return old
		}
		return NewToolUseID()
	}
	if msgRe.MatchString(old) {
		return old
	}
	return NewMsgID()
}
