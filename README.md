# rama-swap

rama-swap is a simple alternative to llama-swap that wraps ramalama.
It is designed to make using ramalama as easy as ollama:

```bash
ramalama pull qwen3:0.6b
go run github.com/wk-y/rama-swap@latest
```

## Configuration

The `RAMALAMA_COMMAND` environment variable can be set to change the underlying command used (ex. `RAMALAMA_COMMAND="uvx ramalama"`)

## Endpoints

The following OpenAI compatible endpoints are proxied to the underlying ramalama instances:

- [x] `/v1/models`
- [x] `/v1/completions`
- [x] `/v1/chat/completions`

Ollama-compatible endpoints are also implemented:

- [x] `/api/version`
- [x] `/api/tags`$^1$
- [x] `/api/chat`$^1$

$^1$ Some features are not yet supported.

Similar to `llama-swap`, upstream endpoints can be accessed via the `/upstream/{model}/...` endpoints.
Models with slashes in their name are accessible through `/upstream` by replacing the slashes with underscores.
