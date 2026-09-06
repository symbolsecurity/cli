package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/symbolsecurity/cli/internal/auth"
	"github.com/symbolsecurity/cli/internal/output"
	"github.com/symbolsecurity/cli/skills"
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
		_, _ = io.WriteString(w, `[{"id":"u1","email":"a@b.c","firstName":"A"}]`)
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
	out, err := run(t, srv, nil, "users", "delete", "11111111-1111-1111-1111-111111111111", "--json")
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
		if r.Method != http.MethodDelete || r.URL.Path != "/users/11111111-1111-1111-1111-111111111111/" {
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
		called = true
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "users", "delete", "11111111-1111-1111-1111-111111111111", "--yes", "--json")
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
		_, _ = io.WriteString(w, `{"accessToken":"a","refreshToken":"r","tokenType":"bearer"}`)
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

func TestListCommands(t *testing.T) {
	seen := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = true
		_, _ = io.WriteString(w, `[]`)
	}))
	t.Cleanup(srv.Close)
	cases := []struct {
		args []string
		path string
	}{
		{[]string{"training", "list", "--json"}, "/training/list"},
		{[]string{"policies", "list", "--json"}, "/policies"},
		{[]string{"threats", "list", "--json"}, "/cyber-threats/results/"},
		{[]string{"phishing", "list", "--json"}, "/reported-phishing"},
		{[]string{"companies", "list", "--json"}, "/msp/companies"},
	}
	for _, tc := range cases {
		out, err := run(t, srv, nil, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v %s", tc.args, err, out)
		}
		if !seen[tc.path] {
			t.Fatalf("%v did not hit %s (got %v)", tc.args, tc.path, seen)
		}
	}
}

func TestDryRunDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call API")
	}))
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "users", "delete", "11111111-1111-1111-1111-111111111111", "--yes", "--dry-run", "--json")
	if err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out, "dry_run") {
		t.Fatalf("%s", out)
	}
}

func TestFullRequiresYes(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "users", "list", "--full", "--json")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(out, "--yes") {
		t.Fatalf("%s", out)
	}
}

func TestWrites(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)
	uid := "11111111-1111-1111-1111-111111111111"
	cases := [][]string{
		{"users", "create", "--email", "a@b.c", "--first-name", "A", "--last-name", "B", "--json"},
		{"users", "update", uid, "--title", "CEO", "--json"},
		{"training", "assign", "--assets", uid, "--users", uid, "--json"},
		{"policies", "assign", "--policy-id", uid, "--user-ids", uid, "--json"},
		{"threats", "status", uid, "--to", "RESOLVED", "--json"},
		{"webhooks", "create", "--endpoint", "https://example.com/hook", "--event-types", "user.created", "--json"},
	}
	for _, args := range cases {
		out, err := run(t, srv, nil, args...)
		if err != nil {
			t.Fatalf("%v: %v %s", args, err, out)
		}
	}
	want := []string{
		"POST /users/",
		"PUT /users/" + uid + "/",
		"POST /training/assign",
		"POST /policies/assign",
		"PUT /cyber-threats/results/" + uid + "/change-status/",
		"POST /event_subscriptions/",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestInvalidProfile(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	out, err := run(t, srv, map[string]string{"SYMBOL_PROFILE": "../etc"}, "version", "--json")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(out, "usage_error") {
		t.Fatalf("%s", out)
	}
}

func TestSkillCommandsExist(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	out, err := run(t, srv, nil, "commands", "--json")
	if err != nil {
		t.Fatal(err, out)
	}
	var env output.Envelope
	if json.Unmarshal([]byte(out), &env) != nil {
		t.Fatal(out)
	}
	raw, _ := json.Marshal(env.Data)
	var data struct {
		Commands []struct {
			Path []string `json:"path"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, c := range data.Commands {
		have[strings.Join(c.Path, " ")] = true
	}
	b, err := skills.FS.ReadFile("symbol/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("(?m)(?:^|\\s|`)symbol ([a-z0-9-]+(?: [a-z0-9-]+)*)")
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		path := m[1]
		if strings.Contains(path, "<") || path == "security" {
			continue
		}
		first := strings.Split(path, " ")[0]
		if first == "commands" {
			continue
		}
		if !have[path] && !have[first] {
			ok := false
			for k := range have {
				if strings.HasPrefix(path, k) || strings.HasPrefix(k, path) {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("SKILL.md command not in CLI: %s", path)
			}
		}
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
