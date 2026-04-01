package sql

import (
	"context"
	"log/slog"
	"time"

	"github.com/serg1732/practicum-first-coursework/internal/model"
	"gorm.io/gorm"
)

// BuildWithdrawsRepo создание репозитория списаний баллов
func BuildWithdrawsRepo(db *gorm.DB) WithdrawsRepoImpl {
	return WithdrawsRepoImpl{
		db,
	}
}

// WithdrawsRepoImpl репозиторий списаний баллов
type WithdrawsRepoImpl struct {
	db *gorm.DB
}

// AddWithdraw добавить списание
func (w *WithdrawsRepoImpl) AddWithdraw(ctx context.Context, log *slog.Logger, accountId int64, withdraw *model.WithdrawRequest) error {
	withdrawDB := &model.Withdrawals{
		AccountId:   accountId,
		OrderId:     withdraw.Order,
		Sum:         withdraw.Sum,
		ProcessedAt: time.Now(),
	}
	if err := w.db.WithContext(ctx).
		Table("withdraws").
		Create(&withdrawDB).Error; err != nil {
		log.Error("Ошибка при добавлении вывода средств")
		return err
	}
	return nil
}

// GetWithdrawals поиск всех списаний для пользователя
func (b *WithdrawsRepoImpl) GetWithdrawals(ctx context.Context, log *slog.Logger, accountId int64) ([]*model.Withdrawals, error) {
	var withdrawals []*model.Withdrawals

	if err := b.db.WithContext(ctx).Table("withdraws").
		Order("processed_at DESC").
		Where("account_id = ?", accountId).
		Find(&withdrawals).Error; err != nil {
		log.Error("Ошибка при получении баланса", "error", err)
		return nil, err
	}
	log.Debug("Успешное получение баланса", "accountId", accountId)
	return withdrawals, nil
}
