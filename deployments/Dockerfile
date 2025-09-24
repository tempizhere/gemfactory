# Многоэтапная сборка для продакшена
FROM golang:1.24.1-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем пользователя для сборки
RUN adduser -D -g '' appuser

# Устанавливаем рабочую директорию
WORKDIR /build

# Копируем go.mod и go.sum
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o gemfactory cmd/bot/main.go

# Финальный образ
FROM scratch

# Копируем сертификаты и временную зону
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Копируем пользователя
COPY --from=builder /etc/passwd /etc/passwd

# Копируем собранное приложение
COPY --from=builder /build/gemfactory /app/gemfactory

# Создаем директорию для логов
RUN mkdir -p /app/logs

# Переключаемся на непривилегированного пользователя
USER appuser

# Устанавливаем рабочую директорию
WORKDIR /app

# Открываем порт для health check
EXPOSE 8080

# Запускаем приложение
CMD ["./gemfactory"]
