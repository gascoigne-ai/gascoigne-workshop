.PHONY: check fmt-check vet test vet-release test-release build release run example-server shot install

check: fmt-check vet test vet-release test-release

fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet:
	go vet ./...

test:
	go test -race ./...

# The release build has its own files (build tag "release"); check them too.
vet-release:
	go vet -tags release ./...

test-release:
	go test -race -tags release ./...

build:
	go build -o bin/workshop ./cmd/workshop

# The release build ships no example endpoints and opens the client config
# (.workshop.yaml, .json or .toml in the config directory and the working
# directory) instead.
release:
	go build -tags release -o bin/workshop-release ./cmd/workshop

run:
	go run ./cmd/workshop

# Serve every protocol on localhost (REST 8081, WebSocket 8082, SSE 8083,
# MCP 8084, gRPC 8085) for the default build's example requests.
example-server:
	go run ./cmd/example-server

# Render the window once to a PNG (needs a display): make shot OUT=shot.png
shot:
	go run ./cmd/workshop -screenshot $(or $(OUT),shot.png) -send

# Install the program, its desktop entry and icon: make install PREFIX=$HOME/.local
PREFIX ?= /usr/local
install: build
	install -Dm755 bin/workshop $(DESTDIR)$(PREFIX)/bin/gascoigne-workshop
	install -Dm644 packaging/gascoigne-workshop.desktop $(DESTDIR)$(PREFIX)/share/applications/gascoigne-workshop.desktop
	mkdir -p $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps
	bin/workshop -icon-png $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps/gascoigne-workshop.png
