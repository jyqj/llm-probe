package target

import (
	"fmt"
	"strings"

	"detector-service/internal/config"
)

// Config identifies the model API target under test.
type Config struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"-"`
	Model   string `json:"model"`
}

// Resolve applies service defaults and normalizes a request-level target.
func Resolve(app *config.Config, baseURL, apiKey, model string) Config {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = app.Upstream.BaseURL
	}
	if strings.TrimSpace(apiKey) == "" {
		apiKey = app.Upstream.APIKey
	}
	if strings.TrimSpace(model) == "" {
		model = app.Models.DefaultModel
	}
	return Config{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIKey:  strings.TrimSpace(apiKey),
		Model:   strings.TrimSpace(model),
	}
}

// Validate requires all target fields needed to call the upstream model API.
func (t Config) Validate() error {
	if t.BaseURL == "" || t.APIKey == "" || t.Model == "" {
		return fmt.Errorf("target_base, target_key and model are required")
	}
	return nil
}
