package sql

import (
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/serg1732/practicum-first-coursework/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func BuildRepository[T AccountRepoImpl | BalanceRepoImpl | OrdersRepoImpl | WithdrawsRepoImpl](db *gorm.DB) T {
	return T{db: db}
}

// BuildConnection создание подключения к БД
func BuildConnection(log *slog.Logger, config *config.GophermartConfig) (*gorm.DB, error) {
	if config.DSN == "" {
		log.Error("Конфиг ДБ пустой")
		return nil, errors.New("DSL required")
	}

	db, err := gorm.Open(postgres.Open(config.DSN), &gorm.Config{})
	if err != nil {
		log.Error("Ошибка при подключении", "error", err)
		return nil, err
	}

	conn, err := db.DB()
	if err != nil {
		log.Error("Ошибка при подключении", "error", err)
		return nil, err
	}

	if errPing := conn.Ping(); errPing != nil {
		log.Error("Нет подключения к БД!", "error", errPing)
		return nil, errPing
	}

	log.Info("Успешное подключение к БД")

	return db, nil
}

// MigrateDataBase миграция данных БД
func MigrateDataBase(log *slog.Logger, config *config.GophermartConfig) error {
	if config.DSN == "" {
		log.Error("Конфиг ДБ пустой")
		return errors.New("DSL required")
	}
	m, err := migrate.New("file://migrations", config.DSN)
	if err != nil {
		log.Error("ошибка миграции", "error", err)
		return err
	}
	if errUp := m.Up(); errUp != nil {
		log.Error("Не удалось <<апнуть>> БД", "error", errUp)
	}

	version, dirty, err := m.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			log.Error("ошибка миграции", "error", err)
			return err
		}
		log.Error("Ошибка при миграции", "error", err)
		return err
	}

	log.Info("Current version:", "version", version)
	log.Info("Dirty:", "dirty", dirty)
	return nil
}
