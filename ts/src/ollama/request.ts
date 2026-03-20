import {
  RunnerError,
  TimeoutError,
  CancelledError,
  NotFoundError,
  HttpError,
  ParseError,
  NoResultError,
} from "../errors.js";
import type { OllamaRunnerConfig, OllamaRunOptions } from "./options.js";
import type { ChatRequest, ChatMessage, ModelOptions } from "./types.js";

export function buildRequestBody(
  prompt: string,
  options: OllamaRunOptions,
): ChatRequest {
  const messages = buildMessages(prompt, options);

  if (!options.model) {
    throw new RunnerError("model is required for Ollama runner");
  }

  const req: ChatRequest = {
    model: options.model,
    messages,
    stream: true,
  };

  if (options.think != null) {
    req.think = options.think;
  }
  if (options.format) {
    req.format = options.format;
  }
  if (options.keepAlive) {
    req.keep_alive = options.keepAlive;
  }

  const modelOpts = buildModelOptions(options);
  if (modelOpts) {
    req.options = modelOpts;
  }

  return req;
}

function buildMessages(
  prompt: string,
  options: OllamaRunOptions,
): ChatMessage[] {
  const messages: ChatMessage[] = [];

  let systemPrompt = options.systemPrompt ?? "";
  if (options.appendSystemPrompt) {
    if (systemPrompt) {
      systemPrompt += "\n" + options.appendSystemPrompt;
    } else {
      systemPrompt = options.appendSystemPrompt;
    }
  }

  if (systemPrompt) {
    messages.push({ role: "system", content: systemPrompt });
  }

  messages.push({ role: "user", content: prompt });
  return messages;
}

function buildModelOptions(
  options: OllamaRunOptions,
): ModelOptions | undefined {
  const hasAny =
    options.temperature != null ||
    options.numCtx != null ||
    options.numPredict != null ||
    options.seed != null ||
    options.stop != null ||
    options.topK != null ||
    options.topP != null ||
    options.minP != null;

  if (!hasAny) return undefined;

  const opts: ModelOptions = {};
  if (options.temperature != null) opts.temperature = options.temperature;
  if (options.numCtx != null) opts.num_ctx = options.numCtx;
  if (options.numPredict != null) opts.num_predict = options.numPredict;
  if (options.seed != null) opts.seed = options.seed;
  if (options.stop != null) opts.stop = options.stop;
  if (options.topK != null) opts.top_k = options.topK;
  if (options.topP != null) opts.top_p = options.topP;
  if (options.minP != null) opts.min_p = options.minP;
  return opts;
}

export function mapFetchError(err: unknown): Error {
  if (
    err instanceof TimeoutError ||
    err instanceof CancelledError ||
    err instanceof NotFoundError ||
    err instanceof HttpError ||
    err instanceof ParseError ||
    err instanceof NoResultError
  ) {
    return err;
  }

  if (
    err instanceof DOMException ||
    (err instanceof Error && err.name === "AbortError")
  ) {
    return new CancelledError("execution cancelled");
  }

  if (err instanceof TypeError) {
    return new NotFoundError(`connection failed: ${err.message}`);
  }

  return new NotFoundError(`request failed: ${err}`);
}

export function logRequest(config: OllamaRunnerConfig, baseURL: string): void {
  if (!config.logger) return;
  config.logger.debug("executing Ollama API request", {
    method: "POST",
    url: `${baseURL}/api/chat`,
  });
}
