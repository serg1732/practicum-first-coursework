package http_client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/serg1732/practicum-first-coursework/internal/config"
	"github.com/serg1732/practicum-first-coursework/internal/model"
)

func BuildAccuralClient(cfg *config.GophermartConfig) *AccuralClientImpl {
	return &AccuralClientImpl{
		httpClient: http.DefaultClient,
		url:        cfg.AccuralAddress,
	}
}

type AccuralClientImpl struct {
	httpClient *http.Client
	url        string
}

func (ac *AccuralClientImpl) GetOrdersAccrual(orderId string) (*model.AccrualResponse, *int, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/orders/%s", ac.url, orderId), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &resp.StatusCode, nil
	}

	defer resp.Body.Close()
	var response model.AccrualResponse
	decoder := json.NewDecoder(resp.Body)
	if errDecode := decoder.Decode(&response); errDecode != nil {
		return nil, nil, errDecode
	}
	return &response, &resp.StatusCode, nil
}
