package adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
)

type HTTPAdapter struct {
	client *http.Client
}

func NewHTTPAdapter() *HTTPAdapter {
	return &HTTPAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *HTTPAdapter) Protocol() string                                        { return "http" }
func (a *HTTPAdapter) Capabilities() []string                                  { return []string{"get", "post", "put", "delete"} }
func (a *HTTPAdapter) Endpoint() string                                        { return "dynamic" }
func (a *HTTPAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error { return nil }
func (a *HTTPAdapter) Send(ctx context.Context, msg *bridge.Message) error {
	method := msg.Metadata["http_method"]
	url := msg.Metadata["http_url"]
	if method == "" || url == "" {
		return fmt.Errorf("missing http_method or http_url in message metadata")
	}
	// Support custom headers from metadata (prefix: http_header_)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(msg.Payload))
	if err != nil {
		return err
	}
	for k, v := range msg.Metadata {
		if len(k) > 12 && k[:12] == "http_header_" {
			req.Header.Set(k[12:], v)
		}
	}
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("[HTTPAdapter] Request error: %v\n", err)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[HTTPAdapter] ReadAll error: %v\n", err)
	}
	fmt.Printf("[HTTPAdapter] %s %s -> %d\n", method, url, resp.StatusCode)
	// Optionally log response body for debugging (caution: may contain sensitive data)
	// fmt.Printf("[HTTPAdapter] Response body: %s\n", string(body))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, string(body))
	}
	return nil
}
func (a *HTTPAdapter) Receive(_ context.Context, _ bridge.MessageHandler) error { return nil }
func (a *HTTPAdapter) HealthCheck() bridge.HealthStatus {
	return bridge.HealthStatus{Status: "UP", Timestamp: time.Now()}
}

func (a *HTTPAdapter) Close() error {
	// No persistent resources to clean up
	return nil
}

func init() {
	bridge.RegisterAdapter(NewHTTPAdapter())
}
