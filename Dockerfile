FROM docker.io/golang:1.25.1@sha256:9c0c9e535601de9c1dd1e8a4dddbce5830359782f700291175ca47c1ef1a6e67 AS builder
WORKDIR /rama-swap

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o rama-swap .

FROM quay.io/ramalama/ramalama:0.12.2@sha256:f8007047df1fa7873cddf702e6da6152d83afcb835d17e26b1b701c36c5e0be1
COPY --from=builder /rama-swap/rama-swap /usr/local/bin/rama-swap

ENTRYPOINT [ "env", "RAMALAMA_STORE=/app/store", "rama-swap", "-ramalama", "ramalama", "--nocontainer", ";", "-host", "0.0.0.0", "-port", "4917" ]
EXPOSE 4917
