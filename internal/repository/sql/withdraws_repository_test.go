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

func initWithdrawTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	cleanup := func() {
		_ = db.Close()
	}

	return gdb, mock, cleanup
}

func newTestLogger() *slog.Logger {
	return slog.Default()
}

func TestWithdrawsRepoSuccessAddWithdraw(t *testing.T) {
	gdb, mock, cleanup := initWithdrawTestDB(t)
	defer cleanup()

	repo := BuildRepository[WithdrawsRepoImpl](gdb)
	logger := newTestLogger()
	ctx := context.Background()

	req := &model.WithdrawRequest{
		Order: "8450781367",
		Sum:   150.5,
	}

	var expectedAccID int64 = 10
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "withdraws" ("account_id","order_id","sum","processed_at") VALUES ($1,$2,$3,$4)`)).
		WithArgs(expectedAccID, req.Order, req.Sum, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.AddWithdraw(ctx, logger, expectedAccID, req)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithdrawsRepoErrorAddWithdraw(t *testing.T) {
	gdb, mock, cleanup := initWithdrawTestDB(t)
	defer cleanup()

	repo := BuildRepository[WithdrawsRepoImpl](gdb)
	logger := newTestLogger()
	ctx := context.Background()

	req := &model.WithdrawRequest{
		Order: "8450781367",
		Sum:   150.5,
	}

	expectedErr := errors.New("ошибка добавления")
	var expectedAccID int64 = 10
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "withdraws" ("account_id","order_id","sum","processed_at") VALUES ($1,$2,$3,$4)`)).
		WithArgs(expectedAccID, req.Order, req.Sum, sqlmock.AnyArg()).
		WillReturnError(expectedErr)
	mock.ExpectRollback()

	err := repo.AddWithdraw(ctx, logger, expectedAccID, req)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithdrawsRepoSuccessGetWithdrawals(t *testing.T) {
	gdb, mock, cleanup := initWithdrawTestDB(t)
	defer cleanup()

	repo := BuildRepository[WithdrawsRepoImpl](gdb)
	logger := newTestLogger()
	ctx := context.Background()

	now := time.Now()
	var expectedAccID int64 = 10
	var orderOne = "8450781367"
	var orderTwo = "2875608057"
	rows := sqlmock.NewRows([]string{
		"account_id",
		"order_id",
		"sum",
		"processed_at",
	}).AddRow(expectedAccID, orderOne, 100.25, now).
		AddRow(expectedAccID, orderTwo, 50.75, now.Add(-time.Hour))

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "withdraws" WHERE account_id = $1 ORDER BY processed_at DESC`,
	)).
		WithArgs(expectedAccID).
		WillReturnRows(rows)

	withdraws, err := repo.GetWithdrawals(ctx, logger, expectedAccID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(withdraws))
	assert.Equal(t, orderOne, withdraws[0].OrderId)
	assert.Equal(t, orderTwo, withdraws[1].OrderId)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithdrawsRepoErrorGetWithdrawals(t *testing.T) {
	gdb, mock, cleanup := initWithdrawTestDB(t)
	defer cleanup()

	repo := BuildRepository[WithdrawsRepoImpl](gdb)
	logger := newTestLogger()
	ctx := context.Background()

	expectedErr := errors.New("Ошибка при получении списаний")
	var expectedAccID int64 = 10
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "withdraws" WHERE account_id = $1 ORDER BY processed_at DESC`,
	)).
		WithArgs(expectedAccID).
		WillReturnError(expectedErr)

	withdrawals, err := repo.GetWithdrawals(ctx, logger, expectedAccID)
	assert.NotNil(t, err)
	assert.Nil(t, withdrawals)
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
