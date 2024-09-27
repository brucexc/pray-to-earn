FROM ghcr.io/rss3-network/go-image/go-builder AS base

WORKDIR /root/pray

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

COPY . .

FROM base AS builder

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go build cmd/main.go

FROM ghcr.io/rss3-network/go-image/go-runtime AS runner

WORKDIR /root/pray

COPY --from=builder /root/pray/main ./pray
RUN mkdir -p deploy
COPY deploy/config.yaml /root/pray/deploy/config.yaml

EXPOSE 80
ENTRYPOINT ["./pray"]
