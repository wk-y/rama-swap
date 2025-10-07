FROM docker.io/golang:1.25.1@sha256:3c9619997c330b7e48c1dd3280444fccaf1d3b68c10c63fbba7d3461a6b61b3f AS builder
WORKDIR /rama-swap

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o rama-swap .

FROM quay.io/ramalama/ramalama:0.12.4@sha256:9811684f49643db589859283125cee9ec615a413c43f355bab9fa646786f43b4
COPY --from=builder /rama-swap/rama-swap /usr/local/bin/rama-swap

ENTRYPOINT [ "env", "RAMALAMA_STORE=/app/store", "rama-swap", "-ramalama", "ramalama", "--nocontainer", ";", "-host", "0.0.0.0", "-port", "4917" ]
EXPOSE 4917
