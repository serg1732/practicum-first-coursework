package main

import (
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/handler"
)

func TestBuildRouteSuccess(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.GophermartConfig{}

	var accountRepo handler.AccountRepository
	var orderRepo handler.OrdersRepository
	var balanceRepo handler.BalanceRepository
	var withdrawRepo handler.WithdrawsRepository

	r := buildRoute(log, cfg, accountRepo, orderRepo, balanceRepo, withdrawRepo)

	routesCheck := make(map[string]struct{})
	err := chi.Walk(r, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method == http.MethodGet || method == http.MethodPost {
			routesCheck[method+" "+route] = struct{}{}
		}
		return nil
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		route string
	}{
		{"Регистрация", "POST /api/user/register"},
		{"Авторизация", "POST /api/user/login"},
		{"Список заказов", "GET /api/user/orders"},
		{"Добавление заказа", "POST /api/user/orders"},
		{"Получение истории списаний", "GET /api/user/withdrawals"},
		{"Получение баланса", "GET /api/user/balance/"},
		{"Запрос на списание баллов", "POST /api/user/balance/withdraw"},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			t.Parallel()
			if _, ok := routesCheck[td.route]; !ok {
				t.Fatalf("такого роутер нет %s", td.route)
			}
		})
	}
}
