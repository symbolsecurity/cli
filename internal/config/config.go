package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const DefaultBaseURL = "https://api.symbolsecurity.com"

type File struct {
	BaseURL string `json:"base_url,omitempty"`
	Company string `json:"company,omitempty"`
	Profile string `json:"profile,omitempty"`
}

type Config struct {
	Home    string
	Cwd     string
	BaseURL string
	Company string
	Profile string
}

func Dir(home string) string {
	return filepath.Join(home, ".config", "symbol")
}

func Path(home string) string {
	return filepath.Join(Dir(home), "config.json")
}

func ProjectPath(cwd string) string {
	return filepath.Join(cwd, ".symbol", "config.json")
}

func Load(home, cwd string, getenv func(string) string) (*Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg := &Config{
		Home:    home,
		Cwd:     cwd,
		BaseURL: DefaultBaseURL,
		Profile: "default",
	}
	if err := mergeFile(cfg, Path(home)); err != nil {
		return nil, err
	}
	if cwd != "" {
		if err := mergeFile(cfg, ProjectPath(cwd)); err != nil {
			return nil, err
		}
	}
	if v := getenv("SYMBOL_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := getenv("SYMBOL_COMPANY"); v != "" {
		cfg.Company = v
	}
	if v := getenv("SYMBOL_PROFILE"); v != "" {
		cfg.Profile = v
	}
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	return cfg, nil
}

func mergeFile(cfg *Config, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	if f.BaseURL != "" {
		cfg.BaseURL = f.BaseURL
	}
	if f.Company != "" {
		cfg.Company = f.Company
	}
	if f.Profile != "" {
		cfg.Profile = f.Profile
	}
	return nil
}

func (c *Config) Write() error {
	if err := os.MkdirAll(Dir(c.Home), 0o700); err != nil {
		return err
	}
	f := File{BaseURL: c.BaseURL, Company: c.Company, Profile: c.Profile}
	if f.BaseURL == DefaultBaseURL {
		f.BaseURL = ""
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(c.Home), append(b, '\n'), 0o600)
}
