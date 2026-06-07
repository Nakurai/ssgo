package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const Filename = "ssgo.json"

type Profile struct {
	BaseURL string `json:"baseURL"`
}

type Colors struct {
	Background string `json:"background"`
	Text       string `json:"text"`
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Surface    string `json:"surface"`
}

var DefaultColors = Colors{
	Background: "#ffffff",
	Text:       "#1a1a1a",
	Primary:    "#2563eb",
	Secondary:  "#64748b",
	Surface:    "#f5f5f5",
}

func (c Colors) withDefaults() Colors {
	if c.Background == "" {
		c.Background = DefaultColors.Background
	}
	if c.Text == "" {
		c.Text = DefaultColors.Text
	}
	if c.Primary == "" {
		c.Primary = DefaultColors.Primary
	}
	if c.Secondary == "" {
		c.Secondary = DefaultColors.Secondary
	}
	if c.Surface == "" {
		c.Surface = DefaultColors.Surface
	}
	return c
}

// CSSVars returns a :root{} block with --color-* custom properties.
func (c Colors) CSSVars() string {
	c = c.withDefaults()
	return fmt.Sprintf(
		":root{--color-background:%s;--color-text:%s;--color-primary:%s;--color-secondary:%s;--color-surface:%s;}",
		c.Background, c.Text, c.Primary, c.Secondary, c.Surface,
	)
}

type Config struct {
	Title   string  `json:"title"`
	Style   string  `json:"style"`
	Logo    string  `json:"logo"`
	Favicon string  `json:"favicon"`
	Host    string  `json:"host,omitempty"`
	Colors  Colors  `json:"colors"`
	Dev     Profile `json:"dev"`
	Prod    Profile `json:"prod"`
}

func Load(dir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", Filename, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", Filename, err)
	}
	if cfg.Style == "" {
		cfg.Style = "default"
	}
	return &cfg, nil
}

func Save(dir string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, Filename), append(data, '\n'), 0644)
}
