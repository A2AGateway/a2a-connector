package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	a2a "github.com/A2AGateway/a2a-protocol"
)

// Client handles registration and heartbeat with the A2A Gateway.
type Client struct {
	gatewayURL   string
	connectorID  string
	connectorURL string
	httpClient   *http.Client
}

// NewClient creates a new gateway client.
// gatewayURL is the base URL of the A2A Gateway (e.g. "http://gateway:8080").
// connectorURL is the public URL of this connector (included in the agent card).
func NewClient(gatewayURL, connectorID, connectorURL string) *Client {
	return &Client{
		gatewayURL:   gatewayURL,
		connectorID:  connectorID,
		connectorURL: connectorURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Register posts the agent card to the gateway's connector registration endpoint.
// Returns nil if the gateway doesn't yet implement the endpoint (404) so the
// connector can still run in standalone mode.
func (c *Client) Register(card *a2a.AgentCard) error {
	data, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal agent card: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/connectors/register", c.gatewayURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[gateway] Registration endpoint not found at %s — connector running in standalone mode", url)
		return nil
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gateway registration returned HTTP %d", resp.StatusCode)
	}

	return nil
}

// Heartbeat sends a keepalive ping to the gateway so it knows this connector
// is still online. Silently ignores 404 (gateway not yet implementing heartbeat).
func (c *Client) Heartbeat() error {
	url := fmt.Sprintf("%s/api/v1/connectors/%s/heartbeat", c.gatewayURL, c.connectorID)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("heartbeat returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// StartHeartbeat sends heartbeats on the given interval until ctx is cancelled.
func (c *Client) StartHeartbeat(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.Heartbeat(); err != nil {
					log.Printf("[gateway] heartbeat failed: %v", err)
				}
			}
		}
	}()
}
