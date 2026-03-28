package channel

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	buildOnce   sync.Once
	buildPath   string
	buildErr    error
	cacheSubdir = filepath.Join("agentrunner", "bin")
)

// BinaryPath returns the path to the agentrunner-channel binary.
//
// Resolution order:
//  1. AGENTRUNNER_CHANNEL_BIN environment variable
//  2. agentrunner-channel on $PATH (via exec.LookPath)
//  3. Build from source into the user cache directory (one-time)
func BinaryPath(ctx context.Context) (string, error) {
	if p := os.Getenv("AGENTRUNNER_CHANNEL_BIN"); p != "" {
		return p, nil
	}

	if p, err := exec.LookPath("agentrunner-channel"); err == nil {
		return p, nil
	}

	return buildFromSource(ctx)
}

// buildFromSource compiles the agentrunner-channel binary into the user's
// cache directory. The build runs at most once per process.
func buildFromSource(ctx context.Context) (string, error) {
	buildOnce.Do(func() {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			buildErr = fmt.Errorf("determining cache directory: %w", err)
			return
		}

		binDir := filepath.Join(cacheDir, cacheSubdir)
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			buildErr = fmt.Errorf("creating cache directory: %w", err)
			return
		}

		binPath := filepath.Join(binDir, "agentrunner-channel")
		cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
			"-trimpath", "-ldflags=-s -w",
			"github.com/driangle/agentrunner/go/cmd/agentrunner-channel",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			buildErr = fmt.Errorf("building agentrunner-channel: %w\n%s", err, out)
			return
		}

		buildPath = binPath
	})

	return buildPath, buildErr
}
