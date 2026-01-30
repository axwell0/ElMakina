FROM golang:1.25-alpine AS dev

WORKDIR /repo
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download

CMD ["go", "run", "./backend/cmd/server"]

FROM golang:1.25-alpine AS builder

WORKDIR /repo
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/elmakina-server ./backend/cmd/server

FROM alpine:3.20 AS prod

RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/elmakina-server /app/elmakina-server
USER app
EXPOSE 8080
CMD ["/app/elmakina-server"]
