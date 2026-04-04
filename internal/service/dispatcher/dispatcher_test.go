package dispatcher

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/model"
	"github.com/serg1732/practicum-first-coursework/internal/service/dispatcher/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func initLogger() *slog.Logger {
	return slog.Default()
}

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestBuildDispatcher(t *testing.T) {
	client := mocks.NewAccrualClient(t)
	repo := mocks.NewOrdersRepository(t)
	ch := make(chan string, 1)

	d := BuildDispatcher(client, repo, ch)

	assert.NotNil(t, d.client)
	assert.NotNil(t, d.ordersRepository)
	assert.NotNil(t, d.OrdersUpdateProcessed)
	assert.NotNil(t, d.orderInWork)
	assert.Empty(t, d.orderInWork)
}

func TestSuccessProcessRequestNewOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	log := initLogger()

	client := mocks.NewAccrualClient(t)
	repo := mocks.NewOrdersRepository(t)
	processedCh := make(chan string, 1)

	d := &Dispatcher{
		client:                client,
		ordersRepository:      repo,
		OrdersUpdateProcessed: processedCh,
		orderInWork:           map[string]any{},
	}

	order := &model.Order{
		OrderId: "12345",
		Status:  model.ORDER_STATUS_NEW,
	}
	resp := &model.AccrualResponse{
		Status: model.ORDER_STATUS_REGISTERED,
	}
	repo.On("UpdateOrderStatus", mock.Anything, mock.Anything, order.OrderId, model.ORDER_STATUS_PROCESSING).
		Return(nil).
		Once()
	client.On("GetOrdersAccrual", order.OrderId).
		Return(resp, intPtr(http.StatusOK), nil).
		Once()

	err := d.processRequest(ctx, log, order, 0)
	assert.NoError(t, err)

	select {
	case got := <-processedCh:
		assert.Equal(t, order.OrderId, got)
	default:
		t.Fatal("expected processed order id in channel")
	}

	repo.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestErrorProcessRequestOrderMarkedInvalid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	log := initLogger()

	client := mocks.NewAccrualClient(t)
	repo := mocks.NewOrdersRepository(t)
	processedCh := make(chan string, 1)

	d := &Dispatcher{
		client:                client,
		ordersRepository:      repo,
		OrdersUpdateProcessed: processedCh,
		orderInWork:           map[string]any{},
	}

	order := &model.Order{
		OrderId: "12345",
		Status:  model.ORDER_STATUS_PROCESSING,
	}
	client.On("GetOrdersAccrual", order.OrderId).
		Return(nil, intPtr(http.StatusInternalServerError), assert.AnError).
		Once()
	repo.On("UpdateOrderStatus", mock.Anything, mock.Anything, order.OrderId, model.ORDER_STATUS_INVALID).
		Return(nil).
		Once()

	err := d.processRequest(ctx, log, order, 0)
	assert.NoError(t, err)

	select {
	case got := <-processedCh:
		assert.Equal(t, order.OrderId, got)
	default:
		t.Fatal("expected processed order id in channel")
	}

	repo.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestSuccessPocessRequestStatusTooManyRequests(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	log := initLogger()

	client := mocks.NewAccrualClient(t)
	repo := mocks.NewOrdersRepository(t)
	processedCh := make(chan string, 1)

	d := &Dispatcher{
		client:                client,
		ordersRepository:      repo,
		OrdersUpdateProcessed: processedCh,
		orderInWork:           map[string]any{},
	}

	order := &model.Order{
		OrderId: "12345",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	client.On("GetOrdersAccrual", order.OrderId).
		Return(nil, intPtr(http.StatusTooManyRequests), nil).
		Once()

	start := time.Now()
	err := d.processRequest(ctx, log, order, 1)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed, time.Second)

	select {
	case got := <-processedCh:
		assert.Equal(t, order.OrderId, got)
	default:
		t.Fatal("expected processed order id in channel")
	}

	repo.AssertNotCalled(t, "UpdateOrderStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "UpdateOrderStatusSum", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	client.AssertExpectations(t)
}

func TestSuccessProcessRequestStatusWithAccrual(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	log := initLogger()

	client := mocks.NewAccrualClient(t)
	repo := mocks.NewOrdersRepository(t)
	processedCh := make(chan string, 1)

	d := &Dispatcher{
		client:                client,
		ordersRepository:      repo,
		OrdersUpdateProcessed: processedCh,
		orderInWork:           map[string]any{},
	}

	order := &model.Order{
		OrderId: "12345",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	accrual := 150.75
	resp := &model.AccrualResponse{
		Status:  model.ORDER_STATUS_PROCESSED,
		Accrual: float64Ptr(accrual),
	}

	client.On("GetOrdersAccrual", order.OrderId).
		Return(resp, intPtr(http.StatusOK), nil).
		Once()

	repo.On("UpdateOrderStatusSum", mock.Anything, mock.Anything, order.OrderId, model.ORDER_STATUS_PROCESSED, accrual).
		Return(nil).
		Once()

	err := d.processRequest(ctx, log, order, 0)
	assert.NoError(t, err)

	select {
	case got := <-processedCh:
		assert.Equal(t, order.OrderId, got)
	default:
		t.Fatal("expected processed order id in channel")
	}

	repo.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestSuccessProcessRequestStatusWithoutAccrual(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	log := initLogger()

	client := mocks.NewAccrualClient(t)
	repo := mocks.NewOrdersRepository(t)
	processedCh := make(chan string, 1)

	d := &Dispatcher{
		client:                client,
		ordersRepository:      repo,
		OrdersUpdateProcessed: processedCh,
		orderInWork:           map[string]any{},
	}

	order := &model.Order{
		OrderId: "12345",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	resp := &model.AccrualResponse{
		Status:  model.ORDER_STATUS_INVALID,
		Accrual: nil,
	}

	client.On("GetOrdersAccrual", order.OrderId).
		Return(resp, intPtr(http.StatusOK), nil).
		Once()

	repo.On("UpdateOrderStatus", mock.Anything, mock.Anything, order.OrderId, model.ORDER_STATUS_INVALID).
		Return(nil).
		Once()

	err := d.processRequest(ctx, log, order, 0)
	assert.NoError(t, err)

	select {
	case got := <-processedCh:
		assert.Equal(t, order.OrderId, got)
	default:
		t.Fatal("expected processed order id in channel")
	}

	repo.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestSuccessProcessRequestRegisteredOrProcessing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status string
	}{
		{
			name:   "Статус регистрации запроса",
			status: model.ORDER_STATUS_REGISTERED,
		},
		{
			name:   "Статус в процессе",
			status: model.ORDER_STATUS_PROCESSING,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			log := initLogger()

			client := mocks.NewAccrualClient(t)
			repo := mocks.NewOrdersRepository(t)
			processedCh := make(chan string, 1)

			d := &Dispatcher{
				client:                client,
				ordersRepository:      repo,
				OrdersUpdateProcessed: processedCh,
				orderInWork:           map[string]any{},
			}

			order := &model.Order{
				OrderId: "12345",
				Status:  model.ORDER_STATUS_PROCESSING,
			}

			resp := &model.AccrualResponse{
				Status: tt.status,
			}

			client.On("GetOrdersAccrual", order.OrderId).
				Return(resp, intPtr(http.StatusOK), nil).
				Once()

			err := d.processRequest(ctx, log, order, 0)
			assert.NoError(t, err)

			select {
			case got := <-processedCh:
				assert.Equal(t, order.OrderId, got)
			default:
				t.Fatal("expected processed order id in channel")
			}

			repo.AssertNotCalled(t, "UpdateOrderStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			repo.AssertNotCalled(t, "UpdateOrderStatusSum", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			client.AssertExpectations(t)
		})
	}
}

func TestSuccessOrderCompleterCheckRemove(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := initLogger()
	d := &Dispatcher{
		OrdersUpdateProcessed: make(chan string, 1),
		orderInWork: map[string]any{
			"12345": struct{}{},
		},
	}

	go d.orderCompleter(ctx, log)
	d.OrdersUpdateProcessed <- "12345"

	assert.Eventually(t, func() bool {
		d.mutex.Lock()
		defer d.mutex.Unlock()

		_, exists := d.orderInWork["12345"]
		return !exists
	}, time.Second, 10*time.Millisecond)
}

func TestSuccessStartsWorkers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := initLogger()

	client := mocks.NewAccrualClient(t)
	repo := mocks.NewOrdersRepository(t)
	processedCh := make(chan string, 1)

	d := &Dispatcher{
		client:                client,
		ordersRepository:      repo,
		OrdersUpdateProcessed: processedCh,
		orderInWork:           map[string]any{},
	}

	repo.On("GetNewOrProcessingOrders", mock.Anything, mock.Anything).
		Return([]model.Order{}, nil).
		Maybe()

	cfg := &config.GophermartConfig{
		RateLimit:         1,
		RateLimitDelaySec: 0,
	}

	d.Run(ctx, log, cfg)

	time.Sleep(1200 * time.Millisecond)
	cancel()

	repo.AssertCalled(t, "GetNewOrProcessingOrders", mock.Anything, mock.Anything)
}
