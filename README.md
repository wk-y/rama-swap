# rama-swap

`rama-swap` is a simple alternative to llama-swap that wraps [ramalama](https://github.com/containers/ramalama).
It is designed to make using ramalama as easy as ollama.

## Running

`rama-swap` can be run as either a container image or directly on the host.

### Podman/Docker (recommended)

The `rama-swap` docker image bundles `rama-swap` with ramalama's inference container image.
All model inference servers will run inside one container.

```bash
# run the container (you will need to add flags to enable gpu inference)
podman run --rm -v ~/.local/share/ramalama:/app/store:ro,Z -p 127.0.0.1:4917:4917 ghcr.io/wk-y/rama-swap:master
```

### Without a container

`rama-swap` can be built/run using standard go tooling.

```bash
go run github.com/wk-y/rama-swap@latest
```

### Command-Line Flags

`rama-swap` supports a few command-line flags for configuration.
See <HELP.txt> or run `rama-swap -help` for the list of supported flags.

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

Similar to `llama-swap`, the `/upstream/{model}/...` endpoints provide access to the upstream model servers.
Models with slashes in their name are accessible through `/upstream` by replacing the slashes with underscores.
`/upstream/` provides links to each models' url.
