package remediation

// Fix is a remediation capability recommended by probe checks.
type Fix string

const (
	FixBodyRewrite      Fix = "body_rewrite"
	FixIDRewrite        Fix = "id_rewrite"
	FixSignatureRewrite Fix = "signature_rewrite"
	FixHeadersFake      Fix = "headers_fake"
	FixStripDone        Fix = "strip_done"
	FixStripContainer   Fix = "strip_container"
	FixStripBedrock     Fix = "strip_bedrock"
	FixForceGeo         Fix = "force_geo"
	FixThinkingInject   Fix = "thinking_inject"
	FixSmallProbeZero   Fix = "small_probe_zero"
	FixCacheFake        Fix = "cache_fake"
)

// Recommendation is the probe output describing what fixes would be needed.
type Recommendation struct {
	Enabled bool  `json:"enabled"`
	Fixes   []Fix `json:"fixes"`
}
