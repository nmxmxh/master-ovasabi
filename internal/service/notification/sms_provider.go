// AzureSMSProvider requires the following environment variables:
//   AZURE_SMS_ENDPOINT, AZURE_SMS_ACCESS_KEY, AZURE_SMS_FROM
//
// Usage:
//   provider := NewAzureSMSProviderFromEnv()

package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

type SMSProvider interface {
	SendSMS(ctx context.Context, to, message string) error
}

type AzureSMSProvider struct {
	Endpoint  string
	AccessKey string
	From      string
}

type azureSMSRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Message string   `json:"message"`
}

func NewAzureSMSProviderFromEnv() *AzureSMSProvider {
	return &AzureSMSProvider{
		Endpoint:  os.Getenv("AZURE_SMS_ENDPOINT"),
		AccessKey: os.Getenv("AZURE_SMS_ACCESS_KEY"),
		From:      os.Getenv("AZURE_SMS_FROM"),
	}
}

func (a *AzureSMSProvider) SendSMS(ctx context.Context, to, message string) error {
	reqBody := azureSMSRequest{
		From:    a.From,
		To:      []string{to},
		Message: message,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	url := a.Endpoint + "/sms?api-version=2021-03-07"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.AccessKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New("azure sms send failed: " + resp.Status)
	}
	return nil
}
