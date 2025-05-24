// AzurePushProvider requires the following environment variables:
//   AZURE_PUSH_HUB_ENDPOINT, AZURE_PUSH_SAS
//
// Usage:
//   provider := NewAzurePushProviderFromEnv()

package notification

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
)

type PushProvider interface {
	SendPush(ctx context.Context, deviceToken, payload string) error
}

type AzurePushProvider struct {
	HubEndpoint string
	SAS         string
}

func NewAzurePushProviderFromEnv() *AzurePushProvider {
	return &AzurePushProvider{
		HubEndpoint: os.Getenv("AZURE_PUSH_HUB_ENDPOINT"),
		SAS:         os.Getenv("AZURE_PUSH_SAS"),
	}
}

func (a *AzurePushProvider) SendPush(ctx context.Context, deviceToken, payload string) error {
	url := fmt.Sprintf("%s/?api-version=2015-01", a.HubEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", a.SAS)
	req.Header.Set("ServiceBusNotification-Format", "gcm") // or "apple" for APNs
	req.Header.Set("ServiceBusNotification-DeviceHandle", deviceToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("azure push send failed: %s", resp.Status)
	}
	return nil
}
