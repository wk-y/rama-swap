## Configuration

The `RAMALAMA_COMMAND` environment variable can be set to change the underlying command used (ex. `RAMALAMA_COMMAND="uvx ramalama")

## Endpoints

The following OpenAI compatible endpoints are proxied to the underlying ramalama instance:

[x] - `/v1/models`
[x] - `/v1/completion`
[x] - `/v1/chat/completion`

Similar to `llama-swap`, upstream models can be directly accessed via the `/upstream/{model}/...` endpoints.
Models with slashes in their name are accessible through `/upstream` by replacing the slashes with underscores.
