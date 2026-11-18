.PHONY: packagemap check fuzz fmt-check vet test vet-release test-release build release run example-server shot install

check: fmt-check vet test vet-release test-release

# Rewrite .harness/packagemap.md from the package tree.
packagemap:
	go test ./internal -run TestThePackageMapIsCurrent -update

fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet:
	go vet ./...

test:
	go test -race ./...

# Short run of every fuzz target (package:Target pairs below); not part of
# check. A crasher lands in the package's testdata/fuzz and becomes a
# regression test: fix it, keep the file.
FUZZTIME ?= 30s
FUZZ_TARGETS = \
	./internal/collection/model/miniyaml:FuzzParse \
	./internal/collection/model/openapi:FuzzParse \
	./internal/collection/model/postman:FuzzParse \
	./internal/collection/model/clientconfig:FuzzParse \
	./internal/request/model:FuzzParseCurl \
	./internal/request/model:FuzzURLParams \
	./internal/request/model/assert:FuzzRun \
	./internal/response/model/jsonview:FuzzCompact \
	./internal/request/adapter/proto/ws:FuzzReadFrame \
	./uitoolkit/markup:FuzzParse \
	./uitoolkit/widget:FuzzEdit \
	./uitoolkit/widget:FuzzFind \
	./uitoolkit/widget:FuzzWrapRunes \
	./internal/request/model:FuzzParseSchema \
	./internal/request/model:FuzzParseMCPLists \
	./internal/request/model:FuzzParseServerInfo \
	./internal/request/model:FuzzParseProtocol \
	./internal/request/model:FuzzParseBodyType \
	./internal/request/model:FuzzParseAuthType \
	./internal/collection/adapter/store:FuzzLoad \
	./internal/workspace/adapter/statefile:FuzzLoad \
	./internal/settings/model/settings:FuzzDecode \
	./internal/settings/model/fileversion:FuzzRead \
	./internal/response/adapter/history:FuzzLoad \
	./internal/response/adapter/history:FuzzLoadSent \
	./internal/response/ui/respview:FuzzDecode \
	./uitoolkit/theming:FuzzParseTheme \
	./internal/request/adapter/proto:FuzzReadEvents \
	./internal/settings/model/settings:FuzzParse \
	./internal/request/model:FuzzParseBulk \
	./internal/settings/ui/shortcutpage:FuzzParseBinding
fuzz:
	@for target in $(FUZZ_TARGETS); do \
		package=$${target%%:*}; name=$${target##*:}; \
		echo "fuzz $$name ($$package)"; \
		go test $$package -run '^$$' -fuzz "^$$name$$" -fuzztime $(FUZZTIME) || exit 1; \
	done

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

# Install the release build, its desktop entry and icon for the current user, no root.
# System-wide: make install PREFIX=/usr/local
PREFIX ?= $(HOME)/.local
install: release
	install -Dm755 bin/workshop-release $(DESTDIR)$(PREFIX)/bin/gascoigne-workshop
	install -Dm644 packaging/gascoigne-workshop.desktop $(DESTDIR)$(PREFIX)/share/applications/gascoigne-workshop.desktop
	mkdir -p $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps
	bin/workshop-release -icon-png $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps/gascoigne-workshop.png
