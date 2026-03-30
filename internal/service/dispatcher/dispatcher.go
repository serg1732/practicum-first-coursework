package dispatcher

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/model"
)

type AccuralClient interface {
	GetOrdersAccrual(orderId string) (*model.AccrualResponse, *int, error)
}

type OrdersRepository interface {
	GetNewOrProcessingOrders(ctx context.Context, log *slog.Logger) ([]model.Order, error)
	UpdateOrderStatus(ctx context.Context, log *slog.Logger, orderId string, st string) error
	UpdateOrderStatusSum(ctx context.Context, log *slog.Logger, orderId string, st string, accural float64) error
}

func BuildDispatcher(client AccuralClient, ordersRepo OrdersRepository, updateChannel chan model.Order, processedChannel chan string) Dispatcher {
	return Dispatcher{
		client:                 client,
		ordersRepository:       ordersRepo,
		OrdersStartWorkChannel: updateChannel,
		OrdersUpdateProcessed:  processedChannel,
	}
}

type Dispatcher struct {
	client                 AccuralClient
	ordersRepository       OrdersRepository
	rwMutex                sync.RWMutex
	RateLimitWaitSeconds   int64
	OrdersStartWorkChannel chan model.Order
	OrdersUpdateProcessed  chan string
	orderInWork            map[string]any
}

func (d *Dispatcher) Run(ctx context.Context, log *slog.Logger, cfg *config.GophermartConfig) {
	for i := 0; i < cfg.RateLimit; i++ {
		go d.worker(ctx, log, cfg.RateLimitDelaySec)
	}
	go d.orderFinder(ctx, log)
	go d.orderCompleter(ctx, log)
}

func (d *Dispatcher) orderFinder(ctx context.Context, log *slog.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			orders, err := d.ordersRepository.GetNewOrProcessingOrders(ctx, log)
			if err != nil {
				log.Error("Ошибка при получении заказов", "error", err)
			}
			for _, order := range orders {
				d.rwMutex.RLock()
				if _, ok := d.orderInWork[order.OrderId]; !ok {
					d.OrdersStartWorkChannel <- order
				}
				d.rwMutex.RUnlock()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *Dispatcher) orderCompleter(ctx context.Context, log *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case orderId := <-d.OrdersUpdateProcessed:
			d.rwMutex.Lock()
			log.Debug("Обработан заказ", "orderId", orderId)
			delete(d.orderInWork, orderId)
			d.rwMutex.Unlock()
		}
	}
}

func (d *Dispatcher) worker(ctx context.Context, log *slog.Logger, rateLimitDelaySec int) {
	for {
		select {
		case <-ctx.Done():
			return
		case order := <-d.OrdersStartWorkChannel:
			defer func() {
				d.OrdersUpdateProcessed <- order.OrderId
			}()
			if order.Status == model.ORDER_STATUS_NEW {
				if err := d.ordersRepository.UpdateOrderStatus(ctx, log, order.OrderId, model.ORDER_STATUS_PROCESSING); err != nil {
					log.Error("Ошибка при изменении статуса заказа")
				}
			}

			resp, status, err := d.client.GetOrdersAccrual(order.OrderId)
			log.Debug("Получен ответ от accrual сервиса", "response", resp)
			if err != nil {
				log.Error("ошибка при запросе в accural сервис", "error", err)
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
		}
	}
}
