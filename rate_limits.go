package youscore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type RateLimitsResponse struct {
	DataAnalytics    *RateLimits `json:"dataAnalytics,omitempty"`
	PDFLegalEntities *RateLimits `json:"PDFLegalEntities,omitempty"`
	PDFIndividuals   *RateLimits `json:"PDFIndividuals,omitempty"`
	Affiliates       *RateLimits `json:"affiliates,omitempty"`
}

type RateLimits struct {
	ActualDate    time.Time `json:"actualDate"`
	RequestsCount []struct {
		Api      string `json:"api"`
		Count    int    `json:"count"`
		Endpoint string `json:"endpoint"`
	} `json:"requestsCount"`
	RequestsLeft int `json:"requestsLeft"`
	TotalLimits  int `json:"totalLimits"`
}

// CheckRateLimits for all (non-blank) keys
func CheckRateLimits(ctx context.Context, keys APIKeys) (*RateLimitsResponse, error) {

	getLimit := func(ctx context.Context, key string) (*RateLimits, error) {
		if key == "" {
			return nil, nil
		}

		// the rate limit endpoint defaults to the fallback API key, which is the DataAnalytics key
		// so we create a new client that sets it
		cl, err := NewClientWithResponses(ServerURL,
			WithAPIKeys(APIKeys{
				DataAnalytics: key,
			}),
		)
		if err != nil {
			return nil, fmt.Errorf("new client: %w", err)
		}
		res, err := cl.GetV1RateLimitsWithResponse(ctx)
		if err != nil {
			return nil, err
		}
		if res.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf("bad status: %d", res.StatusCode())
		}

		var limits RateLimits
		if err = json.Unmarshal(res.Body, &limits); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		return &limits, nil
	}

	dataLimits, err := getLimit(ctx, keys.DataAnalytics)
	if err != nil {
		return nil, fmt.Errorf("get data analytics key limit: %w", err)
	}
	pdfEntitiesLimits, err := getLimit(ctx, keys.PDFLegalEntities)
	if err != nil {
		return nil, fmt.Errorf("get pdf legal entities key limit: %w", err)
	}
	pdfIndividualsLimits, err := getLimit(ctx, keys.PDFIndividuals)
	if err != nil {
		return nil, fmt.Errorf("get pdf individuals key limit: %w", err)
	}
	affiliatesLimits, err := getLimit(ctx, keys.Affiliates)
	if err != nil {
		return nil, fmt.Errorf("get affiliates key limit: %w", err)
	}

	return &RateLimitsResponse{
		DataAnalytics:    dataLimits,
		PDFLegalEntities: pdfEntitiesLimits,
		PDFIndividuals:   pdfIndividualsLimits,
		Affiliates:       affiliatesLimits,
	}, nil
}
