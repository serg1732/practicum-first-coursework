package sql

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newBalanceTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	assert.NoError(t, err)
	return gdb, mock
}

func initBalanceTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestBalanceRepoSuccessGetBalance(t *testing.T) {
	db, mock := newBalanceTestDB(t)
	repo := BuildRepository[BalanceRepoImpl](db)
	logger := initBalanceTestLogger()

	var accountID int64 = 10

	rows := sqlmock.NewRows([]string{"account_id", "balance", "withdraw"}).
		AddRow(accountID, 100.5, 20.0)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "balance" WHERE account_id = $1 ORDER BY "balance"."account_id" LIMIT $2`,
	)).WithArgs(accountID, 1).WillReturnRows(rows)

	balance, err := repo.GetBalance(context.Background(), logger, accountID)
	assert.NoError(t, err)
	assert.NotNil(t, balance)
	assert.Equal(t, accountID, balance.AccountID)
	assert.Equal(t, 100.5, balance.Balance)
	assert.Equal(t, 20.0, balance.Withdraw)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceRepoErrorGetBalance(t *testing.T) {
	db, mock := newBalanceTestDB(t)
	repo := BuildRepository[BalanceRepoImpl](db)
	logger := initBalanceTestLogger()

	var accountID int64 = 10

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "balance" WHERE account_id = $1 ORDER BY "balance"."account_id" LIMIT $2`,
	)).WithArgs(accountID, 1).WillReturnError(errors.New("ошибка при поиске"))

	balance, err := repo.GetBalance(context.Background(), logger, accountID)
	assert.NotNil(t, err)
	assert.Nil(t, balance)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceRepoErrorGetForUpdateSelect(t *testing.T) {
	db, mock := newBalanceTestDB(t)
	repo := BuildRepository[BalanceRepoImpl](db)
	logger := initBalanceTestLogger()

	var accountID int64 = 10
	sum := 50.0

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "balance" WHERE account_id = $1 ORDER BY "balance"."account_id" LIMIT $2 FOR UPDATE`,
	)).WithArgs(accountID, 1).WillReturnError(errors.New("select for update error"))
	mock.ExpectRollback()

	updated, err := repo.GetForUpdate(context.Background(), logger, accountID, sum)
	assert.NotNil(t, err)
	assert.False(t, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceRepoSuccessGetForUpdateNotUpdate(t *testing.T) {
	db, mock := newBalanceTestDB(t)
	repo := BuildRepository[BalanceRepoImpl](db)
	logger := initBalanceTestLogger()

	var accountID int64 = 10
	sum := 150.0

	rows := sqlmock.NewRows([]string{"account_id", "balance", "withdraw"}).
		AddRow(accountID, 100.0, 20.0)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "balance" WHERE account_id = $1 ORDER BY "balance"."account_id" LIMIT $2 FOR UPDATE`,
	)).WithArgs(accountID, 1).WillReturnRows(rows)
	mock.ExpectCommit()

	updated, err := repo.GetForUpdate(context.Background(), logger, accountID, sum)
	assert.NoError(t, err)
	assert.False(t, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceRepoSuccessGetForUpdate(t *testing.T) {
	db, mock := newBalanceTestDB(t)
	repo := BuildRepository[BalanceRepoImpl](db)
	logger := initBalanceTestLogger()

	var accountID int64 = 10
	sum := 30.0

	rows := sqlmock.NewRows([]string{"account_id", "balance", "withdraw"}).
		AddRow(accountID, 100.0, 20.0)

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "balance" WHERE account_id = $1 ORDER BY "balance"."account_id" LIMIT $2 FOR UPDATE`,
	)).WithArgs(accountID, 1).WillReturnRows(rows)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "balance" SET "account_id"=$1,"balance"=$2,"withdraw"=$3 WHERE account_id = $4`,
	)).WithArgs(accountID, 70.0, 50.0, accountID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	updated, err := repo.GetForUpdate(context.Background(), logger, accountID, sum)
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceRepoErrorGetForUpdateSave(t *testing.T) {
	db, mock := newBalanceTestDB(t)
	repo := BuildRepository[BalanceRepoImpl](db)
	logger := initBalanceTestLogger()

	var accountID int64 = 10
	sum := 30.0

	rows := sqlmock.NewRows([]string{"account_id", "balance", "withdraw"}).
		AddRow(accountID, 100.0, 20.0)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "balance" WHERE account_id = $1 ORDER BY "balance"."account_id" LIMIT $2 FOR UPDATE`,
	)).WithArgs(accountID, 1).WillReturnRows(rows)
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "balance" SET "account_id"=$1,"balance"=$2,"withdraw"=$3 WHERE account_id = $4`,
	)).WithArgs(accountID, 70.0, 50.0, accountID).WillReturnError(errors.New("update error"))
	mock.ExpectRollback()

	updated, err := repo.GetForUpdate(context.Background(), logger, accountID, sum)
	assert.NotNil(t, err)
	assert.False(t, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}
