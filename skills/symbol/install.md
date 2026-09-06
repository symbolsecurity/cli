# Install Symbol CLI + skill

1. Install the CLI. Prefer a pre-built binary from https://github.com/symbolsecurity/cli/releases/latest (Linux, macOS, Windows). Or `go install github.com/symbolsecurity/cli/cmd/symbol@latest`, or copy `bin/symbol` onto `$PATH`.
2. `symbol auth login` — paste an API token from the Symbol app, or set `SYMBOL_API_KEY`. Never pass the key as a CLI argument.
3. Install this skill:
   - `npx skills add` if the agent supports it, or
   - `symbol setup`, or
   - copy `skills/symbol` into the agent skills directory (`~/.config/opencode/skills/symbol`, `~/.claude/skills/symbol`, `~/.codex/skills/symbol`).
4. `symbol doctor --json` should report binary, API, auth, and skill install.
