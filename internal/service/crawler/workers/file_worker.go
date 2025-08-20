package workers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unidoc/unioffice/document"

	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"gopkg.in/neurosnap/sentences.v1"
)

type FileWorker struct {
	BaseWorker
}

func (w *FileWorker) WorkerType() crawlerpb.TaskType {
	return crawlerpb.TaskType_TASK_TYPE_FILE
}

func (w *FileWorker) Cleanup() {
	tmpDir := os.TempDir()
	candidates := []string{"in-", "out-pdf"}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		for _, prefix := range candidates {
			if strings.HasPrefix(entry.Name(), prefix) {
				fullPath := filepath.Join(tmpDir, entry.Name())
				_ = os.RemoveAll(fullPath)
			}
		}
	}
}

func (w *FileWorker) Process(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	// Reference unused ctx for diagnostics/cancellation
	if ctx != nil && ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	content, err := os.ReadFile(task.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Content refinement
	var cleanContent string
	switch strings.ToLower(filepath.Ext(task.Target)) {
	case ".pdf":
		cleanContent, err = extractPDFText(content)
		if err != nil {
			return nil, fmt.Errorf("pdf extraction failed: %w", err)
		}
	case ".docx":
		cleanContent, err = extractDOCXText(content)
		if err != nil {
			return nil, fmt.Errorf("docx extraction failed: %w", err)
		}
	default:
		cleanContent = string(content)
	}

	// Sentence boundary detection
	tokenizer := sentences.NewSentenceTokenizer(nil)
	tokens := tokenizer.Tokenize(cleanContent)

	lines := make([]string, len(tokens))
	for i, s := range tokens {
		lines[i] = s.Text
	}

	return &crawlerpb.CrawlResult{
		TaskUuid:         task.Uuid,
		ExtractedContent: []byte(strings.Join(lines, "\n")),
	}, nil
}

// PDF text extraction using pdfcpu.
func extractPDFText(content []byte) (string, error) {
	tmpIn, err := os.CreateTemp("", "in-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpIn.Name())
	defer tmpIn.Close()

	if _, err := tmpIn.Write(content); err != nil {
		return "", err
	}

	outDir, err := os.MkdirTemp("", "out-pdf")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(outDir)

	if err := pdfcpuapi.ExtractContentFile(
		tmpIn.Name(),
		outDir,
		nil,
		model.NewDefaultConfiguration(),
	); err != nil {
		return "", err
	}

	var b strings.Builder
	err = filepath.Walk(outDir, func(path string, _ os.FileInfo, _ error) error {
		if filepath.Ext(path) == ".txt" {
			if d, err := os.ReadFile(path); err == nil {
				b.Write(d)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

// extractDOCXText extracts text from a DOCX document.
func extractDOCXText(content []byte) (string, error) {
	r := bytes.NewReader(content)
	doc, err := document.Read(r, int64(len(content)))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, p := range doc.Paragraphs() {
		for _, run := range p.Runs() {
			sb.WriteString(run.Text())
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
