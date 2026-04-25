FROM golang:1.22-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

ENV GOPROXY=https://goproxy.cn,direct

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o cloud-disk ./main.go

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 appgroup && \
    adduser -u 1001 -G appgroup -D appuser && \
    mkdir -p /app/uploads && \
    chown -R appuser:appgroup /app

USER appuser

COPY --from=builder /build/cloud-disk /app/cloud-disk

EXPOSE 8080

CMD ["/app/cloud-disk"]
