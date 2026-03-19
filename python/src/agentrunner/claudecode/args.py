"""Build CLI arguments from prompt and options."""

from __future__ import annotations

from .options import ClaudeRunOptions


def build_args(prompt: str, options: ClaudeRunOptions | None = None) -> list[str]:
    """Build CLI arguments from prompt and options."""
    args = ["--print", "--output-format", "stream-json", "--verbose"]

    if options is None:
        args.extend(["--", prompt])
        return args

    # Common options.
    if options.model:
        args.extend(["--model", options.model])
    if options.system_prompt:
        args.extend(["--system-prompt", options.system_prompt])
    if options.append_system_prompt:
        args.extend(["--append-system-prompt", options.append_system_prompt])
    if options.max_turns is not None and options.max_turns > 0:
        args.extend(["--max-turns", str(options.max_turns)])
    if options.skip_permissions:
        args.append("--dangerously-skip-permissions")

    # Claude-specific options.
    if options.allowed_tools:
        for tool in options.allowed_tools:
            args.extend(["--allowedTools", tool])
    if options.disallowed_tools:
        for tool in options.disallowed_tools:
            args.extend(["--disallowedTools", tool])
    if options.mcp_config:
        args.extend(["--mcp-config", options.mcp_config])
    if options.json_schema:
        args.extend(["--json-schema", options.json_schema])
    if options.max_budget_usd is not None and options.max_budget_usd > 0:
        args.extend(["--max-budget-usd", str(options.max_budget_usd)])
    if options.resume:
        args.extend(["--resume", options.resume])
    if options.continue_session:
        args.append("--continue")
    if options.session_id:
        args.extend(["--session-id", options.session_id])
    if options.include_partial_messages:
        args.append("--include-partial-messages")

    args.extend(["--", prompt])
    return args
