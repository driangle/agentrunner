.PHONY: check check-go test-go lint-go build-go check-ts build-ts lint-ts test-ts

# Top-level target: check all libraries.
check: check-go check-ts

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
