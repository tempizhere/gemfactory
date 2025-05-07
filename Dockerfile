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
RUN apk add --no-cache tzdata=2025b-r0 && \
    rm -rf /var/cache/apk/*
# Создаем non-root пользователя и группу
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# Создаем директории и устанавливаем права
RUN mkdir -p /app/data /app/internal/telegrambot/releases/data && \
    chown -R appuser:appgroup /app
# Копируем бинарник и entrypoint.sh
COPY --from=builder --chown=appuser:appgroup /app/gemfactory .
COPY --chown=appuser:appgroup entrypoint.sh /app/entrypoint.sh
# Устанавливаем права на выполнение
RUN chmod +x /app/entrypoint.sh
# Настраиваем HEALTHCHECK
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD pgrep gemfactory || exit 1
USER appuser
ENTRYPOINT ["/app/entrypoint.sh"]