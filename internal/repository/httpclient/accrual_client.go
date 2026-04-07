package httpclient

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/model"
)

// BuildAccuralClient создание http клиента для запросов в Accrual сервис
func BuildAccuralClient(cfg *config.GophermartConfig) *AccuralClientImpl {
	return &AccuralClientImpl{
		httpClient: http.DefaultClient,
		url:        cfg.AccrualAddress,
	}
}

// AccuralClientImpl http клиент по работе с Accrual
type AccuralClientImpl struct {
	httpClient *http.Client
	url        string
}

// GetOrdersAccrual запрос на получение баллов за заказ
func (ac *AccuralClientImpl) GetOrdersAccrual(orderID string) (*model.AccrualResponse, *int, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/orders/%s", ac.url, orderID), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &resp.StatusCode, nil
	}

	var response model.AccrualResponse
	decoder := json.NewDecoder(resp.Body)
	if errDecode := decoder.Decode(&response); errDecode != nil {
		return nil, nil, errDecode
	}
	return &response, &resp.StatusCode, nil
}
