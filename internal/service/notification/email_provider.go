package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

type EmailProvider interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

type AzureEmailProvider struct {
	Endpoint  string
	AccessKey string
	From      string
}

type azureEmailRecipient struct {
	Address string `json:"address"`
}

type azureEmailRequest struct {
	SenderAddress string `json:"sender_address"`
	Recipients    struct {
		To []azureEmailRecipient `json:"to"`
	} `json:"recipients"`
	Content struct {
		Subject   string `json:"subject"`
		PlainText string `json:"plain_text"`
	} `json:"content"`
}

func (a *AzureEmailProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	reqBody := azureEmailRequest{
		SenderAddress: a.From,
	}
	reqBody.Recipients.To = append(reqBody.Recipients.To, azureEmailRecipient{Address: to})
	reqBody.Content.Subject = subject
	reqBody.Content.PlainText = body

	b, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := a.Endpoint + "/emails:send?api-version=2023-03-31"
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
		return errors.New("azure email send failed: " + resp.Status)
	}
	return nil
}

// AzureEmailProvider requires the following environment variables:
//   AZURE_EMAIL_ENDPOINT, AZURE_EMAIL_ACCESS_KEY, AZURE_EMAIL_FROM
//
// Usage:
//   provider := NewAzureEmailProviderFromEnv()

func NewAzureEmailProviderFromEnv() *AzureEmailProvider {
	return &AzureEmailProvider{
		Endpoint:  os.Getenv("AZURE_EMAIL_ENDPOINT"),
		AccessKey: os.Getenv("AZURE_EMAIL_ACCESS_KEY"),
		From:      os.Getenv("AZURE_EMAIL_FROM"),
	}
}
