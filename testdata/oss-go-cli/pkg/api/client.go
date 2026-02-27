// Package api provides an HTTP client for the gosync remote API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client communicates with the gosync remote synchronization server.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// SyncStatus represents the status of a remote sync operation.
type SyncStatus struct {
	ID        string    `json:"id"`
	State     string    `json:"state"`
	Files     int       `json:"files"`
	Progress  float64   `json:"progress"`
	StartedAt time.Time `json:"started_at"`
	Error     string    `json:"error,omitempty"`
}

// NewClient creates a new API client with the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetStatus retrieves the sync status for the given operation ID.
func (c *Client) GetStatus(ctx context.Context, id string) (*SyncStatus, error) {
	url := fmt.Sprintf("%s/api/v1/sync/%s/status", c.baseURL, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	var status SyncStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &status, nil
}