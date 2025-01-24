# Используем официальный образ Go
FROM golang:1.23.4-alpine

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы go.mod и go.sum для установки зависимостей
COPY go.mod go.sum ./

# Устанавливаем зависимости
RUN go mod download

# Копируем исходный код в контейнер
COPY . .

# Собираем приложение
RUN go build -o payment-system ./cmd/main.go

# Открываем порт для доступа к приложению
EXPOSE 8080

# Команда для запуска приложения
CMD ["./payment-system"]