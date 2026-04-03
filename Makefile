.PHONY: check check-lite check-go test-go lint-go build-go check-ts build-ts lint-ts test-ts check-python build-python lint-python test-python docs-dev docs-build cross-compile clean install-channel

# Top-level target: check all libraries.
check: check-go check-ts check-python

# Lite target: build and lint only (no tests).
check-lite: build-go lint-go build-ts lint-ts build-python lint-python

# --- Go ---

check-go: build-go lint-go test-go

build-go:
	cd go && go build ./...

lint-go:
	cd go && go vet ./...

test-go:
	cd go && go test ./...

# --- TypeScript ---

check-ts: build-ts lint-ts test-ts

build-ts:
	cd ts && npm run build

lint-ts:
	cd ts && npm run lint

test-ts:
	cd ts && npm test

# --- Python ---

check-python: build-python lint-python test-python

build-python:
	cd python && pip install -e ".[dev]" --quiet

lint-python:
	cd python && ruff check src/ tests/

test-python:
	cd python && python -m pytest

# --- Channel binary ---

install-channel:
	cd go && go build -o /tmp/agentrunner-channel ./cmd/agentrunner-channel

# --- Cross-compilation ---

CHANNEL_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

cross-compile:
	@mkdir -p dist
	@for platform in $(CHANNEL_PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out="dist/agentrunner-channel-$${os}-$${arch}$${ext}"; \
		echo "Building $$out..."; \
		cd go && GOOS=$$os GOARCH=$$arch go build -o ../$$out -trimpath -ldflags="-s -w" ./cmd/agentrunner-channel && cd ..; \
	done

clean:
	rm -rf dist

# --- Docs ---

docs-dev:
	cd docs && npm run docs:dev

docs-build:
	cd docs && npm run docs:build
