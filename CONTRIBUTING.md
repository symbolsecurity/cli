# Contributing

```sh
go test -race ./...
go vet ./...
gofmt -l .
go build -o bin/symbol ./cmd/symbol
```

- Do not import `github.com/symbolsecurity/symbol`.
- Do not commit API keys, `.env`, or `credentials.json`.
- Destructive commands need `--yes`.
- No comments unless asked.
- Unit tests use `httptest` only; do not hit the live API.
