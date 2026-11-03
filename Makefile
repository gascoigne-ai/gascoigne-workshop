.PHONY: check fmt-check vet test build run

check: fmt-check vet test

fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet:
	go vet ./...

test:
	go test -race ./...

build:
	go build -o bin/workshop ./cmd/workshop

run:
	go run ./cmd/workshop
