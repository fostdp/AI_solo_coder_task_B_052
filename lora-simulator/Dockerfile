FROM golang:1.21-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o lora-simulator \
    .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl

ENV TZ=Asia/Shanghai
ENV API_URL=http://backend:8080/api/v1/lora/data
ENV DEVICE_COUNT=50
ENV REPORT_INTERVAL=1h
ENV SIMULATION_SPEED=1

WORKDIR /app

COPY --from=builder /build/lora-simulator /app/

EXPOSE 8081

HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
    CMD curl -f http://localhost:8081/health || exit 1

CMD ["/app/lora-simulator"]
