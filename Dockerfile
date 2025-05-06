# Этап сборки
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gemfactory cmd/bot/main.go

# Финальный образ
FROM alpine:3.18
WORKDIR /app
# Устанавливаем tzdata для поддержки часовых поясов
RUN apk add --no-cache tzdata && \
    rm -rf /var/cache/apk/*
COPY --from=builder /app/gemfactory .
COPY internal/telegrambot/releases/data/ /app/internal/telegrambot/releases/data/
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Добавляем HEALTHCHECK
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD pgrep gemfactory || exit 1

# Создаем non-root пользователя
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Запускаем через entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]