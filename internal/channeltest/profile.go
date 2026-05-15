package channeltest

// ChannelProfile describes the expected behavior of a specific channel type.
// Each profile overrides which checks are info-only (not scored) and which
// checks have adjusted expected values for that channel kind.
type ChannelProfile struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`

	// InfoOnly lists check names that become info-only (not scored) for this profile.
	InfoOnly []string `json:"info_only"`

	// ExpectPass lists check names that are expected to pass even though the
	// default profile would not require them. (reserved for future use)
	ExpectPass []string `json:"expect_pass,omitempty"`

	// ExpectFail lists check names that are expected to fail for this profile
	// and should not count against the score.
	ExpectFail []string `json:"expect_fail,omitempty"`
}

var builtinProfiles = map[string]*ChannelProfile{
	"console": {
		ID:          "console",
		Label:       "Console 直连",
		Description: "Anthropic Console API 直连，所有检查按标准执行",
	},
	"bedrock": {
		ID:          "bedrock",
		Label:       "AWS Bedrock",
		Description: "通过 AWS Bedrock 接入的官方渠道，ID 前缀和头部结构不同",
		InfoOnly: []string{
			"cf_headers", "cf_ray_format", "cookie_domain", "server_header",
			"server_timing", "headers",
		},
		ExpectFail: []string{
			"id_format", "backend_type", "request_id",
			"container", "bedrock_state",
			"service_tier", "signature_type_leak",
		},
	},
	"vertex": {
		ID:          "vertex",
		Label:       "Google Vertex AI",
		Description: "通过 Google Vertex AI 接入的官方渠道",
		InfoOnly: []string{
			"cf_headers", "cf_ray_format", "cookie_domain", "server_header",
			"server_timing", "headers",
		},
		ExpectFail: []string{
			"id_format", "backend_type", "request_id",
			"service_tier",
		},
	},
	"max": {
		ID:          "max",
		Label:       "Max 订阅",
		Description: "Anthropic Max 订阅号，结构基本一致但部分头部和字段可能略有差异",
		InfoOnly: []string{
			"cf_headers", "cf_ray_format", "cookie_domain", "server_header",
			"server_timing", "headers",
		},
	},
}

// GetProfile returns a built-in profile by ID, or nil if not found.
func GetProfile(id string) *ChannelProfile {
	return builtinProfiles[id]
}

// ListProfiles returns all built-in profiles.
func ListProfiles() []*ChannelProfile {
	order := []string{"console", "bedrock", "vertex", "max"}
	out := make([]*ChannelProfile, 0, len(order))
	for _, id := range order {
		if p := builtinProfiles[id]; p != nil {
			out = append(out, p)
		}
	}
	return out
}

// ProfileInfoOnly returns the merged info-only set for a given profile.
// It combines the global infoOnlyChecks with the profile's InfoOnly + ExpectFail.
func ProfileInfoOnly(profileID string) map[string]bool {
	merged := make(map[string]bool, len(infoOnlyChecks)+10)
	for k, v := range infoOnlyChecks {
		merged[k] = v
	}
	p := GetProfile(profileID)
	if p == nil {
		return merged
	}
	for _, name := range p.InfoOnly {
		merged[name] = true
	}
	for _, name := range p.ExpectFail {
		merged[name] = true
	}
	return merged
}
