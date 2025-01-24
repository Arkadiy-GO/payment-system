// Package service предоставляет бизнес-логику для работы с платежной системой.
// Включает методы для получения баланса кошелька, отправки денег и получения списка транзакций.
package service

import (
	db "payment-system/internal/db"
	models "payment-system/internal/models"
)

// Service представляет сервис для работы с платежной системой.
// Содержит методы для взаимодействия с репозиторием базы данных.
type Service struct {
	repo *db.PostgresRepository
}

// NewService создает новый экземпляр Service.
//
// Параметры:
//   - repo: Репозиторий для работы с базой данных.
//
// Возвращает:
//   - Указатель на новый экземпляр Service.
//
// Пример использования:
//
//	repo := db.NewPostgresRepository()
//	svc := service.NewService(repo)
func NewService(repo *db.PostgresRepository) *Service {
	return &Service{repo: repo}
}

// GetBalance возвращает баланс кошелька по его адресу.
//
// Параметры:
//   - address: Адрес кошелька.
//
// Возвращает:
//   - Баланс кошелька.
//   - Ошибку, если кошелек не найден или произошла другая ошибка.
//
// Пример использования:
//
//	balance, err := svc.GetBalance("some_address")
func (s *Service) GetBalance(address string) (float64, error) {
	return s.repo.GetBalance(address)
}

// Send выполняет перевод средств с одного кошелька на другой.
// Включает проверку баланса отправителя, обновление балансов и запись транзакции.
//
// Параметры:
//   - from: Адрес кошелька отправителя.
//   - to: Адрес кошелька получателя.
//   - amount: Сумма перевода.
//
// Возвращает:
//   - Ошибку, если перевод не удался (например, недостаточно средств).
//
// Пример использования:
//
//	err := svc.Send("from_address", "to_address", 10.5)
func (s *Service) Send(from, to string, amount float64) error {
	return s.repo.Send(from, to, amount)
}

// GetLastTransactions возвращает список последних N транзакций.
//
// Параметры:
//   - count: Количество транзакций.
//
// Возвращает:
//   - Список транзакций.
//   - Ошибку, если произошла ошибка при выполнении запроса.
//
// Пример использования:
//
//	transactions, err := svc.GetLastTransactions(5)
func (s *Service) GetLastTransactions(count int) ([]models.Transaction, error) {
	return s.repo.GetLastTransactions(count)
}
