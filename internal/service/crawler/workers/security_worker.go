package workers

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"regexp"

	"github.com/dutchcoders/go-clamd"
)

type SecurityWorker struct {
	clamav *clamd.Clamd
}

func NewSecurityWorker() *SecurityWorker {
	return &SecurityWorker{
		clamav: clamd.NewClamd("tcp://localhost:3310"),
	}
}

func (w *SecurityWorker) SanitizeContent(ctx context.Context, content []byte) ([]byte, error) {
	// 1. Malware Detection
	if w.clamav != nil {
		abortChan := make(chan bool) // or use `nil` if you don't plan to abort

		response, err := w.clamav.ScanStream(bytes.NewReader(content), abortChan)
		if err != nil {
			return nil, fmt.Errorf("clamav scan error: %w", err)
		}

		for res := range response {
			if res.Status == clamd.RES_FOUND {
				return nil, fmt.Errorf("malware detected: %s", res.Description)
			}
		}
	}

	// 2. PII Redaction
	redacted := redactPII(content)

	// 3. Decompression Bomb Check
	if isDecompressionBomb(redacted) {
		return nil, fmt.Errorf("content flagged as decompression bomb")
	}

	// 4. High Entropy Detection (e.g., encrypted/obfuscated content)
	if isHighEntropy(redacted) {
		return nil, fmt.Errorf("content appears encrypted or suspicious (high entropy)")
	}

	return redacted, nil
}

func redactPII(data []byte) []byte {
	patterns := map[string]string{
		`\b\d{3}-\d{2}-\d{4}\b`:                                  "REDACTED-SSN",   // SSN
		`\b\d{4}-\d{4}-\d{4}-\d{4}\b`:                            "REDACTED-CC",    // Credit Card
		`(?i)\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`: "REDACTED-EMAIL", // Email
		`\b(\+\d{1,2}\s?)?\(?\d{3}\)?[\s.-]?\d{3}[\s.-]?\d{4}\b`: "REDACTED-PHONE", // Phone
	}

	for pattern, replacement := range patterns {
		re := regexp.MustCompile(pattern)
		data = re.ReplaceAll(data, []byte(replacement))
	}
	return data
}

func isDecompressionBomb(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Check if the content is composed of a single byte repeated
	first := data[0]
	for _, b := range data {
		if b != first {
			return false
		}
	}
	return len(data) > 1<<20 // >1MB of repeated byte = potential bomb
}

func calculateShannonEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var freq [256]int
	for _, b := range data {
		freq[b]++
	}

	var entropy float64
	dataLen := float64(len(data))
	for _, count := range freq {
		if count > 0 {
			p := float64(count) / dataLen
			entropy -= p * math.Log2(p)
		}
	}

	return entropy / 8.0 // Normalize to 0.0–1.0
}

func isHighEntropy(data []byte) bool {
	const threshold = 0.85
	return calculateShannonEntropy(data) > threshold
}
