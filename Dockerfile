FROM docker.io/golang:1.25.1@sha256:d7098379b7da665ab25b99795465ec320b1ca9d4addb9f77409c4827dc904211 AS builder
WORKDIR /rama-swap

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o rama-swap .

FROM quay.io/ramalama/ramalama:0.12.3@sha256:6692d9055c5fa4b4bbe0cd3708c64e922b5bfb0240e29aea057d523e50897b6f
COPY --from=builder /rama-swap/rama-swap /usr/local/bin/rama-swap

ENTRYPOINT [ "env", "RAMALAMA_STORE=/app/store", "rama-swap", "-ramalama", "ramalama", "--nocontainer", ";", "-host", "0.0.0.0", "-port", "4917" ]
EXPOSE 4917
