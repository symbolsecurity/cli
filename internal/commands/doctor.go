package commands

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/auth"
	"github.com/symbolsecurity/cli/internal/output"
	"github.com/symbolsecurity/cli/internal/version"
	"github.com/symbolsecurity/cli/skills"
)

func (rt *Runtime) doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check binary, auth, API, and skill install",
		RunE: func(cmd *cobra.Command, args []string) error {
			checks := []map[string]any{}
			ok := true
			add := func(name, status, detail string) {
				if status == "fail" {
					ok = false
				}
				checks = append(checks, map[string]any{"name": name, "status": status, "detail": detail})
			}

			add("binary", "ok", "symbol "+version.Version)

			base := rt.Config.BaseURL
			if base == "" {
				add("base_url", "fail", "missing")
			} else {
				add("base_url", "ok", base)
			}

			ctx, cancel := context.WithTimeout(rt.ctx(cmd), 8*time.Second)
			defer cancel()
			if err := rt.Client.Ping(ctx); err != nil {
				add("api", "fail", err.Error())
			} else {
				add("api", "ok", base)
			}

			if _, err := rt.Store.Load(); err != nil {
				if err == auth.ErrNotLoggedIn {
					add("auth", "fail", "not logged in")
				} else {
					add("auth", "fail", err.Error())
				}
			} else if rt.Store.UsingFile() {
				add("auth", "warn", "credentials on disk (0600); keyring unavailable")
			} else {
				add("auth", "ok", "keyring")
			}

			if _, err := skills.FS.ReadFile("symbol/SKILL.md"); err != nil {
				add("skill_embed", "fail", err.Error())
			} else {
				add("skill_embed", "ok", "embedded")
			}

			installed := 0
			for _, dir := range skillDirs(rt.Env.Home) {
				if _, err := os.Stat(filepath.Join(dir, "symbol", "SKILL.md")); err == nil {
					installed++
				}
			}
			if installed == 0 {
				add("skill_install", "fail", "run: symbol setup")
			} else {
				add("skill_install", "ok", plural(installed, "agent"))
			}

			summary := "doctor ok"
			if !ok {
				summary = "doctor found issues"
			}
			return rt.Out.Success(map[string]any{"ok": ok, "checks": checks}, summary, []output.Breadcrumb{
				crumb("login", "symbol auth login"),
				crumb("setup", "symbol setup"),
			}, nil)
		},
	}
}
