.PHONY: check check-go test-go lint-go build-go check-ts build-ts lint-ts test-ts check-python build-python lint-python test-python

# Top-level target: check all libraries.
check: check-go check-ts check-python

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
