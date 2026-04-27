# Dockerfile (API Server)
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /api /usr/local/bin/
COPY --from=builder /app/migrations /migrations

EXPOSE 8081
ENTRYPOINT ["api"]
CMD ["--config", "/config/config.yaml"]