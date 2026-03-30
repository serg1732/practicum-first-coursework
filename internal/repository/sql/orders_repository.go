package sql

import (
	"context"
	"errors"
	"log/slog"

	"github.com/serg1732/practicum-first-coursework/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func BuildOrdersRepo(db *gorm.DB) OrdersRepoImpl {
	return OrdersRepoImpl{
		db,
	}
}

type OrdersRepoImpl struct {
	db *gorm.DB
}

func (o *OrdersRepoImpl) AddNewOrder(ctx context.Context, log *slog.Logger, accountId int64, orderId string) (bool, error) {
	var order *model.OrderDB
	tx := o.db.WithContext(ctx).
		Table("orders").
		Where(&model.OrderDB{
			AccountId: accountId,
			OrderId:   orderId}).
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

func (o *OrdersRepoImpl) UpdateOrderStatus(ctx context.Context, log *slog.Logger, orderId string, st string) error {
	if err := o.db.WithContext(ctx).Table("orders").Where("order_id = ?", orderId).Update("status", st).Error; err != nil {
		log.Error("ошибка при обновление статуса", "error", err)
		return err
	}
	return nil
}

func (o *OrdersRepoImpl) UpdateOrderStatusSum(ctx context.Context, log *slog.Logger, orderId string, st string, accrual float64) error {
	var err = o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		type OrderReturn struct {
			AccountId int64  `gorm:"account_id" json:"account_id"`
			OrderId   string `gorm:"order_id" json:"order_id"`
		}
		var orderReturn OrderReturn
		if err := o.db.WithContext(ctx).
			Table("orders").
			Model(&orderReturn).
			Clauses(clause.Returning{
				Columns: []clause.Column{
					{Name: "account_id"},
				},
			}).
			Where("order_id = ?", orderId).
			Updates(map[string]interface{}{
				"status":  st,
				"accrual": accrual,
			}).Error; err != nil {
			log.Error("ошибка при обновление статуса и начисления", "error", err)
			return err
		}

		var balance model.BalanceDB

		if err := o.db.Table("balance").WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", orderReturn.AccountId).
			First(&balance).Error; err != nil {
			log.Error("Ошибка при получении баланса", "error", err)
			return err
		}

		balance.Balance += accrual

		if err := o.db.WithContext(ctx).Table("balance").Where("account_id = ?", orderReturn.AccountId).Save(&balance).Error; err != nil {
			log.Error("Ошибка при записи баланса", "error", err)
			return err
		}
		return nil
	})
	return err
}

func (o *OrdersRepoImpl) GetAllOrders(ctx context.Context, log *slog.Logger, accountId int64) ([]model.Order, error) {
	var orders []model.Order
	if err := o.db.WithContext(ctx).
		Table("orders").
		Order("uploaded_at DESC").
		Where("account_id = ?", accountId).
		Find(&orders).Error; err != nil {
		log.Error("Ошибка при получении заказов", "error", err)
		return nil, err
	}
	return orders, nil
}

func (o *OrdersRepoImpl) GetNewOrProcessingOrders(ctx context.Context, log *slog.Logger) ([]model.Order, error) {
	var orders []model.Order
	if err := o.db.WithContext(ctx).
		Table("orders").
		Order("uploaded_at DESC").
		Where("status = ? OR status = ?", model.ORDER_STATUS_NEW, model.ORDER_STATUS_PROCESSING).
		Find(&orders).Error; err != nil {
		log.Error("Ошибка при получении заказов", "error", err)
		return nil, err
	}
	return orders, nil
}

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
