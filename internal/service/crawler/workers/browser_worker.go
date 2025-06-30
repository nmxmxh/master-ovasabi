package workers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"go.uber.org/zap"
)

type BrowserEmulatorWorker struct {
	BaseWorker
}

func (w *BrowserEmulatorWorker) WorkerType() crawlerpb.TaskType {
	return crawlerpb.TaskType_TASK_TYPE_BROWSER
}

func (w *BrowserEmulatorWorker) Cleanup() {
	tempDirPrefix := "browser-"
	tmpRoot := os.TempDir()

	entries, err := os.ReadDir(tmpRoot)
	if err != nil {
		zap.L().Sugar().Warnf("Browser cleanup: failed to read temp dir: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), tempDirPrefix) {
			fullPath := filepath.Join(tmpRoot, entry.Name())

			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			// Only clean up if the folder is older than 10 minutes
			if time.Since(info.ModTime()) > 10*time.Minute {
				if err := os.RemoveAll(fullPath); err != nil {
					zap.L().Sugar().Warnf("Browser cleanup: failed to remove %s: %v", fullPath, err)
				} else {
					zap.L().Sugar().Infof("Browser cleanup: removed temp dir %s", fullPath)
				}
			}
		}
	}
}

func (w *BrowserEmulatorWorker) Process(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	// Sandboxed browser profile
	userDir := filepath.Join(os.TempDir(), "browser-"+task.Uuid)
	defer os.RemoveAll(userDir)

	// Launch with security flags
	launch := launcher.New().
		Headless(true).
		UserDataDir(userDir).
		Set("disable-gpu").
		Set("no-sandbox").
		Set("disable-setuid-sandbox").
		Set("disable-dev-shm-usage")

	url := launch.MustLaunch()
	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage()
	defer page.MustClose()

	// Security: Block ads, trackers, and remote fonts
	router := page.HijackRequests()
	defer router.MustStop()

	router.MustAdd("*.{png,jpg,jpeg,gif,webp}", blockResource)
	router.MustAdd("*.{css,woff,woff2,ttf}", blockResource)
	router.MustAdd("*ads*", blockResource)
	router.MustAdd("*track*", blockResource)

	go router.Run()

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := page.Context(ctx).Navigate(task.Target); err != nil {
		return nil, err
	}

	// Wait for network idle
	page.MustWaitIdle()

	// Extract clean content
	content, err := page.HTML()
	if err != nil {
		return nil, err
	}

	// Extract links
	links := page.MustElements("a")
	extractedLinks := make([]string, 0, len(links))
	for _, link := range links {
		if href := link.MustAttribute("href"); href != nil {
			extractedLinks = append(extractedLinks, *href)
		}
	}

	return &crawlerpb.CrawlResult{
		TaskUuid:         task.Uuid,
		ExtractedContent: []byte(content),
		ExtractedLinks:   extractedLinks,
	}, nil
}

func blockResource(ctx *rod.Hijack) {
	urlStr := ctx.Request.URL().String()
	if ctx.Request.Type() == proto.NetworkResourceTypeImage ||
		ctx.Request.Type() == proto.NetworkResourceTypeFont ||
		strings.Contains(urlStr, "ads") ||
		strings.Contains(urlStr, "track") {
		ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
		return
	}
	ctx.ContinueRequest(&proto.FetchContinueRequest{})
}
