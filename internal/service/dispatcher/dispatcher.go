package dispatcher

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/model"
)

type AccrualClient interface {
	GetOrdersAccrual(orderId string) (*model.AccrualResponse, *int, error)
}

type OrdersRepository interface {
	GetNewOrProcessingOrders(ctx context.Context, log *slog.Logger) ([]model.Order, error)
	UpdateOrderStatus(ctx context.Context, log *slog.Logger, orderId string, st string) error
	UpdateOrderStatusSum(ctx context.Context, log *slog.Logger, orderId string, st string, accural float64) error
}

func BuildDispatcher(client AccrualClient, ordersRepo OrdersRepository, processedChannel chan string) Dispatcher {
	return Dispatcher{
		client:                client,
		ordersRepository:      ordersRepo,
		OrdersUpdateProcessed: processedChannel,
		orderInWork:           map[string]any{},
	}
}

type Dispatcher struct {
	client                AccrualClient
	ordersRepository      OrdersRepository
	mutex                 sync.Mutex
	RateLimitWaitSeconds  int64
	OrdersUpdateProcessed chan string
	orderInWork           map[string]any
}

// Run запуск обработчиков заказов
func (d *Dispatcher) Run(ctx context.Context, log *slog.Logger, cfg *config.GophermartConfig) {
	go d.orderFinder(ctx, log, cfg.RateLimit, cfg.RateLimitDelaySec)
	go d.orderCompleter(ctx, log)
}

// orderFinder отбирает заказы и запускает их в обработку
func (d *Dispatcher) orderFinder(ctx context.Context, log *slog.Logger, rateLimit int, delayRateLimit int) {
	ticker := time.NewTicker(1 * time.Second)
	g, groupContext := errgroup.WithContext(ctx)
	g.SetLimit(rateLimit)
	for {
		select {
		case <-ticker.C:
			orders, err := d.ordersRepository.GetNewOrProcessingOrders(ctx, log)
			if err != nil {
				log.Error("Ошибка при получении заказов", "error", err)
			}
			for _, order := range orders {
				d.mutex.Lock()
				if _, ok := d.orderInWork[order.OrderId]; ok {
					d.mutex.Unlock()
					continue
				}
				d.orderInWork[order.OrderId] = order
				d.mutex.Unlock()
				g.Go(func() error {
					return d.processRequest(groupContext, log, &order, delayRateLimit)
				})
			}
		case <-ctx.Done():
			return
		}
	}
}

// orderCompleter завершает работу с заказом
func (d *Dispatcher) orderCompleter(ctx context.Context, log *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case orderId := <-d.OrdersUpdateProcessed:
			d.mutex.Lock()
			log.Debug("Обработан заказ", "orderId", orderId)
			delete(d.orderInWork, orderId)
			d.mutex.Unlock()
		}
	}
}

func (d *Dispatcher) processRequest(ctx context.Context, log *slog.Logger, order *model.Order, rateLimitDelaySec int) error {
	if order.Status == model.ORDER_STATUS_NEW {
		if err := d.ordersRepository.UpdateOrderStatus(ctx, log, order.OrderId, model.ORDER_STATUS_PROCESSING); err != nil {
			log.Error("Ошибка при изменении статуса заказа")
		}
	}

	resp, status, err := d.client.GetOrdersAccrual(order.OrderId)
	log.Debug("Получен ответ от accrual сервиса", "response", resp)
	if err != nil {
		log.Error("ошибка при запросе в accrual сервис", "error", err)
		if errUpdate := d.ordersRepository.UpdateOrderStatus(ctx, log, order.OrderId, model.ORDER_STATUS_INVALID); errUpdate != nil {
			log.Error("Ошибка обновления данных заказа", "error", errUpdate)
		}
	} else if *status != http.StatusOK {
		if *status == http.StatusTooManyRequests {
			log.Debug("Слишком много запросов в accrual сервис, принудительное ожидание", "seconds",
				rateLimitDelaySec,
			)
			timer := time.NewTicker(time.Duration(rateLimitDelaySec) * time.Second)
			<-timer.C
		} else {
			log.Error("Ошибка при получении данных")
		}
	} else if resp != nil && (resp.Status != model.ORDER_STATUS_REGISTERED && resp.Status != model.ORDER_STATUS_PROCESSING) {
		if resp.Accrual != nil {
			if errUpdate := d.ordersRepository.UpdateOrderStatusSum(ctx, log, order.OrderId, resp.Status, *resp.Accrual); errUpdate != nil {
				log.Error("Ошибка обновления данных заказа", "error", errUpdate)
			}
		} else {
			if errUpdate := d.ordersRepository.UpdateOrderStatus(ctx, log, order.OrderId, resp.Status); errUpdate != nil {
				log.Error("Ошибка обновления данных заказа", "error", errUpdate)
			}
		}
	}
	d.OrdersUpdateProcessed <- order.OrderId
	return nil
}
