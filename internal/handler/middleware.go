package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/model"
)

func JwtMiddleware(config *config.GophermartConfig) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractBearerToken(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims := &model.Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("неизвестный метод подписи")
				}
				return []byte(config.Secret), nil
			})

			if claims.ExpiresAt.Before(time.Now()) {
				http.Error(w, "Истек срок действия токена", http.StatusUnauthorized)
				return
			}

			if err != nil {
				http.Error(w, "некорректный токен: "+err.Error(), http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				http.Error(w, "некорректный токен", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), "account_id", claims.AccountID)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("заголовой с токеном обязателен")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("некорректный формат Bearer")
	}

	return parts[1], nil
}
