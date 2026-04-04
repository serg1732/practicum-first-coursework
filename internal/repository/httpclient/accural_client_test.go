package httpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestBuildAccrualClient(t *testing.T) {
	cfg := &config.GophermartConfig{
		AccrualAddress: "http://localhost:8080",
	}

	client := BuildAccuralClient(cfg)
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, cfg.AccrualAddress, client.url)
}

func TestSuccessAccrualClientGetOrdersAccrual(t *testing.T) {
	orderID := "12345"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/orders/" + orderID
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, expectedPath, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client := &AccuralClientImpl{
		httpClient: ts.Client(),
		url:        ts.URL,
	}

	resp, statusCode, err := client.GetOrdersAccrual(orderID)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, statusCode)
	assert.Equal(t, http.StatusOK, *statusCode)
}

func TestErrorAccrualClientGetOrdersAccrualNotStatusOk(t *testing.T) {
	orderID := "12345"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"too many requests"}`))
	}))
	defer ts.Close()

	client := &AccuralClientImpl{
		httpClient: ts.Client(),
		url:        ts.URL,
	}

	resp, statusCode, err := client.GetOrdersAccrual(orderID)
	assert.NoError(t, err)
	assert.Nil(t, resp)
	assert.NotNil(t, statusCode)
	assert.NotNil(t, *statusCode)
	assert.Equal(t, http.StatusTooManyRequests, *statusCode)
}

func TestErrorAccrualClientGetOrdersAccrual(t *testing.T) {
	client := &AccuralClientImpl{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("ошибка при зпросе")
			}),
		},
		url: "http://localhost:9080",
	}

	resp, statusCode, err := client.GetOrdersAccrual("12345")
	assert.NotNil(t, err)
	assert.Nil(t, resp)
	assert.Nil(t, statusCode)
}

func TestErrorAccrualClientGetOrdersAccrualRequestCreation(t *testing.T) {
	client := &AccuralClientImpl{
		httpClient: http.DefaultClient,
		url:        "://bad-url",
	}

	resp, statusCode, err := client.GetOrdersAccrual("12345")
	assert.NotNil(t, err)
	assert.Nil(t, resp)
	assert.Nil(t, statusCode)
}

func TestErrorAccrualClientGetOrdersAccrualInvalidJSON(t *testing.T) {
	orderID := "12345"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid-json}`))
	}))
	defer ts.Close()

	client := &AccuralClientImpl{
		httpClient: ts.Client(),
		url:        ts.URL,
	}

	resp, statusCode, err := client.GetOrdersAccrual(orderID)
	assert.NotNil(t, err)
	assert.Nil(t, resp)
	assert.Nil(t, statusCode)
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
