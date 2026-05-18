package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/testgen/transport/internal/config"
	"github.com/testgen/transport/internal/models"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		baseURL: cfg.AgentBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) CreateGenerationTask(ctx context.Context, req models.GenerationTaskRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	url := c.baseURL + "/v1/generation/tasks"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("agent returned status %d", resp.StatusCode)
	}
	log.Printf("agent task accepted for message %s", req.MessageID)
	return nil
}
