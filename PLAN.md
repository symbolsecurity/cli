# Symbol CLI + Agent Skill

Build a Basecamp-style agent access layer for the Symbol Security API:

```
Agent  →  Skill (SKILL.md)  →  `symbol` CLI  →  api.symbolsecurity.com
```

The skill never talks to Symbol. The CLI is the only thing with credentials.
This repo is standalone — do not import `github.com/symbolsecurity/symbol`.

API docs: https://app.symbolsecurity.com/api/docs.html
Spec: `https://app.symbolsecurity.com/api/swagger.<hash>.json` (Swagger 2.0, host `api.symbolsecurity.com`)
Platform source (reference only): `~/Documents/code/symbolsecurity/symbol`

## Goal

An agent that can run shell commands can manage Symbol the way a human in the
app can: users, training, policies, cyber threats, phishing, tickets, MSP
companies. Every command is JSON-first so models can navigate without scraping.

## Non-goals (v1)

- OAuth browser login (API is API-key → JWT)
- Importing the monolith SDK
- Full 1:1 coverage of every MSP duplicate path as its own command
- MCP server (phase 7, optional)
- TUI

## Architecture

```
cli/
├── cmd/symbol/main.go
├── internal/
│   ├── auth/        # API key → JWT, refresh, keyring
│   ├── config/      # ~/.config/symbol/{config.json,credentials.json}
│   ├── client/      # HTTP, pagination, company scope, retries
│   ├── output/      # envelope, --json/--agent/--jq/--md
│   └── commands/    # cobra (or similar) command tree
├── skills/
│   ├── embed.go     # //go:embed symbol
│   └── symbol/SKILL.md
├── PLAN.md
└── AGENTS.md
```

Binary name: `symbol`. Module: `github.com/symbolsecurity/cli` (adjust if the
org prefers another path). Go 1.25+.

### Output envelope (every command)

Success:

```json
{
  "ok": true,
  "data": {},
  "summary": "12 users",
  "breadcrumbs": [{"action": "show", "cmd": "symbol users show <id>"}]
}
```

Error:

```json
{
  "ok": false,
  "error": "Unauthorized",
  "code": "auth_error",
  "retryable": false,
  "hint": "Run: symbol auth login"
}
```

Flags (global):

| Flag | Behavior |
|---|---|
| `--json` | Envelope JSON |
| `--agent` | Success = raw data, no prompts; errors stay enveloped |
| `--jq <expr>` | Built-in jq on the payload (do not shell out to jq) |
| `--md` | Human tables |
| `--company <id>` | MSP scope: send as `company_id` on `/auth/access/` **or** prefix `/msp/companies/{id}/…` |
| `--profile <name>` | Named identity (MSP vs child, staging vs prod) |
| `--yes` | Required for delete / deactivate / ignore |

TTY default can be styled; piped default is JSON. Agents must pass `--json` or
`--agent` explicitly. `SYMBOL_NONINTERACTIVE=1` turns prompts into errors.

`--agent --help` on any command returns structured JSON (flags, subcommands,
gotchas). `symbol commands --json` dumps the catalog.

### Auth

1. User generates an API token in the Symbol app.
2. `symbol auth login` prompts for it (or reads `SYMBOL_API_KEY`); never argv.
3. `POST /auth/access/` with `Authorization: Bearer <api-key>`.
4. Store `accessToken` + `refreshToken` in OS keyring; file fallback
   `~/.config/symbol/credentials.json` (0600).
5. Access token ~60 min. Refresh via `POST /auth/refresh/` with the current
   refresh token (it **rotates** — persist the new pair every time).
6. On 401, refresh once and retry. If refresh fails, error with
   `hint: symbol auth login`.
7. MSP: `POST /auth/access/?company_id=<id>` scopes the JWT to a child company.
   Prefer this over duplicating every `/msp/companies/{id}/…` path. Keep the
   MSP-prefixed routes as a fallback when an endpoint only exists there.

Never log tokens. Never put secrets in SKILL.md.

### HTTP client

- Base URL `https://api.symbolsecurity.com` (override `SYMBOL_BASE_URL` for
  local: the platform often runs on `127.0.0.1:3000`).
- Parse `X-Pagination` JSON header: `page`, `per_page`, `total_pages`,
  `total_entries_size`.
- Flags: `--page N`, `--per-page N` (API default 50), `--all` (walk pages),
  `--limit N` (stop after N items). `--all` and `--limit` mutually exclusive
  with `--page`.
- Timeouts, one retry on 5xx/network, surface `retryable` in the envelope.
- PII: by default redact fields that look like SSN, CCN, password, raw API
  keys. `--full` disables redaction.

### Company vs MSP

One command tree. `--company` (or `~/.config/symbol/config.json` /
`.symbol/config.json`) selects scope.

- No `--company`: company-token routes (`/users/`, `/training/list`, …).
- With `--company`: scoped token **or** `/msp/companies/{id}/…` for routes
  that only exist under MSP (features, program templates, MSP activity).

Do not expose parallel `symbol msp users` vs `symbol users` unless the MSP
route has no company equivalent.

## Command surface (v1)

Implement in this order. Each group is a mergeable slice of work.

### 0. Scaffold + auth

```
symbol auth login
symbol auth status
symbol auth logout
symbol auth refresh
symbol version
symbol doctor          # binary, auth, base URL, optional skill path
```

### 1. Read path (prove the loop)

```
symbol users list [--keyword] [--training-status OVERDUE|COMPLETED|PENDING]
symbol training list
symbol training assets
symbol training leaderboard
symbol policies list
symbol policies assignments
symbol threats list [--status PENDING|URGENT] [--from] [--until]
symbol assessments list
symbol phishing list
symbol simulations list
```

Done when: `symbol auth login` + `symbol users list --json` works against
production or local API.

### 2. Writes (guarded)

```
symbol users create …
symbol users update <id>
symbol users delete <id> --yes
symbol training assign …
symbol policies assign …
symbol threats status <id> --to PENDING|URGENT|RESOLVED
symbol threats ignore <id> --yes
symbol threats delete <id> --yes
symbol threats keywords list|create|delete
```

### 3. Remaining company domains

```
symbol domains list|create
symbol domains threats
symbol email-threats alerts|activate|deactivate
symbol webhooks list|create|update|delete|test
symbol invoices list
symbol tickets list|conversation|notes|reply|status
```

Tickets are admin-scoped; still wrap them — the skill will say who can use them.

### 4. MSP extras (no company twin)

```
symbol companies list|show|create|update|delete
symbol companies features list|enable|disable
symbol programs list|assign
symbol msp activity
symbol msp schedules
symbol msp users
```

`companies` is MSP-only. `delete` requires `--yes`.

### 5. Agent skill

`skills/symbol/SKILL.md` — Agent Skills format (`name`, `description`,
`triggers`). Teach workflows, not the whole catalog:

- Overdue training
- Urgent / pending cyber threats
- Assign training or a policy
- List/create users
- MSP: pick a company, then operate

Invariants the skill must state:

1. Always `--json` or `--agent`. Never pipe to external `jq` — use `--jq`.
2. Check `.symbol/config.json` / `--company` before assuming tenant.
3. Destructive commands need `--yes`.
4. Do not print secrets, SSN, CCN, raw tokens.
5. Unknown command → `symbol <cmd> --agent --help`.
6. Paginate; do not `--all` huge lists unless asked.

Also add `skills/symbol-doctor/SKILL.md` only if doctor/setup needs its own
remediation playbook. Keep it short.

`install.md` (agent-executable, Basecamp style):

1. Install CLI
2. `symbol auth login`
3. `npx skills add` **or** copy `skills/symbol` into the agent’s skills dir

Ship the skill inside the binary (`//go:embed`) so `symbol setup` can write it
into `~/.config/opencode/skills`, `~/.claude/skills`, `~/.codex/skills`, etc.

### 6. Agent discovery + setup

- `--agent --help` JSON on every command
- `symbol commands --json`
- `symbol setup` copies the embedded skill into detected agents
- Breadcrumbs on every response

### 7. Optional later

- `symbol mcp` stdio server
- Skill-drift CI (commands mentioned in SKILL.md must exist)
- Homebrew / install script
- Named profiles

## Suggested packages

- CLI: `github.com/spf13/cobra` + `github.com/spf13/viper` (or a thinner flag
  lib if cobra feels heavy — stay consistent).
- Keyring: `github.com/zalando/go-keyring`
- jq: `github.com/itchyny/gojq`
- Tests: `net/http/httptest` for client; golden JSON for envelope; no live API
  in unit tests.

## Implementation rules

- JSON tags and command names match the API where possible; human names in help.
- One HTTP helper; commands do not call `http.Client` directly.
- Pagination helper used by every list command.
- Table-driven tests per command group.
- No comments unless the user asked.
- Do not commit API keys, `.env`, or `credentials.json`.

## Phase checklist for the next session

Start here. Do not skip to skills before the read path works.

- [x] Phase 0: module, `cmd/symbol`, auth, config, client, envelope, `doctor`
- [x] Phase 1: users / training / policies / threats list
- [x] Phase 2: guarded writes
- [x] Phase 3: domains, phishing, tickets, webhooks
- [x] Phase 4: MSP companies + extras
- [x] Phase 5: SKILL.md + install.md + embed
- [x] Phase 6: `--agent --help`, `setup`, breadcrumbs
- [ ] Phase 7: MCP / install script only if asked

## Reference: API groups

Company (token-scoped): `/users/`, `/training/*`, `/policies*`, `/assessments`,
`/assessment-assignments`, `/cyber-threats/*`, `/domains`, `/email-threats/*`,
`/reported-phishing`, `/threat-simulations/list`, `/event_subscriptions/`,
`/invoices/`, `/tickets/*`

Auth: `POST /auth/access/`, `POST /auth/refresh/`

MSP: `/msp/companies`, `/msp/companies/{id}/…` (mirrors company + features,
programs, activity, schedules, program templates)

~83 paths, ~99 operations. Cover by domain, not by path count.
