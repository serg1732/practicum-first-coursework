package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantToken   string
		wantErrText string
	}{
		{
			name:       "Валидный токен",
			authHeader: "Bearer my-token",
			wantToken:  "my-token",
		},
		{
			name:        "Пустой заголовой",
			authHeader:  "",
			wantErrText: "заголовой с токеном обязателен",
		},
		{
			name:        "Некорректный токен",
			authHeader:  "Basic my-token",
			wantErrText: "некорректный формат Bearer",
		},
		{
			name:        "Пустой токен",
			authHeader:  "Bearer",
			wantErrText: "некорректный формат Bearer",
		},
		{
			name:       "Невалидный токен",
			authHeader: "Bearer ",
			wantToken:  "",
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if td.authHeader != "" {
				req.Header.Set("Authorization", td.authHeader)
			}

			actualToken, err := extractBearerToken(req)
			if td.wantErrText != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", td.wantErrText)
				}
				if err.Error() != td.wantErrText {
					t.Fatalf("unexpected error: got %q, want %q", err.Error(), td.wantErrText)
				}
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, td.wantToken, actualToken)
		})
	}
}

func TestJwtMiddlewareSuccess(t *testing.T) {
	cfg := &config.GophermartConfig{
		Secret: "test-secret",
	}

	var accountID int64 = 42

	tokenString := makeHMACToken(t, cfg.Secret, &model.Claims{
		AccountID: accountID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	var gotAccountID any
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccountID = r.Context().Value(accountIDKey)
		w.WriteHeader(http.StatusOK)
	})

	handler := JwtMiddleware(cfg)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, accountID, gotAccountID)
}

func TestJwtMiddlewareErrorNoAuthorizationHeader(t *testing.T) {
	cfg := &config.GophermartConfig{
		Secret: "test-secret",
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := JwtMiddleware(cfg)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func TestJwtMiddlewareErrorInvalidBearerFormat(t *testing.T) {
	cfg := &config.GophermartConfig{
		Secret: "test-secret",
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := JwtMiddleware(cfg)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func TestJwtMiddlewareErrorExpiredToken(t *testing.T) {
	cfg := &config.GophermartConfig{
		Secret: "test-secret",
	}

	tokenString := makeHMACToken(t, cfg.Secret, &model.Claims{
		AccountID: 42,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := JwtMiddleware(cfg)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func TestJwtMiddlewareErrorInvalidSignature(t *testing.T) {
	cfg := &config.GophermartConfig{
		Secret: "correct-secret",
	}

	tokenString := makeHMACToken(t, "wrong-secret", &model.Claims{
		AccountID: 42,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := JwtMiddleware(cfg)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func TestJwtMiddlewareErrorUnknownSigningMethod(t *testing.T) {
	cfg := &config.GophermartConfig{
		Secret: "test-secret",
	}

	tokenString := makeRSAToken(t, &model.Claims{
		AccountID: 42,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := JwtMiddleware(cfg)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func makeHMACToken(t *testing.T, secret string, claims *model.Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)
	return tokenString
}

func makeRSAToken(t *testing.T, claims *model.Claims) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	assert.NoError(t, err)
	return tokenString
}
