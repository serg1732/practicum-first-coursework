package sql

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/serg1732/practicum-first-coursework/internal/model"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupAccountsRepoTest(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	assert.NoError(t, err)

	cleanup := func() {
		_ = sqlDB.Close()
	}

	return gdb, mock, cleanup
}

func RandStringBytes(t *testing.T, n int) string {
	t.Helper()
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func setupAccountsLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestBuildAccountsRepo(t *testing.T) {
	gdb, _, cleanup := setupAccountsRepoTest(t)
	defer cleanup()

	repo := BuildRepository[AccountRepoImpl](gdb)
	assert.NotNil(t, repo)
}

func TestAccountRepoSuccessCreateAccount(t *testing.T) {
	gdb, mock, cleanup := setupAccountsRepoTest(t)
	defer cleanup()

	repo := BuildRepository[AccountRepoImpl](gdb)
	ctx := context.Background()
	logger := setupAccountsLogger()

	auth := &model.Accounts{
		Login:    "user",
		Password: "password",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "accounts"`)).
		WithArgs(auth.Login, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "balance"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(ctx, logger, auth)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), auth.ID, "ожидаем ID = 1")
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(auth.Password), []byte("password")))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepoErrorHashCreateAccount(t *testing.T) {
	gdb, mock, cleanup := setupAccountsRepoTest(t)
	defer cleanup()

	repo := BuildRepository[AccountRepoImpl](gdb)
	ctx := context.Background()
	logger := setupAccountsLogger()

	auth := &model.Accounts{
		Login:    "user",
		Password: RandStringBytes(t, 80),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "accounts"`)).
		WithArgs(auth.Login, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err := repo.Create(ctx, logger, auth)
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("bcrypt: password length exceeds 72 bytes"), err)
	assert.NotNil(t, mock.ExpectationsWereMet())
}

func TestAccountRepoErrorCreateAccount(t *testing.T) {
	gdb, mock, cleanup := setupAccountsRepoTest(t)
	defer cleanup()

	repo := BuildRepository[AccountRepoImpl](gdb)
	ctx := context.Background()
	logger := setupAccountsLogger()

	auth := &model.Accounts{
		Login:    "user",
		Password: "password",
	}

	expectedErr := errors.New("ошибка создания аккаунта")
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "accounts"`)).
		WithArgs(auth.Login, sqlmock.AnyArg()).
		WillReturnError(expectedErr)
	mock.ExpectRollback()

	err := repo.Create(ctx, logger, auth)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErr, err, "ожидали другую ошибку")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepoErrorCreateBalance(t *testing.T) {
	gdb, mock, cleanup := setupAccountsRepoTest(t)
	defer cleanup()

	repo := BuildRepository[AccountRepoImpl](gdb)
	ctx := context.Background()
	logger := setupAccountsLogger()

	auth := &model.Accounts{
		Login:    "user",
		Password: "password",
	}

	expectedErr := errors.New("ошибка создания баланса")
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "accounts"`)).
		WithArgs(auth.Login, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "balance"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(expectedErr)
	mock.ExpectRollback()
	err := repo.Create(ctx, logger, auth)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepoSuccessLogin(t *testing.T) {
	gdb, mock, cleanup := setupAccountsRepoTest(t)
	defer cleanup()

	repo := BuildRepository[AccountRepoImpl](gdb)
	ctx := context.Background()
	logger := setupAccountsLogger()

	hashed, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.Nil(t, err, "не ожидали получить ошибку при генерации hash пароля")

	auth := &model.Accounts{
		Login:    "user",
		Password: "password",
	}
	var expectedAccountId int64 = 10
	rows := sqlmock.NewRows([]string{"id", "login", "password"}).
		AddRow(expectedAccountId, "user", string(hashed))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "accounts" WHERE login = $1 ORDER BY "accounts"."id" LIMIT $2`)).
		WithArgs("user", 1).
		WillReturnRows(rows)

	id, err := repo.Login(ctx, logger, auth)
	assert.Nil(t, err, "Ошибка при авторизации")
	assert.NotNil(t, id, "id аккаунта не может быть nil")
	assert.Equal(t, expectedAccountId, *id, "другой идентификатор аккаунта")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepoErrorLoginAccountNotFound(t *testing.T) {
	gdb, mock, cleanup := setupAccountsRepoTest(t)
	defer cleanup()

	repo := BuildRepository[AccountRepoImpl](gdb)
	ctx := context.Background()
	logger := setupAccountsLogger()

	auth := &model.Accounts{
		Login:    "user",
		Password: "password",
	}

	expectedErr := gorm.ErrRecordNotFound
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "accounts" WHERE login = $1 ORDER BY "accounts"."id" LIMIT $2`)).
		WithArgs("user", 1).
		WillReturnError(expectedErr)

	id, err := repo.Login(ctx, logger, auth)
	assert.NotNil(t, err, "неожиданная ошибка при авторизации")
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepoErrorLoginWrongPassword(t *testing.T) {
	gdb, mock, cleanup := setupAccountsRepoTest(t)
	defer cleanup()

	repo := BuildRepository[AccountRepoImpl](gdb)
	ctx := context.Background()
	logger := setupAccountsLogger()

	hashed, err := bcrypt.GenerateFromPassword([]byte("correct-pass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	auth := &model.Accounts{
		Login:    "user",
		Password: "wrong-pass",
	}
	expectedErr := errors.New("неверный пароль")
	rows := sqlmock.NewRows([]string{"id", "login", "password"}).
		AddRow(int64(10), "user", string(hashed))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "accounts" WHERE login = $1 ORDER BY "accounts"."id" LIMIT $2`)).
		WithArgs("user", 1).
		WillReturnRows(rows)

	id, err := repo.Login(ctx, logger, auth)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSuccessHashPassword(t *testing.T) {
	hash, err := hashPassword("password")
	assert.Nil(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, hash, "password")
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("password")))
}

func TestCheckPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.Nil(t, err)
	assert.True(t, checkPassword(string(hash), "password"))
	assert.False(t, checkPassword(string(hash), "wrong-password"))
}
