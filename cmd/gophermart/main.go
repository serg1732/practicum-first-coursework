package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/serg1732/practicum-first-coursework/internal/handler"
	"github.com/serg1732/practicum-first-coursework/internal/model"
	"github.com/serg1732/practicum-first-coursework/internal/repository/http_client"
	"github.com/serg1732/practicum-first-coursework/internal/repository/sql"
	"github.com/serg1732/practicum-first-coursework/internal/service/dispatcher"

	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/logger"
)

func main() {
	log := logger.NewSlogLogger(slog.LevelInfo)
	log.Debug("Старт модуля")
	serverConfig, errConfig := config.GetGophermartConfig()
	log.Debug("Прочитан корнфиг", "config", serverConfig)
	if errConfig != nil {
		log.Error("Ошибка парсинга env значений", "error", errConfig)
	}
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if serverConfig.DSN == "" {
		log.Error("Не заданы настройки БД")
		os.Exit(1)
	}
	log.Debug("Подключение к БД")
	db, err := sql.BuildConnection(log, serverConfig)
	if err != nil {
		log.Error("Ошибка при подключении к БД", "error", err)
		os.Exit(1)
	}
	log.Debug("Миграция БД")
	if errMigrate := sql.MigrateDataBase(log, serverConfig); errMigrate != nil {
		log.Error("Ошибка миграции БД", "error", errMigrate)
		os.Exit(1)
	}

	mux := chi.NewRouter()
	accountRepo := sql.BuildAccountsRepo(db)
	orderRepo := sql.BuildOrdersRepo(db)
	balanceRepo := sql.BuildBalanceRepo(db)
	withdrawRepo := sql.BuildWithdrawsRepo(db)
	chUpdate := make(chan model.Order, serverConfig.RateLimit)
	chProcessed := make(chan string, serverConfig.RateLimit)
	defer close(chUpdate)
	defer close(chProcessed)
	dispatchService := dispatcher.BuildDispatcher(http_client.BuildAccuralClient(serverConfig), &orderRepo, chUpdate, chProcessed)
	log.Debug("Запуск диспатчера")
	dispatchService.Run(ctx, log, serverConfig)
	mux = buildRoute(log, serverConfig, &accountRepo, &orderRepo, &balanceRepo, &withdrawRepo)

	log.Info("Запуск http сервера", "address", serverConfig.RunAddr)
	srv := &http.Server{
		Addr:    serverConfig.RunAddr,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Ошибка в http сервере", "error", err)
			return
		}
		log.Info("Завершение работы http сервера")
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Ошибка при завершении работы http сервера", "error", err)
	}
}

func buildRoute(log *slog.Logger, serverConfig *config.GophermartConfig, accountRepository handler.AccountRepository,
	orderRepository handler.OrdersRepository, balanceRepo handler.BalanceRepository, withdrawRepository handler.WithdrawsRepository,
) *chi.Mux {
	r := chi.NewRouter()
	authHandler := handler.BuildAuthorizationHandler(accountRepository)
	ordersHandler := handler.BuildOrdersHandler(orderRepository)
	withdrawHandler := handler.BuildWithdrawHandler(balanceRepo, withdrawRepository, orderRepository)

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", authHandler.Register(log, serverConfig))
		r.Post("/login", authHandler.Login(log, serverConfig))
		r.Group(func(r chi.Router) {
			r.Use(handler.JwtMiddleware(serverConfig))
			r.Get("/orders", ordersHandler.GetAllOrders(log))
			r.Post("/orders", ordersHandler.AddNewOrder(log))
			r.Get("/withdrawals", withdrawHandler.GetAllWithdraw(log))
		})
		r.Route("/balance", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(handler.JwtMiddleware(serverConfig))
				r.Get("/", withdrawHandler.BalanceRequest(log))
				r.Post("/withdraw", withdrawHandler.WithdrawRequest(log))
			})
		})
	})
	return r
}
