package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/symbolsecurity/cli/internal/auth"
	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) authCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Authenticate with the Symbol API"}
	cmd.AddCommand(rt.authLoginCmd(), rt.authStatusCmd(), rt.authLogoutCmd(), rt.authRefreshCmd())
	return cmd
}

func (rt *Runtime) authLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Exchange an API key for access tokens",
		Annotations: map[string]string{
			"gotchas": "Never pass the API key as argv; use SYMBOL_API_KEY or the prompt",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.TrimSpace(rt.Env.Getenv("SYMBOL_API_KEY"))
			if key == "" {
				if rt.nonInteractive() {
					return rt.Out.Fail(output.AuthError("Set SYMBOL_API_KEY or run interactively: symbol auth login"))
				}
				var err error
				key, err = rt.promptKey()
				if err != nil {
					return rt.Out.Fail(err)
				}
			}
			if key == "" {
				return rt.Out.Fail(output.Usage("API key is required", "Set SYMBOL_API_KEY or paste a key at the prompt"))
			}
			if _, err := rt.Client.Login(rt.ctx(cmd), key); err != nil {
				return rt.Out.Fail(err)
			}
			data := map[string]string{"profile": rt.Config.Profile, "base_url": rt.Config.BaseURL}
			if rt.Company != "" {
				data["company"] = rt.Company
			}
			return rt.Out.Success(data, "logged in", []output.Breadcrumb{
				crumb("status", "symbol auth status"),
				crumb("users", "symbol users list"),
			}, nil)
		},
	}
}

func (rt *Runtime) promptKey() (string, error) {
	fmt.Fprint(rt.Env.Stderr, "API key: ")
	f, ok := rt.Env.Stdin.(*os.File)
	if ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(rt.Env.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	b, err := io.ReadAll(rt.Env.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func (rt *Runtime) authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := rt.Store.Load()
			data := map[string]any{
				"logged_in": false,
				"profile":   rt.Config.Profile,
				"base_url":  rt.Config.BaseURL,
			}
			if rt.Company != "" {
				data["company"] = rt.Company
			}
			if err != nil {
				if err == auth.ErrNotLoggedIn {
					return rt.Out.Success(data, "not logged in", []output.Breadcrumb{crumb("login", "symbol auth login")}, nil)
				}
				return rt.Out.Fail(err)
			}
			data["logged_in"] = creds.AccessToken != "" || creds.APIKey != ""
			data["has_api_key"] = creds.APIKey != ""
			data["store"] = "keyring"
			if rt.Store.UsingFile() {
				data["store"] = "file"
			}
			if exp, ok := jwtExpiry(creds.AccessToken); ok {
				data["expires_at"] = exp.UTC().Format(time.RFC3339)
				data["expired"] = time.Now().After(exp)
			}
			return rt.Out.Success(data, "logged in", []output.Breadcrumb{
				crumb("users", "symbol users list"),
				crumb("refresh", "symbol auth refresh"),
			}, nil)
		},
	}
}

func (rt *Runtime) authLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Store.Clear(); err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(map[string]any{"logged_in": false, "profile": rt.Config.Profile}, "logged out", []output.Breadcrumb{crumb("login", "symbol auth login")}, nil)
		},
	}
}

func (rt *Runtime) authRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Rotate the access token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := rt.Client.Refresh(rt.ctx(cmd)); err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(map[string]any{"refreshed": true, "profile": rt.Config.Profile}, "token refreshed", []output.Breadcrumb{crumb("status", "symbol auth status")}, nil)
		},
	}
}

func jwtExpiry(tok string) (time.Time, bool) {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payload := parts[1]
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}
	b, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		b, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, false
		}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(b, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}
