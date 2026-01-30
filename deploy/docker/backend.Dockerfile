# Development stage
FROM golang:1.25-alpine AS dev

WORKDIR /repo
RUN apk add --no-cache git wget

# Use BuildKit cache mounts for faster rebuilds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

CMD ["go", "run", "./backend/cmd/server"]

# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /repo
RUN apk add --no-cache git

# Use BuildKit cache mounts for dependencies and build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

COPY . .

# Use cache mounts for build cache during compilation
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/elmakina-server ./backend/cmd/server

# Production stage
FROM alpine:3.20 AS prod

RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/elmakina-server /app/elmakina-server
USER app
EXPOSE 8080
CMD ["/app/elmakina-server"]
