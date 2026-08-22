# syntax=docker/dockerfile:1

FROM golang:1.26.6-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/fijimunkii/chatgpt-plugin-tailscale/internal/version.Version=${VERSION} \
      -X github.com/fijimunkii/chatgpt-plugin-tailscale/internal/version.Commit=${COMMIT} \
      -X github.com/fijimunkii/chatgpt-plugin-tailscale/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/gateway ./cmd/gateway \
 && mkdir -p /out/state/tsnet \
 && chmod 0700 /out/state /out/state/tsnet

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="TetherBound Community / ChatGPT Plugin for Tailscale" \
      org.opencontainers.image.description="Self-hosted MCP gateway for bounded or operator access to admitted Linux hosts over Tailscale SSH" \
      org.opencontainers.image.url="https://tetherbound.dev" \
      org.opencontainers.image.source="https://github.com/fijimunkii/chatgpt-plugin-tailscale" \
      org.opencontainers.image.licenses="MIT"

COPY --from=build /out/gateway /usr/local/bin/gateway
COPY --from=build --chown=65532:65532 /out/state /var/lib/chatgpt-plugin-tailscale

USER 65532:65532
ENV TS_STATE_DIR=/var/lib/chatgpt-plugin-tailscale/tsnet
VOLUME ["/var/lib/chatgpt-plugin-tailscale"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gateway"]
