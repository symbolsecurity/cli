# Symbol CLI

Standalone Go CLI and agent skill for the Symbol Security API.
Talks to `https://api.symbolsecurity.com` (or `SYMBOL_BASE_URL`).
Does not import `github.com/symbolsecurity/symbol`.

```
Agent  →  Skill  →  symbol CLI  →  API
```

## Install

Pre-built binaries (Linux, macOS, Windows) are attached when a GitHub Release is published:

https://github.com/symbolsecurity/cli/releases/latest

Unpack `symbol` (or `symbol.exe`) onto `$PATH`.

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
go test -race ./...
go vet ./...
gofmt -l .
go build -o bin/symbol ./cmd/symbol
```

See `CONTRIBUTING.md`. Publishing a GitHub Release runs GoReleaser and uploads binaries.

Secrets live in the OS keyring, with `~/.config/symbol/credentials.json` (0600) as fallback.

Licensed under MIT. See `LICENSE`.
