package sql

import (
	"context"
	"errors"
	"log/slog"

	"github.com/serg1732/practicum-first-coursework/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AccountRepoImpl репозиторий аккаунтов
type AccountRepoImpl struct {
	db *gorm.DB
}

// Create создание аккаунта
func (a AccountRepoImpl) Create(ctx context.Context, log *slog.Logger, authorization *model.Accounts) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		hashPass, errHash := hashPassword(authorization.Password)
		if errHash != nil {
			log.Error("Ошибка получения hash значения пароля", "error", errHash)
			return errHash
		}
		authorization.Password = hashPass
		if err := tx.WithContext(ctx).Create(authorization).Error; err != nil {
			log.Error("Ошибка создания аккаунта", "error", err)
			return err
		}

		if err := tx.WithContext(ctx).Table("balance").Create(&model.BalanceDB{AccountID: authorization.ID}).Error; err != nil {
			log.Error("Ошибка при добавлении баланса", "error", err)
			return err
		}
		return nil
	})
}

// Login авторизация пользователя
func (a AccountRepoImpl) Login(ctx context.Context, log *slog.Logger, auth *model.Accounts) (*int64, error) {
	var account model.Accounts
	err := a.db.WithContext(ctx).Where("login = ?", auth.Login).First(&account).Error
	if err != nil {
		log.Error("Аккаунт не найден", "error", err)
		return nil, err
	}

	if !checkPassword(account.Password, auth.Password) {
		return nil, errors.New("неверный пароль")
	}

	return &account.ID, nil
}

// hashPassword получение hash пароля
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPassword проверка пароля
func checkPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
