.PHONY: check fmt-check vet test build run shot

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

# Render the window once to a PNG (needs a display): make shot OUT=shot.png
shot:
	go run ./cmd/workshop -screenshot $(or $(OUT),shot.png) -send
