.PHONY: check check-go test-go lint-go build-go

# Top-level target: check all libraries.
check: check-go

# --- Go ---

check-go: build-go lint-go test-go

build-go:
	cd go && go build ./...

lint-go:
	cd go && go vet ./...

test-go:
	cd go && go test ./...
