package fingerprint

import (
	"regexp"
	"testing"
)

func TestRewriteID(t *testing.T) {
	msgIDRe := regexp.MustCompile(`^msg_01[0-9A-Za-z]{22}$`)
	toolIDRe := regexp.MustCompile(`^toolu_01[0-9A-Za-z]{22}$`)
	srvToolIDRe := regexp.MustCompile(`^srvtoolu_01[0-9A-Za-z]{22}$`)

	tests := []struct {
		name       string
		input      string
		tool       bool
		serverTool bool
		wantRe     *regexp.Regexp
	}{
		{"bedrock msg", "msg_bdrk_abc123", false, false, msgIDRe},
		{"vertex msg", "msg_vrtx_abc123", false, false, msgIDRe},
		{"empty msg", "", false, false, msgIDRe},
		{"bedrock tool", "toolu_bdrk_abc123", true, false, toolIDRe},
		{"vertex tool", "toolu_vrtx_abc123", true, false, toolIDRe},
		{"server tool", "srvtoolu_bdrk_abc123", false, true, srvToolIDRe},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RewriteID(tc.input, tc.tool, tc.serverTool)
			if !tc.wantRe.MatchString(got) {
				t.Errorf("RewriteID(%q) = %q, want match %s", tc.input, got, tc.wantRe.String())
			}
		})
	}
}

func TestNewMsgID(t *testing.T) {
	re := regexp.MustCompile(`^msg_01[0-9A-Za-z]{22}$`)
	for i := 0; i < 10; i++ {
		id := NewMsgID()
		if !re.MatchString(id) {
			t.Errorf("NewMsgID() = %q, want match msg_01{22}", id)
		}
	}
}

func TestNewRequestID(t *testing.T) {
	re := regexp.MustCompile(`^req_01[0-9A-Za-z]+$`)
	for i := 0; i < 10; i++ {
		id := NewRequestID()
		if !re.MatchString(id) {
			t.Errorf("NewRequestID() = %q, want match req_01...", id)
		}
	}
}
