// Package main отвечает за инициализацию основных компонентов программы и запуск основного цикла выполнения.
// Пакет предоставляет REST API для отправки денег, получения информации о транзакциях и проверки баланса кошельков
// Ссылка на git: https://github.com/Arkadiy-GO/payment-system/tree/master
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	handlers "payment-system/internal/api"
	repository "payment-system/internal/db"
	service "payment-system/internal/service"

	"github.com/gorilla/mux"
)

// Config содержит конфигурационные параметры приложения.
type Config struct {
	Port string // Порт, на котором будет запущен сервер
}

// main инициализирует репозиторий, сервис и маршрутизатор для обработки HTTP-запросов.
// Репозиторий отвечает за взаимодействие с базой данных, сервис — за бизнес-логику.
// С помощью библиотеки Gorilla Mux создаются маршруты и привязываются соответствующие обработчики.
// Функция также запускает HTTP-сервер с поддержкой graceful shutdown.
func main() {
	// Инициализация конфигурации
	cfg := Config{
		Port: getEnv("PORT", "8080"), // Порт по умолчанию: 8080
	}

	// Инициализация репозитория для работы с базой данных
	repo := repository.NewPostgresRepository()

	// Проверка подключения к базе данных
	if err := repo.Ping(context.Background()); err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	// Инициализация сервиса, который содержит бизнес-логику приложения
	svc := service.NewService(repo)

	// Создание маршрутизатора с использованием библиотеки Gorilla Mux
	router := mux.NewRouter()

	// Регистрация обработчиков для API:
	// - POST /api/send: Отправляет деньги с одного кошелька на другой
	router.HandleFunc("/api/send", handlers.SendHandler(svc)).Methods("POST")

	// - GET /api/transactions: Возвращает информацию о последних N транзакциях
	router.HandleFunc("/api/transactions", handlers.GetLastHandler(svc)).Methods("GET")

	// - GET /api/wallet/{address}/balance: Возвращает баланс указанного кошелька
	router.HandleFunc("/api/wallet/{address}/balance", handlers.GetBalanceHandler(svc)).Methods("GET")

	// Создание HTTP-сервера
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Канал для graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера в отдельной горутине
	go func() {
		log.Printf("Запуск сервера на порту %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Ожидание сигнала для graceful shutdown
	<-done
	log.Println("Сервер завершает работу...")

	// Создание контекста с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Завершение работы сервера
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при завершении работы сервера: %v", err)
	}

	log.Println("Сервер успешно завершил работу")
}

// getEnv возвращает значение переменной окружения или значение по умолчанию.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
