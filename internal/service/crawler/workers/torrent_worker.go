package workers

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/anacrolix/torrent"
	"github.com/mholt/archiver/v4"
	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"go.uber.org/zap"
)

type TorrentWorker struct {
	BaseWorker
	client *torrent.Client
}

func (w *TorrentWorker) WorkerType() crawlerpb.TaskType {
	return crawlerpb.TaskType_TASK_TYPE_TORRENT
}

func (w *TorrentWorker) Cleanup() {
	if w.client != nil {
		zap.L().Info("Shutting down torrent client")
		err := w.client.Close()
		if err != nil {
			zap.L().Warn("Error closing torrent client")
		}
	}
}

func (w *TorrentWorker) Process(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	t, err := w.client.AddMagnet(task.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to add magnet: %w", err)
	}
	<-t.GotInfo()

	result := &crawlerpb.CrawlResult{TaskUuid: task.Uuid}
	security := NewSecurityWorker()

	for _, f := range t.Files() {
		filePath := f.Path()
		// Download file to memory
		reader := f.NewReader()
		func() {
			defer reader.Close()
			// Extract archive content if applicable
			if isArchive(ctx, filePath) {
				buf := new(bytes.Buffer)
				_, err := io.Copy(buf, reader)
				if err != nil {
					zap.L().Sugar().Warnf("Error reading archive %s: %v", filePath, err)
					return
				}
				if err := extractAndSanitizeArchive(ctx, buf.Bytes(), security, result); err != nil {
					zap.L().Sugar().Warnf("Error extracting archive %s: %v", filePath, err)
				}
			} else {
				// Read non-archive content and sanitize
				buf := new(bytes.Buffer)
				_, err := io.Copy(buf, reader)
				if err != nil {
					zap.L().Sugar().Warnf("Error reading file %s: %v", filePath, err)
					return
				}
				cleaned, err := security.SanitizeContent(ctx, buf.Bytes())
				if err == nil {
					result.ExtractedContent = append(result.ExtractedContent, cleaned...)
				}
			}
		}()
	}

	return result, nil
}

// Extracts from archive bytes in memory and sanitizes each file.
func extractAndSanitizeArchive(ctx context.Context, archiveData []byte, security *SecurityWorker, result *crawlerpb.CrawlResult) error {
	format, input, err := archiver.Identify(ctx, "", bytes.NewReader(archiveData))
	if err != nil {
		return fmt.Errorf("archive identification failed: %w", err)
	}

	extractor, ok := format.(archiver.Extractor)
	if !ok {
		return fmt.Errorf("unsupported archive format")
	}

	return extractor.Extract(ctx, input, func(ctx context.Context, f archiver.FileInfo) error {
		if f.IsDir() {
			return nil
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		rawContent, err := io.ReadAll(rc)
		if err != nil {
			return err
		}

		sanitized, err := security.SanitizeContent(ctx, rawContent)
		if err != nil {
			return nil // skip bad content silently
		}

		result.ExtractedContent = append(result.ExtractedContent, sanitized...)
		result.ExtractedLinks = append(result.ExtractedLinks, fmt.Sprintf("archive-entry:%s", f.NameInArchive))
		return nil
	})
}
