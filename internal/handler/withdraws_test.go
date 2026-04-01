package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/serg1732/practicum-first-coursework/internal/handler/mocks"
	"github.com/serg1732/practicum-first-coursework/internal/model"
)

func testWithdrawsLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func requestWithAccountID(method, target string, body io.Reader, accountID int64) *http.Request {
	req := httptest.NewRequest(method, target, body)
	ctx := context.WithValue(req.Context(), "account_id", accountID)
	return req.WithContext(ctx)
}

func TestSuccessGetAllWithdraw(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)

	expected := []*model.Withdrawals{}
	var accountID int64 = 42
	withdrawsRepo.
		On("GetWithdrawals", mock.Anything, mock.Anything, accountID).
		Return(expected, nil).
		Once()

	req := requestWithAccountID(http.MethodGet, "/api/user/withdrawals", nil, accountID)
	rr := httptest.NewRecorder()

	h.GetAllWithdraw(testWithdrawsLogger())(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestErrorGetAllWithdrawRepositoryError(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)
	var accountID int64 = 42
	withdrawsRepo.
		On("GetWithdrawals", mock.Anything, mock.Anything, accountID).
		Return(nil, errors.New("db error")).
		Once()

	req := requestWithAccountID(http.MethodGet, "/api/user/withdrawals", nil, accountID)
	rr := httptest.NewRecorder()

	h.GetAllWithdraw(testWithdrawsLogger())(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestSuccessBalanceRequest(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)
	var accountID int64 = 101
	balanceRepo.
		On("GetBalance", mock.Anything, mock.Anything, accountID).
		Return(&model.BalanceDB{}, nil).
		Once()

	req := requestWithAccountID(http.MethodGet, "/api/user/balance", nil, accountID)
	rr := httptest.NewRecorder()

	h.BalanceRequest(testWithdrawsLogger())(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestErrorBalanceRequestRepositoryError(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)
	var accountID int64 = 102
	balanceRepo.
		On("GetBalance", mock.Anything, mock.Anything, accountID).
		Return(nil, errors.New("ошибка при запросе баланса")).
		Once()

	req := requestWithAccountID(http.MethodGet, "/api/user/balance", nil, accountID)
	rr := httptest.NewRecorder()

	h.BalanceRequest(testWithdrawsLogger())(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestErrorWithdrawRequestInvalidJSON(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)

	req := requestWithAccountID(
		http.MethodPost,
		"/api/user/balance/withdraw",
		bytes.NewBufferString(`{"order":"123","sum":`),
		77,
	)
	rr := httptest.NewRecorder()

	h.WithdrawRequest(testWithdrawsLogger())(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestErrorWithdrawRequestGetForUpdateError(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)
	var accountID int64 = 105
	balanceRepo.
		On("GetForUpdate", mock.Anything, mock.Anything, accountID, 10.5).
		Return(false, errors.New("update error")).
		Once()

	req := requestWithAccountID(
		http.MethodPost,
		"/api/user/balance/withdraw",
		bytes.NewBufferString(`{"order":"12345","sum":10.5}`),
		accountID,
	)
	rr := httptest.NewRecorder()

	h.WithdrawRequest(testWithdrawsLogger())(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestErrorWithdrawRequestNotEnoughMoney(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)
	var accountID int64 = 105
	balanceRepo.
		On("GetForUpdate", mock.Anything, mock.Anything, accountID, 999999.0).
		Return(false, nil).
		Once()

	req := requestWithAccountID(
		http.MethodPost,
		"/api/user/balance/withdraw",
		bytes.NewBufferString(`{"order":"12345","sum":999999}`),
		accountID,
	)
	rr := httptest.NewRecorder()

	h.WithdrawRequest(testWithdrawsLogger())(rr, req)

	assert.Equal(t, http.StatusPaymentRequired, rr.Code)
	withdrawsRepo.AssertNotCalled(t, "AddWithdraw", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestErrorWithdrawRequestAddWithdrawError(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)
	var accountID = rand.Int64()
	balanceRepo.
		On("GetForUpdate", mock.Anything, mock.Anything, accountID, 10.5).
		Return(true, nil).
		Once()

	withdrawsRepo.
		On(
			"AddWithdraw",
			mock.Anything,
			mock.Anything,
			accountID,
			mock.MatchedBy(func(req *model.WithdrawRequest) bool {
				return req != nil && req.Order == "12345" && req.Sum == 10.5
			}),
		).
		Return(errors.New("Ошибка добавление данных")).
		Once()

	req := requestWithAccountID(
		http.MethodPost,
		"/api/user/balance/withdraw",
		bytes.NewBufferString(`{"order":"12345","sum":10.5}`),
		accountID,
	)
	rr := httptest.NewRecorder()

	h.WithdrawRequest(testWithdrawsLogger())(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestSuccessWithdrawRequest(t *testing.T) {
	balanceRepo := mocks.NewBalanceRepository(t)
	withdrawsRepo := mocks.NewWithdrawsRepository(t)
	ordersRepo := mocks.NewOrdersRepository(t)

	h := BuildWithdrawHandler(balanceRepo, withdrawsRepo, ordersRepo)
	var accountID = rand.Int64()
	balanceRepo.
		On("GetForUpdate", mock.Anything, mock.Anything, accountID, 10.5).
		Return(true, nil).
		Once()

	withdrawsRepo.
		On(
			"AddWithdraw",
			mock.Anything,
			mock.Anything,
			accountID,
			mock.MatchedBy(func(req *model.WithdrawRequest) bool {
				return req != nil && req.Order == "12345" && req.Sum == 10.5
			}),
		).
		Return(nil).
		Once()

	req := requestWithAccountID(
		http.MethodPost,
		"/api/user/balance/withdraw",
		bytes.NewBufferString(`{"order":"12345","sum":10.5}`),
		accountID,
	)
	rr := httptest.NewRecorder()

	h.WithdrawRequest(testWithdrawsLogger())(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
