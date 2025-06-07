package localization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LibreTranslateConfig holds configuration for the LibreTranslate API.
// This is now fully dynamic and driven by env/config struct (see config.yaml and internal/config/config.go).
type LibreTranslateConfig struct {
	Endpoint string // e.g., "http://localhost:5002" (default for local Docker)
	APIKey   string // optional
	Timeout  time.Duration
}

// BatchTranslateLibre calls the LibreTranslate API to translate a batch of texts.
// Uses the config from env/config struct for endpoint and timeout.
// Returns a map of input keys to translated values, and a slice of keys that failed to translate.
func BatchTranslateLibre(ctx context.Context, cfg LibreTranslateConfig, texts map[string]string, sourceLang, targetLang string) (translations map[string]string, failed []string, err error) {
	if len(texts) == 0 {
		return map[string]string{}, nil, nil
	}
	url := fmt.Sprintf("%s/translate", cfg.Endpoint)
	inputs := make([]string, 0, len(texts))
	keyOrder := make([]string, 0, len(texts))
	for k, v := range texts {
		inputs = append(inputs, v)
		keyOrder = append(keyOrder, k)
	}
	payload := map[string]interface{}{
		"q":      inputs,
		"source": sourceLang,
		"target": targetLang,
		"format": "text",
	}
	if cfg.APIKey != "" {
		payload["api_key"] = cfg.APIKey
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: cfg.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, keyOrder, fmt.Errorf("failed to call LibreTranslate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, keyOrder, fmt.Errorf("failed to read LibreTranslate error body: %w", err)
		}
		return nil, keyOrder, fmt.Errorf("LibreTranslate error: %s", string(b))
	}
	var result []struct {
		TranslatedText string `json:"translated_text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, keyOrder, fmt.Errorf("failed to decode response: %w", err)
	}
	translations = make(map[string]string)
	failed = make([]string, 0)
	for i, k := range keyOrder {
		if i < len(result) && result[i].TranslatedText != "" {
			translations[k] = result[i].TranslatedText
		} else {
			failed = append(failed, k)
		}
	}
	return translations, failed, nil
}
