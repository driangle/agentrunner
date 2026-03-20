---
name: release-go
description: "Release the Go library by running checks, tagging, and pushing. Use when the user says /release-go <version> or asks to release the Go module."
user_invocable: true
argument: "version (e.g., 0.1.0)"
---

# Release Go Library

Release the Go agentrunner library by validating, tagging, and pushing.

## Instructions

The user provides a version number in `$ARGUMENTS` (e.g., `0.1.0`).

Run the release script:

```bash
./scripts/release.sh go check-go agentrunner "$ARGUMENTS"
```

This will:
1. Check the working tree is clean
2. Run `make check-go` to validate
3. Create a git tag `agentrunner/v<version>`
4. Push the tag to origin

The Go module proxy will pick up the tag automatically. The `publish-go` GitHub Actions workflow will validate the module and create a GitHub Release.

If the script fails, report the error to the user.
