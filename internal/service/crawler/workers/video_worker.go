package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/asticode/go-astisub"
	"go.uber.org/zap"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type VideoWorker struct {
	BaseWorker
}

func (w *VideoWorker) WorkerType() crawlerpb.TaskType {
	return crawlerpb.TaskType_TASK_TYPE_VIDEO
}

func (w *VideoWorker) Cleanup() {
	tmpDir := os.TempDir()
	pattern := "-audio.wav"

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		zap.L().Warn("Failed to read temp directory for cleanup", zap.Error(err))
		return
	}

	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), pattern) {
			path := filepath.Join(tmpDir, entry.Name())
			if err := os.Remove(path); err != nil {
				zap.L().Warn("Failed to delete audio artifact", zap.String("path", path), zap.Error(err))
			} else {
				zap.L().Debug("Deleted audio artifact", zap.String("path", path))
			}
		}
	}
}

func (w *VideoWorker) Process(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	// Use context for diagnostics/cancellation (lint fix)
	logger := zap.L().Sugar()
	if ctx != nil && ctx.Err() != nil {
		logger.Warnf("Process cancelled by context: %v", ctx.Err())
		return nil, ctx.Err()
	}
	audioPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s-audio.wav", task.Uuid))

	// Step 1: Extract audio using ffmpeg
	if err := extractAudio(ctx, task.Target, audioPath); err != nil {
		logger.Errorf("audio extraction failed: %v", err)
		return nil, err
	}

	// Step 2: Extract subtitles (if available)
	var subtitleText string
	if subs, err := astisub.OpenFile(task.Target); err == nil {
		var lines []string
		for _, item := range subs.Items {
			for _, line := range item.Lines {
				lines = append(lines, line.String())
			}
		}
		subtitleText = strings.Join(lines, "\n")
	} else {
		logger.Warnf("no subtitles found or failed to parse: %v", err)
	}

	// Step 3: Extract video metadata
	metadata, err := extractVideoMetadata(ctx, task.Target)
	if err != nil {
		logger.Warnf("failed to extract video metadata: %v", err)
	}

	videoMetaMap := map[string]interface{}{
		"audio_path":  audioPath,
		"duration":    metadata.Duration,
		"format":      metadata.FormatName,
		"video_codec": metadata.VideoCodec,
		"width":       metadata.Width,
		"height":      metadata.Height,
	}
	videoMetaStruct, err := structpb.NewStruct(videoMetaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata struct: %w", err)
	}

	// Step 4: Build CrawlResult
	return &crawlerpb.CrawlResult{
		TaskUuid:         task.Uuid,
		ExtractedContent: []byte(subtitleText),
		Metadata: &commonpb.Metadata{
			ServiceSpecific: videoMetaStruct,
		},
	}, nil
}

func extractAudio(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputPath, "-vn", "-acodec", "pcm_s16le", "-ar", "44100", "-ac", "2", outputPath, "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type VideoMetadata struct {
	Duration   string
	FormatName string
	VideoCodec string
	Width      int
	Height     int
}

func extractVideoMetadata(ctx context.Context, path string) (VideoMetadata, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name,width,height",
		"-show_entries", "format=duration,format_name",
		"-of", "json",
		path,
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return VideoMetadata{}, err
	}

	var parsed struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration   string `json:"duration"`
			FormatName string `json:"format_name"`
		} `json:"format"`
	}

	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		return VideoMetadata{}, err
	}

	meta := VideoMetadata{
		Duration:   parsed.Format.Duration,
		FormatName: parsed.Format.FormatName,
	}

	if len(parsed.Streams) > 0 {
		meta.VideoCodec = parsed.Streams[0].CodecName
		meta.Width = parsed.Streams[0].Width
		meta.Height = parsed.Streams[0].Height
	}

	return meta, nil
}
