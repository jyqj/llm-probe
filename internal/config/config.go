package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// expandEnvWithDefaults expands ${VAR:-default} and ${VAR} in s.
// os.ExpandEnv only supports ${VAR}, this adds Bash-style default values.
var envDefaultRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}`)

func expandEnvWithDefaults(s string) string {
	// First expand ${VAR:-default} patterns
	s = envDefaultRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := envDefaultRe.FindStringSubmatch(match)
		if val, ok := os.LookupEnv(parts[1]); ok && val != "" {
			return val
		}
		return parts[2]
	})
	// Then expand remaining ${VAR} / $VAR
	return os.ExpandEnv(s)
}

// Config is the top-level gateway configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Upstream UpstreamConfig `yaml:"upstream"`
	Auth     AuthConfig     `yaml:"auth"`
	Sanitize SanitizeConfig `yaml:"sanitize"`
	Models   ModelConfig    `yaml:"models"`
	Log      LogConfig      `yaml:"log"`
	Disguise DisguiseConfig `yaml:"disguise"`
	KeyMap   KeyMapConfig   `yaml:"keymap"`
	Probe    ProbeConfig    `yaml:"probe"`
}

// ServerConfig for HTTP server.
type ServerConfig struct {
	Listen string `yaml:"listen"` // e.g. ":8080"
}

// UpstreamConfig for the upstream API provider.
type UpstreamConfig struct {
	BaseURL string `yaml:"base_url"` // e.g. "https://sub.claudecode.love"
	APIKey  string `yaml:"api_key"`  // upstream API key
	Timeout int    `yaml:"timeout"`  // request timeout in seconds, default 300
}

// AuthConfig for gateway API key authentication (clients -> gateway).
type AuthConfig struct {
	APIKeys []string `yaml:"api_keys"` // allowed client API keys
}

// SanitizeConfig for request sanitization.
type SanitizeConfig struct {
	BlockSystemPromptInjection bool     `yaml:"block_system_prompt_injection"`
	AllowedSystemPromptHashes  []string `yaml:"allowed_system_prompt_hashes,omitempty"`
	StripUnknownFields         bool     `yaml:"strip_unknown_fields"`
}

// ModelConfig for model mapping.
type ModelConfig struct {
	ModelMap     map[string]string `yaml:"model_map"`
	DefaultModel string           `yaml:"default_model,omitempty"`
}

// LogConfig for logging.
type LogConfig struct {
	Level        string `yaml:"level"`         // "debug", "info", "warn", "error"
	DumpRequests bool   `yaml:"dump_requests"` // log full request/response bodies to file
	DumpFile     string `yaml:"dump_file"`     // JSONL file path
}

// DisguiseConfig controls fingerprint disguise features.
type DisguiseConfig struct {
	Enabled            bool   `yaml:"enabled"`
	BodyRewrite        bool   `yaml:"body_rewrite"`
	IDRewrite          bool   `yaml:"id_rewrite"`
	SignatureRewrite   bool   `yaml:"signature_rewrite"`
	HeadersFake        bool   `yaml:"headers_fake"`
	StripBedrock       bool   `yaml:"strip_bedrock"`
	StripDone          bool   `yaml:"strip_done"`
	StripContainer     bool   `yaml:"strip_container"`
	ThinkingInject     bool   `yaml:"thinking_inject"`
	MaxTokensClamp     bool   `yaml:"max_tokens_clamp"`
	ForceGeo           bool   `yaml:"force_geo"`
	StripThinking      bool   `yaml:"strip_thinking"`
	SmallProbeZero     bool   `yaml:"small_probe_zero"`
	CacheFake          bool   `yaml:"cache_fake"`
	Refusal            bool   `yaml:"refusal"`              // intercept system prompt leak attempts
	SSEPadding         bool   `yaml:"sse_padding"`          // random whitespace before closing } in SSE data (default off)
	SigVerify          bool   `yaml:"sig_verify"`           // verify client-returned thinking.signature (default off)
	Identity           bool   `yaml:"identity"`             // inject model identity system prompt (default off)
	IdentityHide       bool   `yaml:"identity_hide"`        // hide injected identity token cost (default off)
	PassthroughBody    bool   `yaml:"passthrough_body"`     // skip body rewrite entirely (default off)
	PassthroughHeaders bool   `yaml:"passthrough_headers"`  // forward upstream headers as-is (default off)
	CaptureEnabled     bool   `yaml:"capture_enabled"`      // save request/response captures (default off)
	CaptureDir         string `yaml:"capture_dir"`          // capture output directory
	TavilyAPIKey       string `yaml:"tavily_api_key"`       // Tavily API key for web search synthesis
	WebSearch          bool   `yaml:"web_search"`           // synthesize web search responses locally (default off)
	SigSecret          string `yaml:"sig_secret"`           // HMAC secret for thinking.signature (auto-generated if empty)
}

// ProbeConfig for the fingerprint probe system.
type ProbeConfig struct {
	AutoProbe bool `yaml:"auto_probe"` // auto-probe new upstreams on first request
}

// KeyMapConfig for the key mapping system.
type KeyMapConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Strict            bool   `yaml:"strict"`              // reject unknown keys
	KeysFile          string `yaml:"keys_file"`           // path to keys.json
	AdminToken        string `yaml:"admin_token"`
	PublicGatewayBase string `yaml:"public_gateway_base"` // e.g. "https://aws.claudecode.love"
}

// Load reads config from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	expanded := expandEnvWithDefaults(string(data))

	cfg := &Config{
		Server:   ServerConfig{Listen: ":8080"},
		Upstream: UpstreamConfig{Timeout: 300},
		Log:      LogConfig{Level: "info"},
		Disguise: DisguiseConfig{
			Enabled:          true,
			BodyRewrite:      true,
			IDRewrite:        true,
			SignatureRewrite: true,
			HeadersFake:      true,
			StripBedrock:     true,
			StripDone:        true,
			StripContainer:   true,
			ThinkingInject:   true,
			MaxTokensClamp:   true,
			ForceGeo:         true,
			StripThinking:    true,
			SmallProbeZero:   true,
			CacheFake:        true,
			Refusal:          true,
			CaptureDir:       "",
		},
		KeyMap: KeyMapConfig{
			KeysFile: "keys.json",
		},
		Probe: ProbeConfig{},
	}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Upstream.BaseURL == "" {
		return fmt.Errorf("upstream.base_url is required")
	}
	if c.Upstream.APIKey == "" {
		return fmt.Errorf("upstream.api_key is required")
	}
	// Strip trailing slash
	c.Upstream.BaseURL = strings.TrimRight(c.Upstream.BaseURL, "/")

	// If keymap is enabled, auth.api_keys is optional (keymap handles auth)
	if !c.KeyMap.Enabled && len(c.Auth.APIKeys) == 0 {
		return fmt.Errorf("auth.api_keys must have at least one key (or enable keymap)")
	}
	return nil
}

// HasAPIKey checks if a client key is authorized.
func (c *Config) HasAPIKey(key string) bool {
	key = strings.TrimSpace(key)
	for _, k := range c.Auth.APIKeys {
		if k == key {
			return true
		}
	}
	return false
}
