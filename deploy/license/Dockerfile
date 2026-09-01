FROM golang:1.24-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cmsingbox-license-server ./cmd/license-server && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cmsingbox-license-tool ./cmd/license-tool

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/cmsingbox-license-server /usr/local/bin/cmsingbox-license-server
COPY --from=builder /out/cmsingbox-license-tool /usr/local/bin/cmsingbox-license-tool
VOLUME ["/license"]
EXPOSE 9093/tcp
ENTRYPOINT ["cmsingbox-license-server", "-listen", "0.0.0.0:9093", "-private-key", "/license/private.key", "-audit", "/license/license-audit.jsonl"]
