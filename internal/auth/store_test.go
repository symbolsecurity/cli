package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileFallback(t *testing.T) {
	home := t.TempDir()
	s := NewStore(home, "default", failingKeyring{})
	c := Credentials{APIKey: "k", AccessToken: "a", RefreshToken: "r"}
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(s.filePath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm %v", info.Mode().Perm())
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey != "k" || got.AccessToken != "a" {
		t.Fatalf("%+v", got)
	}
}

func TestMemoryRoundTrip(t *testing.T) {
	s := NewStore(t.TempDir(), "work", NewMemoryKeyring())
	if err := s.Save(Credentials{AccessToken: "tok"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.filePath()); !os.IsNotExist(err) {
		t.Fatalf("expected no file, err=%v", err)
	}
	got, err := s.Load()
	if err != nil || got.AccessToken != "tok" {
		t.Fatalf("%+v %v", got, err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err != ErrNotLoggedIn {
		t.Fatalf("expected not logged in, got %v", err)
	}
}

func TestNamedProfileFile(t *testing.T) {
	home := t.TempDir()
	s := NewStore(home, "msp", failingKeyring{})
	if err := s.Save(Credentials{APIKey: "x"}); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(s.filePath()) != "credentials-msp.json" {
		t.Fatalf("path %s", s.filePath())
	}
}

type failingKeyring struct{}

func (failingKeyring) Get(service, user string) (string, error) { return "", os.ErrNotExist }
func (failingKeyring) Set(service, user, password string) error { return os.ErrPermission }
func (failingKeyring) Delete(service, user string) error        { return os.ErrNotExist }
