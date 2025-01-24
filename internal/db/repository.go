// Package db предоставляет функционал для работы с базой данных PostgreSQL.
// Включает создание таблиц, генерацию кошельков, перевод средств между кошельками,
// получение баланса и списка транзакций.
package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"payment-system/internal/models"

	_ "github.com/lib/pq"
)

// PostgresRepository представляет репозиторий для работы с PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создает новый экземпляр PostgresRepository.
// Подключается к базе данных PostgreSQL, инициализирует таблицы и создает 10 кошельков
// с балансом 100.0, если таблица пуста.
//
// Пример использования:
//
//	repo := NewPostgresRepository()
func NewPostgresRepository() *PostgresRepository {
	db, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		"5432",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	))
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Инициализация таблиц
	if err := initTables(db); err != nil {
		log.Fatal("Failed to initialize tables:", err)
	}

	// Создание 10 кошельков с балансом 100.0
	if err := generateWallets(db, 10, 100.0); err != nil {
		log.Fatal("Failed to generate wallets:", err)
	}

	return &PostgresRepository{db: db}
}

// initTables создает таблицы wallets и transactions, если они не существуют.
//
// Параметры:
//   - db: Указатель на подключение к базе данных.
//
// Возвращает:
//   - Ошибку, если не удалось создать таблицы.
func initTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS wallets (
			address TEXT PRIMARY KEY,
			balance FLOAT
		);
		CREATE TABLE IF NOT EXISTS transactions (
			id SERIAL PRIMARY KEY,
			from_address TEXT,
			to_address TEXT,
			amount FLOAT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

// generateWallets создает указанное количество кошельков с заданным балансом.
//
// Параметры:
//   - db: Указатель на подключение к базе данных.
//   - count: Количество кошельков для создания.
//   - balance: Начальный баланс каждого кошелька.
//
// Возвращает:
//   - Ошибку, если не удалось создать кошельки.
func generateWallets(db *sql.DB, count int, balance float64) error {
	for i := 0; i < count; i++ {
		address := generateRandomAddress()
		_, err := db.Exec("INSERT INTO wallets (address, balance) VALUES ($1, $2) ON CONFLICT (address) DO NOTHING", address, balance)
		if err != nil {
			return err
		}
	}
	return nil
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
//	balance, err := repo.GetBalance("some_address")
func (r *PostgresRepository) GetBalance(address string) (float64, error) {
	var balance float64
	err := r.db.QueryRow("SELECT balance FROM wallets WHERE address = $1", address).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance, nil
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
//	err := repo.Send("from_address", "to_address", 10.5)
func (r *PostgresRepository) Send(from, to string, amount float64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверка баланса отправителя
	var fromBalance float64
	err = tx.QueryRow("SELECT balance FROM wallets WHERE address = $1", from).Scan(&fromBalance)
	if err != nil {
		return fmt.Errorf("failed to get sender balance: %w", err)
	}

	if fromBalance < amount {
		return fmt.Errorf("insufficient funds")
	}

	// Обновление баланса отправителя
	_, err = tx.Exec("UPDATE wallets SET balance = balance - $1 WHERE address = $2", amount, from)
	if err != nil {
		return fmt.Errorf("failed to update sender balance: %w", err)
	}

	// Обновление баланса получателя
	_, err = tx.Exec("UPDATE wallets SET balance = balance + $1 WHERE address = $2", amount, to)
	if err != nil {
		return fmt.Errorf("failed to update receiver balance: %w", err)
	}

	// Запись транзакции
	_, err = tx.Exec("INSERT INTO transactions (from_address, to_address, amount) VALUES ($1, $2, $3)", from, to, amount)
	if err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}

	return tx.Commit()
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
//	transactions, err := repo.GetLastTransactions(5)
func (r *PostgresRepository) GetLastTransactions(count int) ([]models.Transaction, error) {
	rows, err := r.db.Query("SELECT from_address, to_address, amount, timestamp FROM transactions ORDER BY timestamp DESC LIMIT $1", count)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		err := rows.Scan(&t.From, &t.To, &t.Amount, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return transactions, nil
}

// GenerateAddress генерирует случайный адрес длиной 64 символа (32 байта в hex).
//
// Возвращает:
//   - Сгенерированный адрес.
//   - Ошибку, если не удалось сгенерировать случайные байты.
//
// Пример использования:
//
//	address, err := GenerateAddress()
func GenerateAddress() (string, error) {
	buffer := make([]byte, 32)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

// generateRandomAddress генерирует случайный адрес кошелька.
// В случае ошибки завершает выполнение программы.
//
// Возвращает:
//   - Сгенерированный адрес.
func generateRandomAddress() string {
	address, err := GenerateAddress()
	if err != nil {
		log.Fatal("Failed to generate wallet address:", err)
	}
	return address
}

// Ping проверяет подключение к базе данных.
//
// Параметры:
//   - ctx: Контекст для выполнения запроса.
//
// Возвращает:
//   - Ошибку, если подключение не удалось.
//
// Пример использования:
//
//	err := repo.Ping(context.Background())
func (r *PostgresRepository) Ping(ctx context.Context) error {
	if err := r.db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
}
