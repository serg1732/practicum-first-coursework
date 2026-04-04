package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/handler/mocks"
	"github.com/serg1732/practicum-first-coursework/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestConfig() *config.GophermartConfig {
	return &config.GophermartConfig{
		Secret: "test-secret",
	}
}

func TestAuthorizationHandlerSuccessRegister(t *testing.T) {
	repo := mocks.NewAccountRepository(t)
	h := BuildAuthorizationHandler(repo)

	log := newTestLogger()
	cfg := newTestConfig()

	body := `{"login":"user","password":"password"}`

	repo.
		On("Create", mock.Anything, mock.Anything, mock.MatchedBy(func(acc *model.Accounts) bool {
			return acc.Login == "user" && acc.Password == "password"
		})).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler := h.Register(log, cfg)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	authHeader := res.Header.Get("Authorization")
	require.NotEmpty(t, authHeader)
	assert.Contains(t, authHeader, "Bearer ")
}

func TestErrorAuthorizationHandlerRegisterConflict(t *testing.T) {
	repo := mocks.NewAccountRepository(t)
	h := BuildAuthorizationHandler(repo)

	log := newTestLogger()
	cfg := newTestConfig()

	body := `{"login":"testuser","password":"password"}`

	repo.
		On("Create", mock.Anything, mock.Anything, mock.AnythingOfType("*model.Accounts")).
		Return(&pgconn.PgError{Code: "23505"}).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler := h.Register(log, cfg)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusConflict, res.StatusCode)
}

func TestErrorAuthorizationHandlerRegisterCreateError(t *testing.T) {
	repo := mocks.NewAccountRepository(t)
	h := BuildAuthorizationHandler(repo)

	log := newTestLogger()
	cfg := newTestConfig()

	body := `{"login":"testuser","password":"password"}`

	repo.
		On("Create", mock.Anything, mock.Anything, mock.AnythingOfType("*model.Accounts")).
		Return(errors.New("db error")).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler := h.Register(log, cfg)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}

func TestErrorAuthorizationHandlerRegisterInvalidJSON(t *testing.T) {
	repo := mocks.NewAccountRepository(t)
	h := BuildAuthorizationHandler(repo)

	log := newTestLogger()
	cfg := newTestConfig()

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{bad json`))
	rec := httptest.NewRecorder()

	handler := h.Register(log, cfg)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.NotEqual(t, http.StatusOK, res.StatusCode)
}

func TestSuccessAuthorizationHandlerLogin(t *testing.T) {
	repo := mocks.NewAccountRepository(t)
	h := BuildAuthorizationHandler(repo)

	log := newTestLogger()
	cfg := newTestConfig()

	body := `{"login":"testuser","password":"password"}`
	accountID := int64(42)

	repo.
		On("Login", mock.Anything, mock.Anything, mock.MatchedBy(func(acc *model.Accounts) bool {
			return acc.Login == "testuser" && acc.Password == "password"
		})).
		Return(&accountID, nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler := h.Login(log, cfg)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	authHeader := res.Header.Get("Authorization")
	require.NotEmpty(t, authHeader)
	assert.Contains(t, authHeader, "Bearer ")
}

func TestErrorAuthorizationHandlerLoginInvalidCredentials(t *testing.T) {
	repo := mocks.NewAccountRepository(t)
	h := BuildAuthorizationHandler(repo)

	log := newTestLogger()
	cfg := newTestConfig()

	body := `{"login":"testuser","password":"wrong-password"}`

	repo.
		On("Login", mock.Anything, mock.Anything, mock.AnythingOfType("*model.Accounts")).
		Return(nil, errors.New("некорректный логи / пароль")).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler := h.Login(log, cfg)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestErrorAuthorizationHandlerLoginBadJSON(t *testing.T) {
	repo := mocks.NewAccountRepository(t)
	h := BuildAuthorizationHandler(repo)

	log := newTestLogger()
	cfg := newTestConfig()

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(`{bad json`))
	rec := httptest.NewRecorder()

	handler := h.Login(log, cfg)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	assert.NotEqual(t, http.StatusOK, res.StatusCode)
}

func TestSuccessGenerateJWT(t *testing.T) {
	log := newTestLogger()
	cfg := newTestConfig()

	acc := &model.Accounts{
		ID:       42,
		Login:    "testuser",
		Password: "password",
	}

	tokenString, err := generateJWT(log, cfg, acc)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	token, err := jwt.ParseWithClaims(tokenString, &model.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(*model.Claims)
	require.True(t, ok)

	assert.Equal(t, int64(42), claims.AccountID)
	assert.Equal(t, "testuser", claims.Subject)
	assert.Equal(t, "gophermart", claims.Issuer)
}

func TestBuildAuthorizationHandler(t *testing.T) {
	repo := mocks.NewAccountRepository(t)

	h := BuildAuthorizationHandler(repo)

	assert.NotNil(t, h)
	assert.NotNil(t, h.accountRepo)
}

func TestSuccessAuthorizationHandlerRegisterRequest(t *testing.T) {
	repo := mocks.NewAccountRepository(t)
	h := BuildAuthorizationHandler(repo)

	log := newTestLogger()
	cfg := newTestConfig()

	body := `{"login":"testuser","password":"password"}`

	ctxVal := "test-value"

	repo.
		On("Create",
			mock.MatchedBy(func(ctx context.Context) bool {
				return ctx.Value(accountIDKey) == ctxVal
			}),
			mock.Anything,
			mock.AnythingOfType("*model.Accounts"),
		).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	req = req.WithContext(context.WithValue(req.Context(), accountIDKey, ctxVal))
	rec := httptest.NewRecorder()

	handler := h.Register(log, cfg)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
