// Package api предоставляет HTTP-обработчики для работы с платежной системой.
// Включает методы для отправки денег, получения информации о транзакциях и проверки баланса кошелька.
package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	service "payment-system/internal/service"

	"github.com/gorilla/mux"
)

// SendHandler возвращает HTTP-обработчик для отправки денег с одного кошелька на другой.
//
// Параметры:
//   - svc: Сервис для работы с бизнес-логикой.
//
// Возвращает:
//   - HTTP-обработчик.
//
// Пример использования:
//
//	router.HandleFunc("/api/send", SendHandler(svc)).Methods("POST")
func SendHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода запроса
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Проверка заголовка Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Invalid Content-Type, expected application/json", http.StatusBadRequest)
			return
		}

		// Декодирование JSON
		var req struct {
			From   string  `json:"from"`
			To     string  `json:"to"`
			Amount float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация данных
		if req.Amount <= 0 {
			http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
			return
		}

		// Проверка формата адресов кошельков
		if !isValidAddress(req.From) || !isValidAddress(req.To) {
			http.Error(w, "Invalid wallet address", http.StatusBadRequest)
			return
		}

		// Вызов сервиса
		if err := svc.Send(req.From, req.To, req.Amount); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// GetLastHandler возвращает HTTP-обработчик для получения информации о последних N транзакциях.
//
// Параметры:
//   - svc: Сервис для работы с бизнес-логикой.
//
// Возвращает:
//   - HTTP-обработчик.
//
// Пример использования:
//
//	router.HandleFunc("/api/transactions", GetLastHandler(svc)).Methods("GET")
func GetLastHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получение параметра count из query-строки
		countStr := r.URL.Query().Get("count")
		count, err := strconv.Atoi(countStr)
		if err != nil || count <= 0 {
			http.Error(w, "Invalid count parameter", http.StatusBadRequest)
			return
		}

		// Получение последних транзакций
		transactions, err := svc.GetLastTransactions(count)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Отправка ответа в формате JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(transactions); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// GetBalanceHandler возвращает HTTP-обработчик для получения баланса кошелька.
//
// Параметры:
//   - svc: Сервис для работы с бизнес-логикой.
//
// Возвращает:
//   - HTTP-обработчик.
//
// Пример использования:
//
//	router.HandleFunc("/api/wallet/{address}/balance", GetBalanceHandler(svc)).Methods("GET")
func GetBalanceHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получение адреса кошелька из пути запроса
		address := mux.Vars(r)["address"]

		// Проверка формата адреса кошелька
		if !isValidAddress(address) {
			http.Error(w, "Invalid wallet address", http.StatusBadRequest)
			return
		}

		// Получение баланса кошелька
		balance, err := svc.GetBalance(address)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Отправка ответа в формате JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]float64{"balance": balance}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// isValidAddress проверяет, что адрес состоит из 64 шестнадцатеричных символов.
//
// Параметры:
//   - address: Адрес кошелька.
//
// Возвращает:
//   - true, если адрес валиден, иначе false.
func isValidAddress(address string) bool {
	if len(address) != 64 {
		return false
	}
	_, err := hex.DecodeString(address)
	return err == nil
}
