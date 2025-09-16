FROM docker.io/golang:1.25.1 AS builder
WORKDIR /rama-swap

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o rama-swap .

FROM quay.io/ramalama/ramalama:latest
COPY --from=builder /rama-swap/rama-swap /usr/local/bin/rama-swap

CMD RAMALAMA_STORE=/app/store rama-swap -ramalama ramalama --nocontainer \; -host 0.0.0.0 -port 4917
EXPOSE 4917
