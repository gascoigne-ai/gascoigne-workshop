.PHONY: check fuzz fmt-check vet test vet-release release-packages check-release-scope test-release build release run example-server shot install

check: fmt-check vet test vet-release check-release-scope test-release

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
	./internal/collection/miniyaml:FuzzParse \
	./internal/collection/openapi:FuzzParse \
	./internal/collection/postman:FuzzParse \
	./internal/collection/clientconfig:FuzzParse \
	./internal/request:FuzzParseCurl \
	./internal/request:FuzzDecodeJSONSchema \
	./internal/request:FuzzURLParams \
	./internal/request/assert:FuzzRun \
	./internal/kernel/jsonview:FuzzCompact \
	./internal/request/adapter/proto/ws:FuzzReadFrame \
	./internal/request/adapter/proto:FuzzMCPDeliver \
	./uitoolkit/markup:FuzzParse \
	./uitoolkit/widget:FuzzEdit \
	./uitoolkit/widget:FuzzFind \
	./uitoolkit/widget:FuzzParseSVG \
	./uitoolkit/widget:FuzzWrapRunes \
	./internal/request:FuzzParseSchema \
	./internal/request:FuzzParseMCPLists \
	./internal/request:FuzzParseServerInfo \
	./internal/request:FuzzParseProtocol \
	./internal/request:FuzzParseBodyType \
	./internal/request:FuzzParseAuthType \
	./internal/collection/adapter/store:FuzzLoad \
	./internal/workspace/adapter/statefile:FuzzLoad \
	./internal/settings:FuzzDecode \
	./internal/kernel/fileversion:FuzzRead \
	./internal/response/adapter/history:FuzzLoad \
	./internal/response/adapter/history:FuzzLoadSent \
	./internal/response/ui/respview:FuzzDecode \
	./uitoolkit/theming:FuzzParseTheme \
	./internal/request/adapter/proto:FuzzReadEvents \
	./internal/settings:FuzzParse \
	./internal/request:FuzzParseBulk \
	./internal/settings/ui/shortcutpage:FuzzParseBinding \
	./internal/request/adapter/proto:FuzzDeliverJSON \
	./internal/request/adapter/proto:FuzzAnswerServer \
	./internal/request/adapter/proto:FuzzSample \
	./internal/request:FuzzToolFields \
	./internal/request:FuzzArguments \
	./internal/kernel/jsonview:FuzzLines \
	./internal/settings/adapter/displayscale:FuzzParseMonitors \
	./internal/settings/adapter/keyrepeat:FuzzParseHyprland \
	./internal/settings/adapter/keyrepeat:FuzzParseGSettings \
	./internal/settings/adapter/keyrepeat:FuzzParseXset \
	./internal/settings/adapter/keyrepeat:FuzzParseMac \
	./internal/settings/adapter/keyrepeat:FuzzParseWindows
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
RELEASE_BASE = cmd/workshop internal/collection internal/kernel/userdir

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
