package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/serg1732/practicum-first-coursework/internal/model"
)

type OrdersRepository interface {
	AddNewOrder(ctx context.Context, log *slog.Logger, accountId int64, orderId string) (bool, error)
	FindOrderById(ctx context.Context, orderId string) (*model.Order, error)
	GetAllOrders(ctx context.Context, log *slog.Logger, accountId int64) ([]model.Order, error)
	UpdateOrderStatus(ctx context.Context, log *slog.Logger, orderId string, st string) error
	UpdateOrderStatusSum(ctx context.Context, log *slog.Logger, orderId string, st string, accural float64) error
}

func BuildOrdersHandler(repo OrdersRepository) OrdersHandlerImpl {
	return OrdersHandlerImpl{
		ordersRepo: repo,
	}
}

type OrdersHandlerImpl struct {
	ordersRepo OrdersRepository
}

func (o *OrdersHandlerImpl) GetAllOrders(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var accountId = r.Context().Value("account_id").(int64)
		orders, err := o.ordersRepo.GetAllOrders(r.Context(), log, accountId)
		if err != nil {
			log.Error("ошибка при получении заказов")
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		if err = enc.Encode(orders); err != nil {
			log.Error("Ошибка при конвертации в JSON данных для отправки", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (o *OrdersHandlerImpl) AddNewOrder(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var accountId = r.Context().Value("account_id").(int64)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var orderId = string(body)
		if errLuna := goluhn.Validate(orderId); errLuna != nil {
			log.Error("неверный формат номера заказа")
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		isAdded, err := o.ordersRepo.AddNewOrder(r.Context(), log, accountId, orderId)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.Code == "23505" {
					http.Error(w, "номер заказа уже был загружен другим пользователем", http.StatusConflict)
					return
				}
			}

			log.Error("ошибка при добавлении нового заказа", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !isAdded {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}
