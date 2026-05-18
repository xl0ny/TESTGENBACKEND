package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/testgen/app/internal/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	url := strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: url,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) Send(ctx context.Context, payload map[string]any) (domain.SendAccepted, error) {
	var accepted domain.SendAccepted
	if err := c.postJSON(ctx, c.baseURL+"/v1/messages/send", payload, &accepted); err != nil {
		return accepted, err
	}
	return accepted, nil
}

func (c *Client) Receive(ctx context.Context, query domain.ReceiveQuery) (domain.ReceiveResult, error) {
	var result domain.ReceiveResult
	if err := c.postJSON(ctx, c.baseURL+"/v1/messages/receive", query, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (c *Client) Health(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false, err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK, nil
}

func (c *Client) postJSON(ctx context.Context, url string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var errBody struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &errBody)
		msg := errBody.Message
		if msg == "" {
			msg = fmt.Sprintf("transport HTTP %d", res.StatusCode)
		}
		return fmt.Errorf("%s", msg)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}
