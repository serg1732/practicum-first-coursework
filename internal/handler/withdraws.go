package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/serg1732/practicum-first-coursework/internal/model"
)

type BalanceRepository interface {
	GetBalance(ctx context.Context, log *slog.Logger, accountId int64) (*model.BalanceDB, error)
	GetForUpdate(ctx context.Context, log *slog.Logger, accountId int64, sum float64) (bool, error)
}

type WithdrawsRepository interface {
	AddWithdraw(ctx context.Context, log *slog.Logger, accountId int64, withdraw *model.WithdrawRequest) error
	GetWithdrawals(ctx context.Context, log *slog.Logger, accountId int64) ([]*model.Withdrawals, error)
}

func BuildWithdrawHandler(balanceRepo BalanceRepository, repo WithdrawsRepository, orders OrdersRepository) WithdrawsHandlerImpl {
	return WithdrawsHandlerImpl{
		balanceRepo:   balanceRepo,
		withdrawsRepo: repo,
		ordersRepo:    orders,
	}
}

type WithdrawsHandlerImpl struct {
	balanceRepo   BalanceRepository
	withdrawsRepo WithdrawsRepository
	ordersRepo    OrdersRepository
}

// GetAllWithdraw handler получения всех списаний
func (wh WithdrawsHandlerImpl) GetAllWithdraw(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var accountId = r.Context().Value("account_id").(int64)
		logsWithdraw, err := wh.withdrawsRepo.GetWithdrawals(r.Context(), log, accountId)
		if err != nil {
			log.Error("Ошибка при получении списаний", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		if err = enc.Encode(logsWithdraw); err != nil {
			log.Error("Ошибка при конвертации в JSON данных для отправки", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	}
}

// BalanceRequest handler получения баланса пользователя
func (wh *WithdrawsHandlerImpl) BalanceRequest(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var accountId = r.Context().Value("account_id").(int64)
		w.Header().Set("Content-Type", "application/json")
		balance, err := wh.balanceRepo.GetBalance(r.Context(), log, accountId)
		if err != nil {
			log.Error("ошибка получения баланса", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		if err = enc.Encode(balance); err != nil {
			log.Error("Ошибка при конвертации в JSON данных для отправки", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// WithdrawRequest handler запроса на списание баллов
func (wh *WithdrawsHandlerImpl) WithdrawRequest(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var accountId = r.Context().Value("account_id").(int64)
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		var req model.WithdrawRequest
		if err := decoder.Decode(&req); err != nil {
			log.Error("Ошибка при получении данных из тела запроса", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		updated, err := wh.balanceRepo.GetForUpdate(r.Context(), log, accountId, req.Sum)
		if err != nil {
			log.Error("Не удалось обновить баланс после списания", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !updated {
			log.Info("на счету недостаточно средств", "account_id", accountId)
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}

		if err := wh.withdrawsRepo.AddWithdraw(r.Context(), log, accountId, &req); err != nil {
			log.Error("ошибка при получении баланса", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
