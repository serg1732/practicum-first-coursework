package sql

import (
	"context"
	"errors"
	"log/slog"

	"github.com/serg1732/practicum-first-coursework/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrdersRepoImpl репозиторий заказов
type OrdersRepoImpl struct {
	db *gorm.DB
}

// AddNewOrder доблавение новного заказа
func (o *OrdersRepoImpl) AddNewOrder(ctx context.Context, log *slog.Logger, accountID int64, orderID string) (bool, error) {
	var order *model.OrderDB
	tx := o.db.WithContext(ctx).
		Table("orders").
		Where(&model.OrderDB{
			AccountID: accountID,
			OrderID:   orderID}).
		FirstOrCreate(&order)
	if tx.Error != nil {
		log.Error("Ошибка при добавлении заказа", "error", tx.Error)
		return false, tx.Error
	}

	if tx.RowsAffected == 1 {
		return true, nil
	}

	return false, nil
}

// UpdateOrderStatus Обновление статуса и запись в БД
func (o *OrdersRepoImpl) UpdateOrderStatus(ctx context.Context, log *slog.Logger, orderID string, st string) error {
	if err := o.db.WithContext(ctx).Table("orders").Where("order_id = ?", orderID).Update("status", st).Error; err != nil {
		log.Error("ошибка при обновление статуса", "error", err)
		return err
	}
	return nil
}

// UpdateOrderStatusSum Обновление статуса и баланса
func (o *OrdersRepoImpl) UpdateOrderStatusSum(ctx context.Context, log *slog.Logger, orderID string, st string, accrual float64) error {
	var err = o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		type OrderReturn struct {
			AccountID int64  `gorm:"account_id" json:"account_id"`
			OrderID   string `gorm:"order_id" json:"order_id"`
		}
		var orderReturn OrderReturn
		if err := tx.WithContext(ctx).
			Table("orders").
			Model(&orderReturn).
			Clauses(clause.Returning{
				Columns: []clause.Column{
					{Name: "account_id"},
				},
			}).
			Where("order_id = ?", orderID).
			Updates(map[string]interface{}{
				"status":  st,
				"accrual": accrual,
			}).Error; err != nil {
			log.Error("ошибка при обновление статуса и начисления", "error", err)
			return err
		}

		var balance model.BalanceDB

		if err := tx.Table("balance").WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", orderReturn.AccountID).
			First(&balance).Error; err != nil {
			log.Error("Ошибка при получении баланса", "error", err)
			return err
		}

		balance.Balance += accrual

		if err := tx.WithContext(ctx).Table("balance").Where("account_id = ?", orderReturn.AccountID).Save(&balance).Error; err != nil {
			log.Error("Ошибка при записи баланса", "error", err)
			return err
		}
		return nil
	})
	return err
}

// GetAllOrders получение всех заказов пользователя
func (o *OrdersRepoImpl) GetAllOrders(ctx context.Context, log *slog.Logger, accountID int64) ([]model.Order, error) {
	var orders []model.Order
	if err := o.db.WithContext(ctx).
		Table("orders").
		Order("uploaded_at DESC").
		Where("account_id = ?", accountID).
		Find(&orders).Error; err != nil {
		log.Error("Ошибка при получении заказов", "error", err)
		return nil, err
	}
	return orders, nil
}

// GetNewOrProcessingOrders Получение всех заказов NEW / PROCESSING для аккаунта
func (o *OrdersRepoImpl) GetNewOrProcessingOrders(ctx context.Context, log *slog.Logger) ([]model.Order, error) {
	var orders []model.Order
	if err := o.db.WithContext(ctx).
		Table("orders").
		Order("uploaded_at DESC").
		Where("status = ? OR status = ?", model.OrderStatusNew, model.OrderStatusProcessing).
		Find(&orders).Error; err != nil {
		log.Error("Ошибка при получении заказов", "error", err)
		return nil, err
	}
	return orders, nil
}

// FindOrderById поиск заказа
func (o *OrdersRepoImpl) FindOrderById(ctx context.Context, orderId string) (*model.Order, error) {
	var order *model.Order
	err := o.db.WithContext(ctx).
		Table("orders").
		Where("order_id = ?", orderId).
		First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return order, err
}
