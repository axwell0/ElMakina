FROM golang:1.26-alpine AS dev

WORKDIR /repo
RUN apk add --no-cache git wget

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

CMD ["go", "run", "./server/cmd/server"]

FROM golang:1.26-alpine AS builder

WORKDIR /repo
RUN apk add --no-cache git

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/elmakina-server ./server/cmd/server

FROM alpine:3.20 AS prod

RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/elmakina-server /app/elmakina-server
USER app
EXPOSE 8080

CMD ["/app/elmakina-server"]
