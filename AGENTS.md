# Symbol CLI

Standalone Go CLI + Agent Skill for the Symbol Security API.

**Read `PLAN.md` first.** Execute phases in order. Phase 0+1 is the first
session unless the user says otherwise.

## Commands

```sh
go test ./...
go build -o bin/symbol ./cmd/symbol
gofmt -l .
go vet ./...
```

## Rules

- Talk to `https://api.symbolsecurity.com` (or `SYMBOL_BASE_URL`). Do not
  import `github.com/symbolsecurity/symbol`.
- Skill instructs; CLI authenticates and calls the API.
- Secrets in keyring / `~/.config/symbol/credentials.json` only.
- `--json` / `--agent` envelope on every command. `--yes` on deletes.
- No comments unless asked.
