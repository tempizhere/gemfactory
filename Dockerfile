# Этап сборки
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gemfactory cmd/bot/main.go

# Финальный образ
FROM alpine:latest
WORKDIR /app
# Устанавливаем tzdata для поддержки часовых поясов
RUN apk add --no-cache tzdata
COPY --from=builder /app/gemfactory .
# Копируем начальные файлы вайтлистов в образ
COPY internal/telegrambot/releases/data/ /app/internal/telegrambot/releases/data/
# Создаём entrypoint скрипт для инициализации вайтлистов
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Запускаем через entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]