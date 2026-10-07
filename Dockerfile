# ============================
# 1. Builder Stage
# ============================
FROM golang:alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Build binary with optimizations
RUN go build -ldflags="-w -s" -o server main.go

# ============================
# 2. Runner Stage (Production / Staging)
# ============================
FROM alpine:latest AS runner

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Jakarta

# Create non-root user for security
RUN adduser -D -u 10001 -s /bin/sh appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy database migrations so container can execute migrations if needed
COPY database/migrations /app/database/migrations

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 3000

ENTRYPOINT ["./server"]
CMD ["serve"]
