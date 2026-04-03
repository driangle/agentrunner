package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/driangle/agentrunner/go/channel"
)

// mcpServerConfig is one entry in the MCP configuration file.
type mcpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// mcpConfig is the top-level MCP configuration file format.
type mcpConfig struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

// channelSetup holds the artifacts created by setupChannel.
type channelSetup struct {
	sockPath      string
	mcpConfigPath string
	cleanup       func()
}

// setupChannel prepares the channel infrastructure for a Claude CLI invocation.
// It resolves the agentrunner-channel binary, creates a temp directory with a
// Unix socket path and MCP config file, and optionally merges the user's
// existing MCP config.
func setupChannel(ctx context.Context, co *ClaudeOptions) (*channelSetup, error) {
	binPath, err := channel.BinaryPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving channel binary: %w", err)
	}

	// Use /tmp for short socket paths (macOS has a 104-char limit).
	tmpDir, err := os.MkdirTemp("/tmp", "ar-ch-")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	sockPath := filepath.Join(tmpDir, "ch.sock")

	env := map[string]string{
		"AGENTRUNNER_CHANNEL_SOCK": sockPath,
	}
	if co.ChannelLogFile != "" {
		env["AGENTRUNNER_CHANNEL_LOG"] = co.ChannelLogFile
	}
	if co.ChannelLogLevel != "" {
		env["AGENTRUNNER_CHANNEL_LOG_LEVEL"] = co.ChannelLogLevel
	}

	cfg := mcpConfig{
		MCPServers: map[string]mcpServerConfig{
			"agentrunner-channel": {
				Command: binPath,
				Env:     env,
			},
		},
	}

	// Merge with user's MCP config if provided.
	if co.MCPConfig != "" {
		userData, err := os.ReadFile(co.MCPConfig)
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("reading user MCP config %s: %w", co.MCPConfig, err)
		}
		var userCfg mcpConfig
		if err := json.Unmarshal(userData, &userCfg); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("parsing user MCP config: %w", err)
		}
		for k, v := range userCfg.MCPServers {
			if k != "agentrunner-channel" {
				cfg.MCPServers[k] = v
			}
		}
	}

	cfgData, err := json.Marshal(cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("marshaling MCP config: %w", err)
	}

	cfgPath := filepath.Join(tmpDir, "mcp.json")
	if err := os.WriteFile(cfgPath, cfgData, 0o600); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("writing MCP config: %w", err)
	}

	return &channelSetup{
		sockPath:      sockPath,
		mcpConfigPath: cfgPath,
		cleanup:       func() { os.RemoveAll(tmpDir) },
	}, nil
}
