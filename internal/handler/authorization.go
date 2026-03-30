package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/model"
)

type AccountRepository interface {
	Create(ctx context.Context, log *slog.Logger, authorization *model.Accounts) error
	Login(ctx context.Context, log *slog.Logger, auth *model.Accounts) (*int64, error)
}

func BuildAuthorizationHandler(repo AccountRepository) AuthorizationHandlerImpl {
	return AuthorizationHandlerImpl{
		accountRepo: repo,
	}
}

type AuthorizationHandlerImpl struct {
	accountRepo AccountRepository
}

func (a *AuthorizationHandlerImpl) Register(log *slog.Logger, config *config.GophermartConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var accData model.Accounts
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		if err := decoder.Decode(&accData); err != nil {
			log.Error("Ошибка при конвертации тела запрос в JSON")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		if err := a.accountRepo.Create(r.Context(), log, &accData); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.Code == "23505" {
					http.Error(w, "логин уже занят", http.StatusConflict)
					return
				}
			}
			log.Error("Ошибка при добавлении аккаунта", "error", err)
			http.Error(w, "Ошибка при добавлении аккаунта", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		token, err := generateJWT(log, config, &accData)
		if err != nil {
			log.Error("ошибка при генерации jwt", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Authorization", "Bearer "+token)
		w.WriteHeader(http.StatusOK)
	}
}

func (a *AuthorizationHandlerImpl) Login(log *slog.Logger, config *config.GophermartConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var accData model.Accounts
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		if err := decoder.Decode(&accData); err != nil {
			log.Error("Ошибка при конвертации тела запрос в JSON")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		accountId, err := a.accountRepo.Login(r.Context(), log, &accData)
		if err != nil {
			log.Error("неверный логин / пароль")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		accData.ID = *accountId

		tokenString, err := generateJWT(log, config, &accData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Authorization", "Bearer "+tokenString)
		w.WriteHeader(http.StatusOK)
	}
}

func generateJWT(log *slog.Logger, cfg *config.GophermartConfig, accData *model.Accounts) (string, error) {
	claims := &model.Claims{
		AccountID: accData.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   accData.Login,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gophermart",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret))

	if err != nil {
		log.Error("Ошибка при получении токена")
		return "", err
	}
	return tokenString, nil
}
