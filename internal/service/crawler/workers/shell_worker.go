//go:build linux
// +build linux

package workers

import (
	"context"
	"fmt"
	"path/filepath"

	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
)

type ShellWorker struct {
	BaseWorker
}

func (w *ShellWorker) WorkerType() crawlerpb.TaskType {
	return crawlerpb.TaskType_TASK_TYPE_SHELL
}

func (w *ShellWorker) Cleanup() {
	// Optionally clean up VM instances, temp files, or stale sockets
	if err := cleanupFirecrackerArtifacts(); err != nil {
		fmt.Printf("shell worker cleanup warning: %v\n", err)
	}
}

// Process runs the shell command task inside a Firecracker VM.
func (w *ShellWorker) Process(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	// Command allowlist for security
	if !isAllowedCommand(task.Target) {
		return nil, fmt.Errorf("command not allowed: %s", task.Target)
	}

	// Run within Firecracker VM
	result, err := runInFirecracker(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("firecracker execution failed: %w", err)
	}

	return result, nil
}

// Very basic pattern matcher for allowed commands
func isAllowedCommand(cmd string) bool {
	allowed := []string{
		"file *", "pdfinfo *", "exiftool *",
		"ffmpeg -i *", "ffprobe *",
	}

	for _, pattern := range allowed {
		if matched, _ := filepath.Match(pattern, cmd); matched {
			return true
		}
	}
	return false
}
