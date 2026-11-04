.PHONY: check fmt-check vet test build run shot install

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

# Install the program, its desktop entry and icon: make install PREFIX=$HOME/.local
PREFIX ?= /usr/local
install: build
	install -Dm755 bin/workshop $(DESTDIR)$(PREFIX)/bin/gascoigne-workshop
	install -Dm644 packaging/gascoigne-workshop.desktop $(DESTDIR)$(PREFIX)/share/applications/gascoigne-workshop.desktop
	mkdir -p $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps
	bin/workshop -icon-png $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps/gascoigne-workshop.png
