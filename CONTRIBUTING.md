# Contributing

```sh
make test
make vet
make fmt
make build
```

- Do not import `github.com/symbolsecurity/symbol`.
- Do not commit API keys, `.env`, or `credentials.json`.
- Destructive commands need `--yes`.
- No comments unless asked.
- Unit tests use `httptest` only; do not hit the live API.
