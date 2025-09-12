# rama-swap

`rama-swap` is a simple alternative to llama-swap that wraps ramalama.
It is designed to make using ramalama as easy as ollama.

## Running

`rama-swap` can be run using standard go tooling.
You will need to have ramalama installed on the host.

```bash
go run github.com/wk-y/rama-swap@latest
```

## Podman/Docker

`rama-swap` can be built to run in a container.
In this mode, all model servers will run in the same container as well.

```bash
# build the container
git clone http://github.com/wk-y/rama-swap
cd rama-swap
podman build . -t rama-swap

# run the container (you will need to add flags to enable gpu inference)
podman run --rm -v ~/.local/share/ramalama:/app/store:ro,Z -p 127.0.0.1:4917:4917 rama-swap
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
