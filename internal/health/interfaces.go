package health

import "database/sql"

// DatabaseInterface определяет интерфейс для проверки здоровья базы данных
type DatabaseInterface interface {
	Ping() error
	Close() error
	Query(query string, args ...interface{}) (*sql.Rows, error)
}
