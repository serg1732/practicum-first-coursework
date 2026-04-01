package sql

import (
	"context"
	"log/slog"

	"github.com/serg1732/practicum-first-coursework/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BuildBalanceRepo создание репозитория BALANCE
func BuildBalanceRepo(db *gorm.DB) BalanceRepoImpl {
	return BalanceRepoImpl{
		db,
	}
}

// BalanceRepoImpl репозиторий по работе с балансом
type BalanceRepoImpl struct {
	db *gorm.DB
}

// GetBalance получение баланса пользователя
func (b *BalanceRepoImpl) GetBalance(ctx context.Context, log *slog.Logger, accountId int64) (*model.BalanceDB, error) {
	var balance model.BalanceDB
	if err := b.db.WithContext(ctx).Table("balance").Where("account_id = ?", accountId).First(&balance).Error; err != nil {
		log.Error("Ошибка при получении баланса", "error", err)
		return nil, err
	}
	return &balance, nil
}

// GetForUpdate изменение баланса
func (b *BalanceRepoImpl) GetForUpdate(ctx context.Context, log *slog.Logger, accountId int64, sum float64) (bool, error) {
	var balance model.BalanceDB
	var updated = false
	err := b.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("balance").WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", accountId).
			First(&balance).Error; err != nil {
			log.Error("Ошибка при получении баланса", "error", err)
			return err
		}
		if balance.Balance < sum {
			log.Info("Недостаточно средств для списания")
			return nil
		}
		balance.Balance -= sum
		balance.Withdraw += sum
		if err := tx.WithContext(ctx).Table("balance").Where("account_id = ?", accountId).Save(&balance).Error; err != nil {
			log.Error("Ошибка при записи баланса", "error", err)
			return err
		}
		updated = true
		return nil
	})
	log.Debug("Успешное обновление баланса после списания", "accountId", accountId)
	return updated, err
}
