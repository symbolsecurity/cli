# Symbol CLI

Standalone Go CLI and agent skill for the Symbol Security API.
Talks to `https://api.symbolsecurity.com` (or `SYMBOL_BASE_URL`).
Does not import `github.com/symbolsecurity/symbol`.

```
Agent  →  Skill  →  symbol CLI  →  API
```

## Install

```sh
go install github.com/symbolsecurity/cli/cmd/symbol@latest
```

Or from this repo:

```sh
go build -o bin/symbol ./cmd/symbol
```

## Setup

```sh
symbol auth login          # SYMBOL_API_KEY or prompt; never pass the key as argv
symbol doctor --json
symbol setup               # copies the embedded skill into detected agents
```

Every command accepts `--json` or `--agent`. Destructive actions need `--yes`.

```sh
symbol users list --json
symbol commands --json
symbol completion bash
```

## Develop

```sh
make test
make vet
make fmt
make build
```

See `CONTRIBUTING.md`. Releases use GoReleaser (`.goreleaser.yaml`).

Secrets live in the OS keyring, with `~/.config/symbol/credentials.json` (0600) as fallback.
