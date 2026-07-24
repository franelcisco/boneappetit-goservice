package foodapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Client struct {
	baseURL string
	http    *http.Client
	logger  *zap.Logger
}

type RecurringRequest struct {
	DraftOrderID string `json:"draftOrderId"`
}

type RecurringResponse struct {
	Success   bool `json:"success"`
	Affiliate bool `json:"affiliate"`
}

func NewClient(baseURL string, timeout time.Duration, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
		logger:  logger,
	}
}

// DirectDebitAccountRecurring POSTs to /payments/direct-debit-account/recurring
// on the configured food-api-dev base URL.
func (c *Client) DirectDebitAccountRecurring(
	ctx context.Context, req RecurringRequest,
) (*RecurringResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/payments/direct-debit-account/recurring"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		c.logger.Error("food-api recurring charge returned non-2xx",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return nil, fmt.Errorf("food-api recurring charge: status %d: %s", resp.StatusCode, string(respBody))
	}

	var out RecurringResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &out, nil
}
