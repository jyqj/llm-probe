package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `
server:
  listen: ":9999"
upstream:
  base_url: "https://test.example.com/"
  api_key: "sk-test-key"
  timeout: 60
models:
  default_model: "claude-sonnet-4-6"
log:
  level: "debug"
admin:
  token: "adm"
probe:
  sig_secret: "sig"
`
	cfg := loadTempConfig(t, content)

	if cfg.Server.Listen != ":9999" {
		t.Errorf("listen = %s, want :9999", cfg.Server.Listen)
	}
	if cfg.Upstream.BaseURL != "https://test.example.com" {
		t.Errorf("base_url = %s, want trimmed https://test.example.com", cfg.Upstream.BaseURL)
	}
	if cfg.Upstream.Timeout != 60 {
		t.Errorf("timeout = %d, want 60", cfg.Upstream.Timeout)
	}
	if cfg.Models.DefaultModel != "claude-sonnet-4-6" {
		t.Errorf("default_model = %s, want claude-sonnet-4-6", cfg.Models.DefaultModel)
	}
	if cfg.Admin.Token != "adm" || cfg.Probe.SigSecret != "sig" {
		t.Errorf("admin/probe config not loaded: %+v", cfg)
	}
}

func TestLoadWithEnvVarDefault(t *testing.T) {
	t.Setenv("TEST_API_KEY", "sk-from-env")
	cfg := loadTempConfig(t, `
upstream:
  base_url: "${TEST_BASE:-https://default.example.com}"
  api_key: "${TEST_API_KEY}"
`)
	if cfg.Upstream.BaseURL != "https://default.example.com" {
		t.Errorf("base_url = %s", cfg.Upstream.BaseURL)
	}
	if cfg.Upstream.APIKey != "sk-from-env" {
		t.Errorf("api_key = %s, want sk-from-env", cfg.Upstream.APIKey)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid default target",
			cfg:  Config{Upstream: UpstreamConfig{BaseURL: "https://test.com", APIKey: "sk-test"}},
		},
		{
			name:    "missing upstream url",
			cfg:     Config{Upstream: UpstreamConfig{APIKey: "sk-test"}},
			wantErr: true,
		},
		{
			name:    "missing upstream key",
			cfg:     Config{Upstream: UpstreamConfig{BaseURL: "https://test.com"}},
			wantErr: true,
		},
		{
			name: "no upstream defaults is valid for request-supplied targets",
			cfg:  Config{},
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

func loadTempConfig(t *testing.T, content string) *Config {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return cfg
}
