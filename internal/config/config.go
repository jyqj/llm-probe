package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// expandEnvWithDefaults expands ${VAR:-default} and ${VAR} in s.
var envDefaultRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}`)

func expandEnvWithDefaults(s string) string {
	s = envDefaultRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := envDefaultRe.FindStringSubmatch(match)
		if val, ok := os.LookupEnv(parts[1]); ok && val != "" {
			return val
		}
		return parts[2]
	})
	return os.ExpandEnv(s)
}

// Config is the top-level channel/intelligence test service configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Upstream UpstreamConfig `yaml:"upstream"`
	Models   ModelConfig    `yaml:"models"`
	Log      LogConfig      `yaml:"log"`
	Admin    AdminConfig    `yaml:"admin"`
	Channel  ChannelConfig  `yaml:"channel"`
	Alert    AlertCfg       `yaml:"alert"`
	Storage  StorageConfig  `yaml:"storage"`
}

// StorageConfig controls persistent storage.
type StorageConfig struct {
	Path string `yaml:"path"`
}

// AlertCfg controls alert webhook configuration.
type AlertCfg struct {
	Enabled  bool           `yaml:"enabled"`
	Webhooks []WebhookCfg   `yaml:"webhooks"`
}

// WebhookCfg is a notification target in config.
type WebhookCfg struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

// UpstreamConfig optionally provides a default target for channel and intelligence runs.
type UpstreamConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Timeout int    `yaml:"timeout"`
}

type ModelConfig struct {
	DefaultModel string `yaml:"default_model,omitempty"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

// AdminConfig controls optional protection for channel/intelligence APIs.
type AdminConfig struct {
	Token string `yaml:"token"`
}

// ChannelConfig controls the channel authenticity test system.
type ChannelConfig struct {
	SigSecret string `yaml:"sig_secret"`
}

// Load reads config from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		Server:   ServerConfig{Listen: ":8080"},
		Upstream: UpstreamConfig{Timeout: 300},
		Models:   ModelConfig{DefaultModel: "claude-opus-4-6"},
		Log:      LogConfig{Level: "info"},
		Storage:  StorageConfig{Path: "data/detector.db"},
	}
	if err := yaml.Unmarshal([]byte(expandEnvWithDefaults(string(data))), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	c.Server.Listen = strings.TrimSpace(c.Server.Listen)
	if c.Server.Listen == "" {
		c.Server.Listen = ":8080"
	}
	c.Upstream.BaseURL = strings.TrimRight(strings.TrimSpace(c.Upstream.BaseURL), "/")
	c.Upstream.APIKey = strings.TrimSpace(c.Upstream.APIKey)
	c.Models.DefaultModel = strings.TrimSpace(c.Models.DefaultModel)
	c.Admin.Token = strings.TrimSpace(c.Admin.Token)
	c.Channel.SigSecret = strings.TrimSpace(c.Channel.SigSecret)
	if (c.Upstream.BaseURL == "") != (c.Upstream.APIKey == "") {
		return fmt.Errorf("upstream.base_url and upstream.api_key must be configured together, or both omitted")
	}
	return nil
}
