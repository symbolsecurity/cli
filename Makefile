.PHONY: test vet fmt build

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

build:
	go build -o bin/symbol ./cmd/symbol

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
