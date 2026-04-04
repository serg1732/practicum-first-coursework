package sql

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/serg1732/practicum-first-coursework/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupOrdersRepositoryTest(t *testing.T) (*OrdersRepoImpl, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	assert.NoError(t, err)

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	assert.NoError(t, err)

	repo := BuildRepository[OrdersRepoImpl](gdb)

	cleanup := func() {
		_ = sqlDB.Close()
	}
	return &repo, mock, cleanup
}

func setupTestLogger() *slog.Logger {
	return slog.Default()
}

func requiredNotError(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSuccessOrdersRepoAddNewOrderCreated(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()
	var accountId int64 = 10
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE "orders"."account_id" = $1 AND "orders"."order_id" = $2 ORDER BY "orders"."account_id" LIMIT $3`,
	)).
		WithArgs(accountId, "order-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "order_id"}))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO "orders" ("account_id","order_id") VALUES ($1,$2)`,
	)).
		WithArgs(accountId, "order-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	created, err := repo.AddNewOrder(ctx, logger, accountId, "order-1")
	assert.NoError(t, err)
	assert.True(t, created)
	requiredNotError(t, mock)
}

func TestOrdersRepoAddNewOrder(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()
	var accountId int64 = 10

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE "orders"."account_id" = $1 AND "orders"."order_id" = $2 ORDER BY "orders"."account_id" LIMIT $3`,
	)).
		WithArgs(accountId, "order-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "order_id"}).AddRow(accountId, "order-1"))

	created, err := repo.AddNewOrder(ctx, logger, accountId, "order-1")
	assert.NoError(t, err)
	assert.False(t, created)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoAddNewOrder(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()
	var accountId int64 = 10

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE "orders"."account_id" = $1 AND "orders"."order_id" = $2 ORDER BY "orders"."account_id" LIMIT $3`,
	)).
		WithArgs(accountId, "order-1", 1).
		WillReturnError(errors.New("select failed"))

	created, err := repo.AddNewOrder(ctx, logger, accountId, "order-1")
	assert.Error(t, err)
	assert.NotNil(t, err)
	assert.False(t, created)
	requiredNotError(t, mock)
}

func TestSuccessOrdersRepoUpdateOrder(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "orders" SET "status"=$1 WHERE order_id = $2`,
	)).
		WithArgs("PROCESSED", "order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.UpdateOrderStatus(ctx, logger, "order-1", "PROCESSED")
	assert.NoError(t, err)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoUpdateOrderStatus(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "orders" SET "status"=$1 WHERE order_id = $2`,
	)).
		WithArgs("PROCESSED", "order-1").
		WillReturnError(errors.New("ошибка при обновлении данных"))
	mock.ExpectRollback()

	err := repo.UpdateOrderStatus(ctx, logger, "order-1", "PROCESSED")
	assert.NotNil(t, err)
	requiredNotError(t, mock)
}

func TestSuccessOrdersRepoGetAllOrders(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()
	now := time.Now()

	var accountId int64 = 42
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE account_id = $1 ORDER BY uploaded_at DESC`,
	)).
		WithArgs(accountId).
		WillReturnRows(
			sqlmock.NewRows([]string{"account_id", "order_id", "status", "accrual", "uploaded_at"}).
				AddRow(accountId, "order-2", model.ORDER_STATUS_PROCESSING, 10.5, now).
				AddRow(accountId, "order-1", model.ORDER_STATUS_NEW, 0.0, now.Add(-time.Minute)),
		)

	orders, err := repo.GetAllOrders(ctx, logger, accountId)
	assert.NoError(t, err)
	assert.Len(t, orders, 2)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoGetAllOrders(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()
	var accountId int64 = 42

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE account_id = $1 ORDER BY uploaded_at DESC`,
	)).
		WithArgs(accountId).
		WillReturnError(errors.New("ошибка при select"))

	orders, err := repo.GetAllOrders(ctx, logger, accountId)
	assert.NotNil(t, err)
	assert.Nil(t, orders)
	requiredNotError(t, mock)
}

func TestOrdersRepoGetNewOrProcessingOrders(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()
	now := time.Now()
	var accountId int64 = 42

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE status = $1 OR status = $2 ORDER BY uploaded_at DESC`,
	)).
		WithArgs(model.ORDER_STATUS_NEW, model.ORDER_STATUS_PROCESSING).
		WillReturnRows(
			sqlmock.NewRows([]string{"account_id", "order_id", "status", "accrual", "uploaded_at"}).
				AddRow(accountId, "order-2", model.ORDER_STATUS_PROCESSING, 10.5, now).
				AddRow(accountId, "order-1", model.ORDER_STATUS_NEW, 0.0, now.Add(-time.Minute)),
		)

	orders, err := repo.GetNewOrProcessingOrders(ctx, logger)
	assert.NoError(t, err)
	assert.Len(t, orders, 2)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoGetNewOrProcessingOrders(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE status = $1 OR status = $2 ORDER BY uploaded_at DESC`,
	)).
		WithArgs(model.ORDER_STATUS_NEW, model.ORDER_STATUS_PROCESSING).
		WillReturnError(errors.New("ошибка при select"))

	orders, err := repo.GetNewOrProcessingOrders(ctx, logger)
	assert.NotNil(t, err)
	assert.Nil(t, orders)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoUpdateOrderStatusSum(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`UPDATE "orders" SET "accrual"=$1,"status"=$2 WHERE order_id = $3 RETURNING "account_id"`,
	)).
		WithArgs(150.5, "PROCESSED", "order-1").
		WillReturnError(errors.New("не удалось обновить баланс"))

	mock.ExpectRollback()

	err := repo.UpdateOrderStatusSum(ctx, logger, "order-1", "PROCESSED", 150.5)
	assert.NotNil(t, err)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoBalanceSelectError(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()
	var accountId int64 = 42

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`UPDATE "orders" SET "accrual"=$1,"status"=$2 WHERE order_id = $3 RETURNING "account_id"`,
	)).
		WithArgs(150.5, "PROCESSED", "order-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"account_id"}).AddRow(accountId),
		)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "balance" WHERE account_id = $1 ORDER BY "balance"."account_id" LIMIT $2 FOR UPDATE`,
	)).
		WithArgs(accountId, 1).
		WillReturnError(errors.New("не удалось сохранить в баланс"))

	mock.ExpectRollback()

	err := repo.UpdateOrderStatusSum(ctx, logger, "order-1", "PROCESSED", 150.5)
	assert.NotNil(t, err)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoBalanceSaveError(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	logger := setupTestLogger()

	var accountId int64 = 42
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`UPDATE "orders" SET "accrual"=$1,"status"=$2 WHERE order_id = $3 RETURNING "account_id"`,
	)).
		WithArgs(150.5, "PROCESSED", "order-1").
		WillReturnRows(
			sqlmock.NewRows([]string{"account_id"}).AddRow(accountId),
		)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "balance" WHERE account_id = $1 ORDER BY "balance"."account_id" LIMIT $2 FOR UPDATE`,
	)).
		WithArgs(accountId, 1).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "balance", "withdraw"}).AddRow(accountId, 100.0, 0.0))

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "balance" SET "account_id"=$1,"balance"=$2,"withdraw"=$3 WHERE account_id = $4`,
	)).
		WithArgs(accountId, 250.5, 0.0, accountId).
		WillReturnError(errors.New("не удалось сохранить в баланс"))

	mock.ExpectRollback()

	err := repo.UpdateOrderStatusSum(ctx, logger, "order-1", "PROCESSED", 150.5)
	assert.NotNil(t, err)
	requiredNotError(t, mock)
}
func TestSuccessOrdersRepoFindOrderById(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	var accountId int64 = 42
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE order_id = $1 ORDER BY "orders"."order_id" LIMIT $2`,
	)).
		WithArgs("order-1", 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"account_id", "order_id", "status", "accrual", "uploaded_at"}).
				AddRow(accountId, "order-1", model.ORDER_STATUS_NEW, 0.0, now),
		)

	order, err := repo.FindOrderById(ctx, "order-1")
	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "order-1", order.OrderId)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoNotFound(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE order_id = $1 ORDER BY "orders"."order_id" LIMIT $2`,
	)).
		WithArgs("order-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "order_id", "status", "accrual", "uploaded_at"}))

	order, err := repo.FindOrderById(ctx, "order-1")
	assert.NoError(t, err)
	assert.Nil(t, order)
	requiredNotError(t, mock)
}

func TestErrorOrdersRepoFindOrderById(t *testing.T) {
	repo, mock, cleanup := setupOrdersRepositoryTest(t)
	defer cleanup()

	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "orders" WHERE order_id = $1 ORDER BY "orders"."order_id" LIMIT $2`,
	)).
		WithArgs("order-1", 1).
		WillReturnError(errors.New("Ошибка при запросе на получение данных"))

	_, err := repo.FindOrderById(ctx, "order-1")
	assert.NotNil(t, err)
	requiredNotError(t, mock)
}
