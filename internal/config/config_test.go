package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create temp config file
	content := `
server:
  listen: ":9999"
upstream:
  base_url: "https://test.example.com"
  api_key: "sk-test-key"
  timeout: 60
auth:
  api_keys:
    - "client-key-1"
models:
  default_model: "claude-sonnet-4-6"
log:
  level: "debug"
disguise:
  enabled: true
  body_rewrite: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Listen != ":9999" {
		t.Errorf("listen = %s, want :9999", cfg.Server.Listen)
	}
	if cfg.Upstream.BaseURL != "https://test.example.com" {
		t.Errorf("base_url = %s, want https://test.example.com", cfg.Upstream.BaseURL)
	}
	if cfg.Upstream.Timeout != 60 {
		t.Errorf("timeout = %d, want 60", cfg.Upstream.Timeout)
	}
	if !cfg.Disguise.Enabled {
		t.Error("disguise.enabled should be true")
	}
}

func TestLoadWithEnvVar(t *testing.T) {
	os.Setenv("TEST_API_KEY", "sk-from-env")
	defer os.Unsetenv("TEST_API_KEY")

	content := `
upstream:
  base_url: "https://test.example.com"
  api_key: "${TEST_API_KEY}"
auth:
  api_keys:
    - "test"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Upstream.APIKey != "sk-from-env" {
		t.Errorf("api_key = %s, want sk-from-env", cfg.Upstream.APIKey)
	}
}

func TestHasAPIKey(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			APIKeys: []string{"key1", "key2", "key3"},
		},
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"key1", true},
		{"key2", true},
		{"key3", true},
		{"key4", false},
		{"", false},
		{" key1 ", true}, // trimmed
	}

	for _, tc := range tests {
		got := cfg.HasAPIKey(tc.key)
		if got != tc.want {
			t.Errorf("HasAPIKey(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Upstream: UpstreamConfig{BaseURL: "https://test.com", APIKey: "sk-test"},
				Auth:     AuthConfig{APIKeys: []string{"key1"}},
			},
			wantErr: false,
		},
		{
			name: "missing upstream url",
			cfg: Config{
				Upstream: UpstreamConfig{APIKey: "sk-test"},
				Auth:     AuthConfig{APIKeys: []string{"key1"}},
			},
			wantErr: true,
		},
		{
			name: "missing upstream key",
			cfg: Config{
				Upstream: UpstreamConfig{BaseURL: "https://test.com"},
				Auth:     AuthConfig{APIKeys: []string{"key1"}},
			},
			wantErr: true,
		},
		{
			name: "missing auth keys (no keymap)",
			cfg: Config{
				Upstream: UpstreamConfig{BaseURL: "https://test.com", APIKey: "sk-test"},
			},
			wantErr: true,
		},
		{
			name: "no auth keys but keymap enabled",
			cfg: Config{
				Upstream: UpstreamConfig{BaseURL: "https://test.com", APIKey: "sk-test"},
				KeyMap:   KeyMapConfig{Enabled: true},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
