package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/symbolsecurity/cli/internal/auth"
	"github.com/symbolsecurity/cli/internal/output"
)

func TestUsersListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("training_status") != "OVERDUE" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		w.Header().Set("X-Pagination", `{"page":1,"per_page":50,"total_pages":1,"total_entries_size":1,"current_entries_size":1}`)
		io.WriteString(w, `[{"id":"u1","email":"a@b.c","firstName":"A"}]`)
	}))
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "users", "list", "--training-status", "OVERDUE", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var env output.Envelope
	if json.Unmarshal([]byte(out), &env) != nil || !env.OK || env.Summary != "1 user" {
		t.Fatalf("%s", out)
	}
}

func TestDeleteRequiresYes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call API")
	}))
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "users", "delete", "u1", "--json")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(out, "auth_error") && !strings.Contains(out, "usage_error") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "--yes") {
		t.Fatalf("expected --yes hint: %s", out)
	}
}

func TestUsersDelete(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/users/u1/" {
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
		called = true
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "users", "delete", "u1", "--yes", "--json")
	if err != nil {
		t.Fatal(err, out)
	}
	if !called {
		t.Fatal("not called")
	}
}

func TestAuthLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/access/" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer k3y" {
			t.Fatalf("auth %s", r.Header.Get("Authorization"))
		}
		io.WriteString(w, `{"accessToken":"a","refreshToken":"r","tokenType":"bearer"}`)
	}))
	t.Cleanup(srv.Close)
	out, err := run(t, srv, map[string]string{"SYMBOL_API_KEY": "k3y"}, "auth", "login", "--json")
	if err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out, `"ok":true`) {
		t.Fatalf("%s", out)
	}
	if strings.Contains(out, "k3y") || strings.Contains(out, `"a"`) && strings.Contains(out, "access") {
		if strings.Contains(out, "k3y") {
			t.Fatalf("leaked key: %s", out)
		}
	}
}

func TestAgentHelpJSON(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "users", "delete", "--agent", "--help")
	if err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out, `"gotchas"`) || !strings.Contains(out, "--yes") {
		t.Fatalf("%s", out)
	}
}

func TestCommandsCatalog(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "commands", "--json")
	if err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out, "users") || !strings.Contains(out, "auth login") {
		t.Fatalf("%s", out)
	}
}

func TestSetupWritesSkill(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	out, err := runHome(t, home, srv, nil, "setup", "--json")
	if err != nil {
		t.Fatal(err, out)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "symbol", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, srv *httptest.Server, env map[string]string, args ...string) (string, error) {
	t.Helper()
	return runHome(t, t.TempDir(), srv, env, args...)
}

func runHome(t *testing.T, home string, srv *httptest.Server, env map[string]string, args ...string) (string, error) {
	t.Helper()
	store := auth.NewStore(home, "default", auth.NewMemoryKeyring())
	_ = store.Save(auth.Credentials{APIKey: "k", AccessToken: "tok", RefreshToken: "ref"})
	var buf bytes.Buffer
	getenv := func(k string) string {
		if k == "SYMBOL_BASE_URL" {
			if env != nil {
				if v, ok := env[k]; ok {
					return v
				}
			}
			return srv.URL
		}
		if env != nil {
			return env[k]
		}
		return ""
	}
	err := ExecuteEnv(Env{
		Args:       args,
		Home:       home,
		Cwd:        t.TempDir(),
		Stdin:      strings.NewReader(""),
		Stdout:     &buf,
		Stderr:     io.Discard,
		HTTPClient: srv.Client(),
		Keyring:    store.Keyring,
		Getenv:     getenv,
		IsTerminal: func(io.Writer) bool { return false },
	})
	return buf.String(), err
}
