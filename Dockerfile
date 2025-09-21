FROM docker.io/golang:1.25.1@sha256:8305f5fa8ea63c7b5bc85bd223ccc62941f852318ebfbd22f53bbd0b358c07e1 AS builder
WORKDIR /rama-swap

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o rama-swap .

FROM quay.io/ramalama/ramalama:0.12.2@sha256:f8007047df1fa7873cddf702e6da6152d83afcb835d17e26b1b701c36c5e0be1
COPY --from=builder /rama-swap/rama-swap /usr/local/bin/rama-swap

ENTRYPOINT [ "env", "RAMALAMA_STORE=/app/store", "rama-swap", "-ramalama", "ramalama", "--nocontainer", ";", "-host", "0.0.0.0", "-port", "4917" ]
EXPOSE 4917
