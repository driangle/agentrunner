"""Tests for CLI argument building."""

from agentrunner.claudecode.args import build_args
from agentrunner.claudecode.options import ClaudeRunOptions


class TestBuildArgs:
    def test_minimal(self):
        args = build_args("hello")
        assert args == ["--print", "--output-format", "stream-json", "--verbose", "--", "hello"]

    def test_with_model(self):
        opts = ClaudeRunOptions(model="claude-sonnet-4-6")
        args = build_args("hello", opts)
        assert "--model" in args
        assert args[args.index("--model") + 1] == "claude-sonnet-4-6"

    def test_with_system_prompt(self):
        opts = ClaudeRunOptions(system_prompt="be helpful")
        args = build_args("hello", opts)
        assert "--system-prompt" in args
        assert args[args.index("--system-prompt") + 1] == "be helpful"

    def test_with_append_system_prompt(self):
        opts = ClaudeRunOptions(append_system_prompt="extra context")
        args = build_args("hello", opts)
        assert "--append-system-prompt" in args

    def test_with_max_turns(self):
        opts = ClaudeRunOptions(max_turns=5)
        args = build_args("hello", opts)
        assert "--max-turns" in args
        assert args[args.index("--max-turns") + 1] == "5"

    def test_max_turns_zero_skipped(self):
        opts = ClaudeRunOptions(max_turns=0)
        args = build_args("hello", opts)
        assert "--max-turns" not in args

    def test_skip_permissions(self):
        opts = ClaudeRunOptions(skip_permissions=True)
        args = build_args("hello", opts)
        assert "--dangerously-skip-permissions" in args

    def test_allowed_tools(self):
        opts = ClaudeRunOptions(allowed_tools=["Read", "Write"])
        args = build_args("hello", opts)
        indices = [i for i, a in enumerate(args) if a == "--allowedTools"]
        assert len(indices) == 2
        assert args[indices[0] + 1] == "Read"
        assert args[indices[1] + 1] == "Write"

    def test_disallowed_tools(self):
        opts = ClaudeRunOptions(disallowed_tools=["Bash"])
        args = build_args("hello", opts)
        assert "--disallowedTools" in args
        assert args[args.index("--disallowedTools") + 1] == "Bash"

    def test_mcp_config(self):
        opts = ClaudeRunOptions(mcp_config="/path/to/config.json")
        args = build_args("hello", opts)
        assert "--mcp-config" in args

    def test_json_schema(self):
        opts = ClaudeRunOptions(json_schema='{"type":"object"}')
        args = build_args("hello", opts)
        assert "--json-schema" in args

    def test_max_budget_usd(self):
        opts = ClaudeRunOptions(max_budget_usd=1.5)
        args = build_args("hello", opts)
        assert "--max-budget-usd" in args
        assert args[args.index("--max-budget-usd") + 1] == "1.5"

    def test_max_budget_zero_skipped(self):
        opts = ClaudeRunOptions(max_budget_usd=0)
        args = build_args("hello", opts)
        assert "--max-budget-usd" not in args

    def test_resume(self):
        opts = ClaudeRunOptions(resume="sess-123")
        args = build_args("hello", opts)
        assert "--resume" in args
        assert args[args.index("--resume") + 1] == "sess-123"

    def test_continue_session(self):
        opts = ClaudeRunOptions(continue_session=True)
        args = build_args("hello", opts)
        assert "--continue" in args

    def test_session_id(self):
        opts = ClaudeRunOptions(session_id="sess-456")
        args = build_args("hello", opts)
        assert "--session-id" in args

    def test_include_partial_messages(self):
        opts = ClaudeRunOptions(include_partial_messages=True)
        args = build_args("hello", opts)
        assert "--include-partial-messages" in args

    def test_prompt_after_separator(self):
        opts = ClaudeRunOptions(model="claude-sonnet-4-6")
        args = build_args("my prompt", opts)
        sep_idx = args.index("--")
        assert args[sep_idx + 1] == "my prompt"
