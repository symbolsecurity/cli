package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrecedence(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "symbol"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(home), []byte(`{"base_url":"https://home.example","company":"from-home","profile":"p1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cwd, ".symbol"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ProjectPath(cwd), []byte(`{"company":"from-project"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"SYMBOL_BASE_URL": "https://env.example"}
	cfg, err := Load(home, cwd, func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://env.example" {
		t.Fatalf("base url: %s", cfg.BaseURL)
	}
	if cfg.Company != "from-project" {
		t.Fatalf("company: %s", cfg.Company)
	}
	if cfg.Profile != "p1" {
		t.Fatalf("profile: %s", cfg.Profile)
	}
}
