/** Callback invoked for each streaming message. */
export type OnMessageFn = (message: Message) => void;

/** Logger interface for debug output. Opt-in, disabled by default. */
export interface Logger {
  debug(message: string, ...args: unknown[]): void;
  error(message: string, ...args: unknown[]): void;
}

/** Identifies the kind of streaming message. */
export type MessageType =
  | "system"
  | "assistant"
  | "user"
  | "tool_use"
  | "tool_result"
  | "result"
  | "error"
  | "stream_event"
  | (string & {});

/** The unit of streaming output from runStream. */
export interface Message<T = unknown> {
  /** Identifies the message kind. */
  type: MessageType;

  /** Original JSON line for runner-specific parsing. */
  raw: string;

  /** Parsed payload from the runner. */
  data: T;
}

/** Token consumption counts. */
export interface Usage {
  /** Number of prompt/input tokens consumed. */
  inputTokens: number;

  /** Number of completion/output tokens generated. */
  outputTokens: number;

  /** Number of tokens written to cache (Claude only). */
  cacheCreationInputTokens?: number;

  /** Number of tokens read from cache (Claude only). */
  cacheReadInputTokens?: number;
}

/** Final output from a runner invocation. */
export interface Result {
  /** Final response text. */
  text: string;

  /** Whether the run ended in error. */
  isError: boolean;

  /** Process exit code (CLI runners) or 0 (API runners). */
  exitCode: number;

  /** Token counts. */
  usage: Usage;

  /** Estimated cost in USD (0 for local runners). */
  costUSD: number;

  /** Wall-clock duration in milliseconds. */
  durationMs: number;

  /** Session identifier for resumption. */
  sessionId: string;
}

/** Common options for a runner invocation. All fields are optional. */
export interface RunOptions {
  /** Model name or alias. */
  model?: string;

  /** System prompt override. */
  systemPrompt?: string;

  /** Appended to the default system prompt. */
  appendSystemPrompt?: string;

  /** Working directory for the subprocess. */
  workingDir?: string;

  /** Additional environment variables for the subprocess. */
  env?: Record<string, string>;

  /** Maximum number of agentic turns. */
  maxTurns?: number;

  /** Overall execution timeout in milliseconds. */
  timeout?: number;

  /** AbortSignal for cancellation. */
  signal?: AbortSignal;
}

/** Session encapsulates a running agent process. */
export interface Session<M extends Message = Message> {
  /** Iterable of messages as they arrive. */
  messages: AsyncIterable<M>;

  /** Resolves when the agent finishes with the final result. */
  result: Promise<Result>;

  /** Terminates the agent process. */
  abort(): void;

  /** Reserved for future write-side support. Throws "not yet supported". */
  send(input: unknown): void;
}

/** Runner executes prompts against an AI coding agent. */
export interface Runner<
  O extends RunOptions = RunOptions,
  M extends Message = Message,
> {
  /** Launch an agent process and return a Session for full control. */
  start(prompt: string, options?: O): Session<M>;

  /** Send a prompt and block until the agent finishes. */
  run(prompt: string, options?: O): Promise<Result>;

  /** Send a prompt and stream messages as they arrive. */
  runStream(prompt: string, options?: O): AsyncIterable<M>;
}
