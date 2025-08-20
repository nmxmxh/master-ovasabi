package workers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
)

type ArchiveWorker struct {
	BaseWorker
}

func (w *ArchiveWorker) WorkerType() crawlerpb.TaskType {
	return crawlerpb.TaskType_TASK_TYPE_ARCHIVE
}

func (w *ArchiveWorker) Cleanup() {
	tempDirPrefix := "extract-"

	// Look through the system temp directory
	tmpRoot := os.TempDir()

	entries, err := os.ReadDir(tmpRoot)
	if err != nil {
		zap.L().Sugar().Warnf("Cleanup failed to read temp dir: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), tempDirPrefix) {
			fullPath := filepath.Join(tmpRoot, entry.Name())
			if err := os.RemoveAll(fullPath); err != nil {
				zap.L().Sugar().Warnf("Failed to clean temp dir %s: %v", fullPath, err)
			} else {
				zap.L().Sugar().Infof("Cleaned up temp dir: %s", fullPath)
			}
		}
	}
}

func (w *ArchiveWorker) Process(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	result := &crawlerpb.CrawlResult{TaskUuid: task.Uuid}

	tmpDir, err := os.MkdirTemp("", "extract-"+task.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			zap.L().Sugar().Warnf("failed to clean temp dir %s: %v", tmpDir, err)
		}
	}()

	if task.Depth <= 0 {
		return nil, fmt.Errorf("maximum recursion depth exceeded")
	}

	file, err := os.Open(task.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	format, input, err := archiver.Identify(ctx, file.Name(), file)
	if errors.Is(err, archiver.NoMatch) {
		return nil, fmt.Errorf("unrecognized archive format for %s: %w", filepath.Base(task.Target), err)
	} else if err != nil {
		return nil, fmt.Errorf("archive identification failed: %w", err)
	}

	extractor, ok := format.(archiver.Extractor)
	if !ok {
		return nil, ErrUnsupportedArchive
	}

	fileCount := 0
	const maxFiles = 1000

	err = extractor.Extract(ctx, input, func(ctx context.Context, f archiver.FileInfo) error {
		fileCount++
		if fileCount > maxFiles {
			return fmt.Errorf("archive exceeds maximum file count (%d)", maxFiles)
		}

		if isDangerousPath(f.Name()) {
			zap.L().Sugar().Debugf("Skipping potentially dangerous file: %s", f.Name())
			return nil
		}

		if !strings.HasSuffix(task.Uuid, "-nested") && w.contentSize > 0 && f.Size() > w.contentSize {
			zap.L().Sugar().Debugf("Skipping large file (%d bytes) in archive %s: %s", f.Size(), filepath.Base(task.Target), f.Name())
			return nil
		}

		outputPath := filepath.Join(tmpDir, f.NameInArchive)
		if f.IsDir() {
			return os.MkdirAll(outputPath, f.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return err
		}

		outFile, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		if _, err := io.Copy(outFile, rc); err != nil {
			return err
		}

		if isArchive(ctx, outputPath) && task.Depth > 1 {
			nestedTask := &crawlerpb.CrawlTask{
				Uuid:   task.Uuid + "-nested",
				Type:   task.Type,
				Target: outputPath,
				Depth:  task.Depth - 1,
			}
			nestedResult, err := w.Process(ctx, nestedTask)
			if err != nil {
				zap.L().Sugar().Warnf("Nested archive processing failed: %v", err)
			} else if nestedResult != nil {
				result.ExtractedLinks = append(result.ExtractedLinks, nestedResult.ExtractedLinks...)
			}
		} else {
			result.ExtractedLinks = append(result.ExtractedLinks, outputPath)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	return result, nil
}

func isDangerousPath(path string) bool {
	dangerousPatterns := []string{
		"__MACOSX", ".DS_Store",
		"*.exe", "*.dll", "*.bat", "*.cmd",
		"/etc/passwd", "/etc/shadow",
	}
	for _, pattern := range dangerousPatterns {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			// Log or handle the error as needed
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// isArchive checks if a given file path points to a known archive format.
func isArchive(ctx context.Context, filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		// If we can't open it, we can't identify it.
		return false
	}
	defer file.Close()

	_, _, err = archiver.Identify(ctx, filePath, file)
	if errors.Is(err, archiver.NoMatch) {
		return false
	}
	return err == nil
}
