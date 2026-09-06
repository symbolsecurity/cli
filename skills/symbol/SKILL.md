---
name: symbol
description: Manage Symbol Security via the `symbol` CLI — users, training, policies, cyber threats, phishing simulations, reported phishing, tickets, MSP companies, and monthly MSP schedules/activity. Use when the user asks about Symbol, overdue training, unsigned policies, new-hire onboarding, offboarding, urgent threats, phishing, MSP child companies, or planned vs sent work this month.
triggers:
  - symbol
  - symbol security
  - overdue training
  - unsigned policies
  - new hire onboarding
  - offboarding
  - cyber threats
  - phishing reports
  - phishing simulations
  - MSP company
  - MSP schedule
  - monthly activity
---

# Symbol Security

Talk to Symbol only through the `symbol` CLI. Do not call `api.symbolsecurity.com` yourself. Do not read `~/.config/symbol/credentials.json`.

## Invariants

1. Always pass `--json` or `--agent`. Never pipe to external `jq` — use `--jq`.
2. Check `.symbol/config.json` / `--company` before assuming tenant.
3. Destructive commands need `--yes`.
4. Do not print secrets, SSN, CCN, or raw tokens.
5. Unknown command → `symbol <cmd> --agent --help`.
6. Paginate; do not `--all` huge lists unless asked.

`symbol commands --json` dumps the catalog. `SYMBOL_NONINTERACTIVE=1` turns prompts into errors.

## Setup

```sh
symbol doctor --json
symbol auth login          # reads SYMBOL_API_KEY or prompts; never pass the key as argv
symbol setup               # copies this skill into detected agents
```

MSP: pick a company first (`symbol companies list --json`), then `--company <id>` or write it to `.symbol/config.json`.

## Workflows

### Overdue training

```sh
symbol users list --training-status OVERDUE --json
symbol training list --status OVERDUE --json
symbol training assets --json
symbol training assign --assets <id> --users <id> --json
```

### Urgent / pending cyber threats

```sh
symbol threats list --status URGENT --json
symbol threats list --status PENDING --json
symbol threats status <id> --to RESOLVED --json
symbol threats ignore <id> --yes --json
```

### Unsigned / overdue policies

```sh
symbol policies list --json
symbol policies assignments --status OVERDUE --json
symbol policies assignments --status PENDING --json
symbol policies assign --policy-id <id> --user-ids <id> --json
```

### Phishing program

```sh
symbol simulations list --json
symbol simulations list --status FAILED --json
symbol phishing list --json
```

### New-hire onboarding

Create the user, then assign training and the required policy. Confirm IDs from `training assets` and `policies list` first.

```sh
symbol users create --email <email> --first-name <n> --last-name <n> --json
symbol training assets --json
symbol training assign --assets <id> --users <id> --json
symbol policies list --json
symbol policies assign --policy-id <id> --user-ids <id> --json
```

### Offboarding

```sh
symbol users list --keyword <email> --json
symbol users delete <id> --yes --json
```

### List / create users

```sh
symbol users list --keyword <q> --json
symbol users create --email <email> --first-name <n> --last-name <n> --json
symbol users update <id> --title <title> --json
```

### MSP: pick a company, then operate

```sh
symbol companies list --json
symbol users list --company <id> --json
symbol companies features list <id> --json
symbol programs list --json
symbol programs assign <template_id> --company <id> --json
```

### MSP: planned vs sent work this month

Use for "what's scheduled this month?", "what actually went out?", QBRs, or catching companies with no activity.

`--month` (1-12) and `--year` are required by the API. Optional `--company <id>` limits to one child.

```sh
symbol msp schedules --month 9 --year 2026 --json
symbol msp activity --month 9 --year 2026 --json
symbol msp schedules --month 9 --year 2026 --company <id> --json
symbol msp activity --month 9 --year 2026 --company <id> --json
```

`schedules` = planned program work. `activity` = assigned/sent work that actually ran. Compare the two; do not `--all` across the whole MSP unless asked.

## Gotchas

- `--agent` returns raw data on success; errors stay enveloped (`ok: false`, `hint`).
- `--jq` runs on the payload (success data), not the envelope.
- `--page` cannot combine with `--all` or `--limit`.
- Tickets are admin-scoped.
- `companies` is MSP-only. `delete` always needs `--yes`.
