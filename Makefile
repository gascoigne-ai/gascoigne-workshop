.PHONY: check test-formal bench fuzz fmt-check vet test vet-release release-packages check-release-scope test-release build release run example-server shot install

check: fmt-check vet test test-formal vet-release check-release-scope test-release

fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet:
	go vet ./...

test:
	go test -race ./...

# The formal claims: every claim with the evidence kind behind it, and the lock that makes a
# change of a claim a recorded decision (.harness/architecture/formal.md).
test-formal:
	FORMAL_VISUAL=off go run github.com/gascoigne-ai/gascoigne-formal/cmd/formal check ./formal/...

# Frame-cost benchmarks (CPU side, no display needed beyond what the tests
# use); not part of check. Compare two runs with benchstat when installed.
BENCHCOUNT ?= 6
bench:
	go test -run '^$$' -bench . -benchmem -count $(BENCHCOUNT) ./internal/ui/window ./uitoolkit/...

# Short run of every fuzz target (package:Target pairs below); not part of
# check. A crasher lands in the package's testdata/fuzz and becomes a
# regression test: fix it, keep the file.
FUZZTIME ?= 30s
FUZZ_TARGETS = \
	./internal/feature/collections/importers/miniyaml:FuzzParse \
	./internal/feature/collections/importers/openapi:FuzzParse \
	./internal/feature/collections/importers/postman:FuzzParse \
	./internal/feature/collections/clientconfig:FuzzParse \
	./internal/feature/send:FuzzParseCurl \
	./internal/feature/request:FuzzDecodeJSONSchema \
	./internal/feature/request:FuzzURLParams \
	./internal/feature/send/assert:FuzzRun \
	./internal/ui/format:FuzzCompact \
	./internal/feature/send/adapter/protocol/ws:FuzzReadFrame \
	./internal/feature/send/adapter/protocol:FuzzMCPDeliver \
	./uitoolkit/markup:FuzzParse \
	./uitoolkit/widget:FuzzEdit \
	./uitoolkit/widget:FuzzFind \
	./uitoolkit/widget:FuzzParseSVG \
	./uitoolkit/widget:FuzzWrapRunes \
	./internal/feature/send:FuzzParseSchema \
	./internal/feature/request:FuzzParseMCPLists \
	./internal/feature/request:FuzzParseServerInfo \
	./internal/feature/request:FuzzParseProtocol \
	./internal/feature/request:FuzzParseBodyType \
	./internal/feature/request:FuzzParseAuthType \
	./internal/feature/collections/adapter/collectionstore:FuzzLoad \
	./internal/feature/workspace/adapter/statefile:FuzzLoad \
	./internal/feature/settings:FuzzDecode \
	./internal/storage:FuzzRead \
	./internal/feature/history/adapter/historyfile:FuzzLoad \
	./internal/feature/history/adapter/historyfile:FuzzLoadSent \
	./internal/ui/response/respview:FuzzDecode \
	./uitoolkit/theming:FuzzParseTheme \
	./internal/feature/send/adapter/protocol:FuzzReadEvents \
	./internal/feature/settings:FuzzParse \
	./internal/feature/request:FuzzParseBulk \
	./internal/ui/overlays/settingsdialog/shortcutpage:FuzzParseBinding \
	./internal/feature/send/adapter/protocol:FuzzDeliverJSON \
	./internal/feature/send/adapter/protocol:FuzzAnswerServer \
	./internal/feature/send/adapter/protocol:FuzzSample \
	./internal/feature/request:FuzzToolFields \
	./internal/feature/request:FuzzArguments \
	./internal/ui/format:FuzzLines \
	./uitoolkit/desktop/displayscale:FuzzParseMonitors \
	./uitoolkit/desktop/keyrepeat:FuzzParseHyprland \
	./uitoolkit/desktop/keyrepeat:FuzzParseGSettings \
	./uitoolkit/desktop/keyrepeat:FuzzParseXset \
	./uitoolkit/desktop/keyrepeat:FuzzParseMac \
	./uitoolkit/desktop/keyrepeat:FuzzParseWindows
fuzz:
	@for target in $(FUZZ_TARGETS); do \
		package=$${target%%:*}; name=$${target##*:}; \
		echo "fuzz $$name ($$package)"; \
		go test $$package -run '^$$' -fuzz "^$$name$$" -fuzztime $(FUZZTIME) || exit 1; \
	done

# The release build has its own files (build tag "release"); check them too.
vet-release:
	go vet -tags release ./...

# Directories that hold a file with a release build tag.
RELEASE_BASE = cmd/workshop internal/feature/collections internal/storage

# The release test run covers RELEASE_BASE and every package whose build or
# test binary links one of them: the rest compile the same under both tags.
# Prints import paths, computed from the import graph so it cannot drift.
release-packages:
	@module=$$(go list -m); base=""; for dir in $(RELEASE_BASE); do base="$$base $$module/$$dir"; done; \
	go list -tags release -test -f '{{.ImportPath}} {{join .Deps " "}}' ./... | \
	awk -v base="$$base" 'BEGIN { count = split(base, names, " "); for (i = 1; i <= count; i++) isbase[names[i]] = 1 } \
		{ path = $$1; sub(/\.test$$/, "", path); hit = (path in isbase); \
		  for (i = 2; i <= NF; i++) if ($$i in isbase) hit = 1; \
		  if (hit) print path }' | sort -u

# Fails when a file with a release build tag sits in a package that
# test-release does not run: add its directory to RELEASE_BASE.
check-release-scope:
	@module=$$(go list -m); covered=$$($(MAKE) -s release-packages); status=0; \
	for dir in $$(grep -rlE '^//go:build (.* )?release\b' --include='*.go' . | xargs -n1 dirname | sort -u); do \
		path=$$module/$${dir#./}; \
		echo "$$covered" | grep -qxF "$$path" || { echo "release-tagged file in $$dir is outside the test-release set; add it to RELEASE_BASE"; status=1; }; \
	done; exit $$status

test-release:
	go test -race -tags release $$($(MAKE) -s release-packages)

build:
	go build -o bin/workshop ./cmd/workshop
	go build -o bin/shot ./e2e/cmd/shot

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
	go run ./e2e/cmd/shot -screenshot $(or $(OUT),shot.png) -send

# Install the release build for the current user, no root; on Linux also its
# desktop entry and icon (freedesktop paths that macOS does not read; the Mac
# app is packaging/macos/make-app.sh). mkdir -p and install -m rather than
# GNU install -D, so BSD install runs it too.
# System-wide: make install PREFIX=/usr/local
PREFIX ?= $(HOME)/.local
UNAME := $(shell uname -s)
install: release
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	install -m 755 bin/workshop-release $(DESTDIR)$(PREFIX)/bin/gascoigne-workshop
ifeq ($(UNAME),Linux)
	mkdir -p $(DESTDIR)$(PREFIX)/share/applications $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps
	install -m 644 packaging/gascoigne-workshop.desktop $(DESTDIR)$(PREFIX)/share/applications/gascoigne-workshop.desktop
	bin/workshop-release -icon-png $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps/gascoigne-workshop.png
endif
