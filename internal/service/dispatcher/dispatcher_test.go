package dispatcher

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/serg1732/practicum-first-coursework/internal/model"
	"github.com/serg1732/practicum-first-coursework/internal/service/dispatcher/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testDispatherLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func intPtr(v int) *int {
	return &v
}

func TestBuildDispatcher(t *testing.T) {
	client := &mocks.AccuralClient{}
	repo := &mocks.OrdersRepository{}
	updateCh := make(chan model.Order, 1)
	processedCh := make(chan string, 1)

	d := BuildDispatcher(client, repo, updateCh, processedCh)

	require.Equal(t, client, d.client)
	require.Equal(t, repo, d.ordersRepository)
	require.Equal(t, updateCh, d.OrdersStartWorkChannel)
	require.Equal(t, processedCh, d.OrdersUpdateProcessed)
}

func TestErrorClientErrorSetProcessingInvalid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testDispatherLogger()
	client := mocks.NewAccuralClient(t)
	repo := mocks.NewOrdersRepository(t)

	order := model.Order{
		OrderId: "12345",
		Status:  model.ORDER_STATUS_NEW,
	}

	repo.EXPECT().
		UpdateOrderStatus(ctx, mock.Anything, order.OrderId, model.ORDER_STATUS_PROCESSING).
		Return(nil).
		Once()

	client.EXPECT().
		GetOrdersAccrual(order.OrderId).
		Return(nil, intPtr(http.StatusInternalServerError), errors.New("accrual error")).
		Once()

	repo.EXPECT().
		UpdateOrderStatus(ctx, mock.Anything, order.OrderId, model.ORDER_STATUS_INVALID).
		Return(nil).
		Once()

	d := Dispatcher{
		client:                 client,
		ordersRepository:       repo,
		OrdersStartWorkChannel: make(chan model.Order, 1),
		OrdersUpdateProcessed:  make(chan string, 1),
		orderInWork:            make(map[string]any),
	}

	go d.worker(ctx, log, 1)

	d.OrdersStartWorkChannel <- order

	require.Eventually(t, func() bool {
		return true
	}, 150*time.Millisecond, 10*time.Millisecond)
}

func TestWorkerSuccessWithAccrualUpdateStatusSum(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testDispatherLogger()
	client := mocks.NewAccuralClient(t)
	repo := mocks.NewOrdersRepository(t)

	order := model.Order{
		OrderId: "777",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	accrual := 150.75

	client.EXPECT().
		GetOrdersAccrual(order.OrderId).
		Return(&model.AccrualResponse{
			Status:  model.ORDER_STATUS_PROCESSED,
			Accrual: &accrual,
		}, intPtr(http.StatusOK), nil).
		Once()

	repo.EXPECT().
		UpdateOrderStatusSum(ctx, mock.Anything, order.OrderId, model.ORDER_STATUS_PROCESSED, accrual).
		Return(nil).
		Once()

	d := Dispatcher{
		client:                 client,
		ordersRepository:       repo,
		OrdersStartWorkChannel: make(chan model.Order, 1),
		OrdersUpdateProcessed:  make(chan string, 1),
		orderInWork:            make(map[string]any),
	}

	go d.worker(ctx, log, 1)

	d.OrdersStartWorkChannel <- order

	require.Eventually(t, func() bool {
		return true
	}, 150*time.Millisecond, 10*time.Millisecond)
}

func TestWorkerSuccessWithoutAccrualAndUpdateOnlyStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testDispatherLogger()
	client := mocks.NewAccuralClient(t)
	repo := mocks.NewOrdersRepository(t)

	order := model.Order{
		OrderId: "555",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	client.EXPECT().
		GetOrdersAccrual(order.OrderId).
		Return(&model.AccrualResponse{
			Status:  model.ORDER_STATUS_INVALID,
			Accrual: nil,
		}, intPtr(http.StatusOK), nil).
		Once()

	repo.EXPECT().
		UpdateOrderStatus(ctx, mock.Anything, order.OrderId, model.ORDER_STATUS_INVALID).
		Return(nil).
		Once()

	d := Dispatcher{
		client:                 client,
		ordersRepository:       repo,
		OrdersStartWorkChannel: make(chan model.Order, 1),
		OrdersUpdateProcessed:  make(chan string, 1),
		orderInWork:            make(map[string]any),
	}

	go d.worker(ctx, log, 1)

	d.OrdersStartWorkChannel <- order

	require.Eventually(t, func() bool {
		return true
	}, 150*time.Millisecond, 10*time.Millisecond)
}

func TestWorkerSuccessRegisteredNotUpdateOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testDispatherLogger()
	client := mocks.NewAccuralClient(t)
	repo := mocks.NewOrdersRepository(t)

	order := model.Order{
		OrderId: "101",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	client.EXPECT().
		GetOrdersAccrual(order.OrderId).
		Return(&model.AccrualResponse{
			Status: model.ORDER_STATUS_REGISTERED,
		}, intPtr(http.StatusOK), nil).
		Once()

	d := Dispatcher{
		client:                 client,
		ordersRepository:       repo,
		OrdersStartWorkChannel: make(chan model.Order, 1),
		OrdersUpdateProcessed:  make(chan string, 1),
		orderInWork:            make(map[string]any),
	}

	go d.worker(ctx, log, 1)

	d.OrdersStartWorkChannel <- order

	require.Eventually(t, func() bool {
		return true
	}, 150*time.Millisecond, 10*time.Millisecond)
}

func TestWorkerSuccessProcessingNotUpdateOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testDispatherLogger()
	client := mocks.NewAccuralClient(t)
	repo := mocks.NewOrdersRepository(t)

	order := model.Order{
		OrderId: "102",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	client.EXPECT().
		GetOrdersAccrual(order.OrderId).
		Return(&model.AccrualResponse{
			Status: model.ORDER_STATUS_PROCESSING,
		}, intPtr(http.StatusOK), nil).
		Once()

	d := Dispatcher{
		client:                 client,
		ordersRepository:       repo,
		OrdersStartWorkChannel: make(chan model.Order, 1),
		OrdersUpdateProcessed:  make(chan string, 1),
		orderInWork:            make(map[string]any),
	}

	go d.worker(ctx, log, 1)

	d.OrdersStartWorkChannel <- order

	require.Eventually(t, func() bool {
		return true
	}, 150*time.Millisecond, 10*time.Millisecond)
}

func TestWorkerErrorStatusTooManyRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testDispatherLogger()
	client := mocks.NewAccuralClient(t)
	repo := mocks.NewOrdersRepository(t)

	order := model.Order{
		OrderId: "429-order",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	client.EXPECT().
		GetOrdersAccrual(order.OrderId).
		Return(nil, intPtr(http.StatusTooManyRequests), nil).
		Once()

	d := Dispatcher{
		client:                 client,
		ordersRepository:       repo,
		OrdersStartWorkChannel: make(chan model.Order, 1),
		OrdersUpdateProcessed:  make(chan string, 1),
		orderInWork:            make(map[string]any),
	}

	start := time.Now()

	go d.worker(ctx, log, 1)

	d.OrdersStartWorkChannel <- order

	time.Sleep(1100 * time.Millisecond)

	require.GreaterOrEqual(t, time.Since(start), time.Second)
}

func TestWorkerSuccessRemovesOrderFromWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testDispatherLogger()

	d := Dispatcher{
		OrdersUpdateProcessed: make(chan string, 1),
		orderInWork: map[string]any{
			"order-1": struct{}{},
		},
	}

	go d.orderCompleter(ctx, log)

	d.OrdersUpdateProcessed <- "order-1"

	require.Eventually(t, func() bool {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		_, exists := d.orderInWork["order-1"]
		return !exists
	}, time.Second, 10*time.Millisecond)
}

func TestWorkerSuccessSendProcessedNotification(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := testDispatherLogger()
	client := mocks.NewAccuralClient(t)
	repo := mocks.NewOrdersRepository(t)

	order := model.Order{
		OrderId: "processed-1",
		Status:  model.ORDER_STATUS_PROCESSING,
	}

	client.EXPECT().
		GetOrdersAccrual(order.OrderId).
		Return(&model.AccrualResponse{
			Status: model.ORDER_STATUS_PROCESSING,
		}, intPtr(http.StatusOK), nil).
		Once()

	d := Dispatcher{
		client:                 client,
		ordersRepository:       repo,
		OrdersStartWorkChannel: make(chan model.Order, 1),
		OrdersUpdateProcessed:  make(chan string, 1),
		orderInWork:            make(map[string]any),
	}

	go d.worker(ctx, log, 1)

	d.OrdersStartWorkChannel <- order

	select {
	case got := <-d.OrdersUpdateProcessed:
		require.Equal(t, order.OrderId, got)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("expected processed notification, but got none")
	}
}
