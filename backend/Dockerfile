FROM golang:1.21-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o ancient-wood-server \
    ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

ENV TZ=Asia/Shanghai
ENV CONFIG_PATH=/app/config/config.yaml

WORKDIR /app

COPY --from=builder /build/ancient-wood-server /app/
COPY --from=builder /build/models /app/models
COPY --from=builder /build/config/config.yaml /app/config/

EXPOSE 8080 6060

HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
    CMD wget --spider -q http://localhost:8080/api/v1/health || exit 1

CMD ["/app/ancient-wood-server"]
