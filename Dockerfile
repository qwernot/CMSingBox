FROM node:20-bookworm-slim AS frontend
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.24-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown
ARG LICENSE_PUBLIC_KEY=""
ARG FREE_SUBSCRIPTION_LIMIT=1
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT} -X main.LicensePublicKey=${LICENSE_PUBLIC_KEY} -X main.FreeSubscriptionLimit=${FREE_SUBSCRIPTION_LIMIT}" -o /out/cmsingbox ./cmd/sbm

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates nftables iproute2 tzdata && rm -rf /var/lib/apt/lists/*
COPY --from=backend /out/cmsingbox /usr/local/bin/cmsingbox
VOLUME ["/data"]
EXPOSE 9092/tcp 2080/tcp 53/tcp 53/udp
ENTRYPOINT ["cmsingbox", "-data", "/data", "-port", "9092"]
