package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/jackc/pgx/v5/pgconn"
	handlerMocks "github.com/serg1732/practicum-first-coursework/internal/handler/mocks"
	"github.com/serg1732/practicum-first-coursework/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ordersTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func withAccountID(req *http.Request, accountID int64) *http.Request {
	ctx := context.WithValue(req.Context(), accountIDKey, accountID)
	return req.WithContext(ctx)
}

func TestOrdersHandlerAddNewOrders(t *testing.T) {
	t.Run("OrderId не в формате Луна", func(t *testing.T) {
		repo := handlerMocks.NewOrdersRepository(t)
		h := BuildOrdersHandler(repo)

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("12345"))
		req = withAccountID(req, 1)
		rec := httptest.NewRecorder()

		h.AddNewOrder(ordersTestLogger()).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("Повторяющийся order_id для другого аккаунта", func(t *testing.T) {
		repo := handlerMocks.NewOrdersRepository(t)
		h := BuildOrdersHandler(repo)

		var accountId int64 = 10
		repo.
			On("AddNewOrder", mock.Anything, mock.Anything, accountId, "79927398713").
			Return(false, &pgconn.PgError{Code: "23505"}).
			Once()

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
		req = withAccountID(req, accountId)
		rec := httptest.NewRecorder()

		h.AddNewOrder(ordersTestLogger()).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "номер заказа уже был загружен другим пользователем\n", rec.Body.String())
	})

	t.Run("Ошибка в БД при добавлении", func(t *testing.T) {
		repo := handlerMocks.NewOrdersRepository(t)
		h := BuildOrdersHandler(repo)
		var accountId int64 = 10
		repo.
			On("AddNewOrder", mock.Anything, mock.Anything, accountId, "79927398713").
			Return(false, errors.New("db error")).
			Once()

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
		req = withAccountID(req, accountId)
		rec := httptest.NewRecorder()

		h.AddNewOrder(ordersTestLogger()).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Загружен order_id пользователем повторно", func(t *testing.T) {
		repo := handlerMocks.NewOrdersRepository(t)
		h := BuildOrdersHandler(repo)
		var accountId int64 = 10
		repo.
			On("AddNewOrder", mock.Anything, mock.Anything, accountId, "79927398713").
			Return(false, nil).
			Once()

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
		req = withAccountID(req, accountId)
		rec := httptest.NewRecorder()

		h.AddNewOrder(ordersTestLogger()).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Успешное добавление нового заказа", func(t *testing.T) {
		repo := handlerMocks.NewOrdersRepository(t)
		h := BuildOrdersHandler(repo)
		var accountId int64 = 10
		repo.
			On("AddNewOrder", mock.Anything, mock.Anything, accountId, "79927398713").
			Return(true, nil).
			Once()

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
		req = withAccountID(req, accountId)
		rec := httptest.NewRecorder()

		h.AddNewOrder(ordersTestLogger()).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusAccepted, rec.Code)
	})
}

func TestOrdersHandlerGetAllOrders(t *testing.T) {
	t.Run("Успешное получение списка заказов", func(t *testing.T) {
		repo := handlerMocks.NewOrdersRepository(t)
		h := BuildOrdersHandler(repo)
		accrualProcessed := 100.5
		accrualNew := 0.0
		expected := []model.Order{
			{OrderId: "79927398713", Status: "PROCESSED", Accrual: &accrualProcessed},
			{OrderId: "12345678903", Status: "NEW", Accrual: &accrualNew},
		}
		var accountId int64 = 50
		repo.
			On("GetAllOrders", mock.Anything, mock.Anything, accountId).
			Return(expected, nil).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		req = withAccountID(req, accountId)
		rec := httptest.NewRecorder()

		h.GetAllOrders(ordersTestLogger()).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var orders []model.Order
		err := json.Unmarshal(rec.Body.Bytes(), &orders)
		assert.NoError(t, err)
		assert.Equal(t, expected, orders)
	})

	t.Run("Ошибка в БД", func(t *testing.T) {
		repo := handlerMocks.NewOrdersRepository(t)
		h := BuildOrdersHandler(repo)
		var accountId int64 = 50
		repo.
			On("GetAllOrders", mock.Anything, mock.Anything, accountId).
			Return(nil, errors.New("db error")).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		req = withAccountID(req, accountId)
		rec := httptest.NewRecorder()

		h.GetAllOrders(ordersTestLogger()).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
