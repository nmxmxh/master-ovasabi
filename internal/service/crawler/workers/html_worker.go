package workers

import (
	"bytes"
	"context"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"go.uber.org/zap"
	"golang.org/x/net/html/charset"
)

type HTMLWorker struct {
	BaseWorker
	collector *colly.Collector
}

func (w *HTMLWorker) WorkerType() crawlerpb.TaskType {
	return crawlerpb.TaskType_TASK_TYPE_HTML
}

func (w *HTMLWorker) Cleanup() {
	if w.collector != nil {
		// Reset the collector to nil to help with GC in long-lived workers
		w.collector = nil
	}
}

func (w *HTMLWorker) Process(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	// Use context for diagnostics/cancellation (lint fix)
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	result := &crawlerpb.CrawlResult{TaskUuid: task.Uuid}

	w.collector = colly.NewCollector(
		colly.Async(true),
		colly.DetectCharset(),
	)

	w.collector.OnResponse(func(r *colly.Response) {
		encoding, _, _ := charset.DetermineEncoding(r.Body, r.Headers.Get("Content-Type"))
		utf8Body, err := encoding.NewDecoder().Bytes(r.Body)
		if err != nil {
			w.Logger.Error("Failed to decode response body", zap.Error(err))
			return
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(utf8Body))
		if err != nil {
			w.Logger.Error("Failed to create goquery document", zap.Error(err))
			return
		}

		cleanContent := sanitizeHTML(doc)
		links := extractLinks(doc, r.Request.URL)

		result.ExtractedContent = []byte(cleanContent)
		result.ExtractedLinks = links
	})

	err := w.collector.Visit(task.Target)
	if err != nil {
		return nil, err
	}

	w.collector.Wait()

	return result, nil
}

// Remove ads, scripts, and boilerplate elements.
func sanitizeHTML(doc *goquery.Document) string {
	doc.Find("script, style, iframe, noscript").Remove()
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		_ = i // Use i to avoid revive unused-parameter warning
		if isAdElement(s) {
			s.Remove()
		}
	})
	return strings.TrimSpace(doc.Text())
}

// Detect if element is likely an ad.
func isAdElement(s *goquery.Selection) bool {
	class, _ := s.Attr("class")
	id, _ := s.Attr("id")
	if strings.Contains(class, "ad") || strings.Contains(id, "ad") {
		return true
	}
	return false
}

// Extract all hyperlinks, resolving relative URLs.
func extractLinks(doc *goquery.Document, base *url.URL) []string {
	var links []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		_ = i // Use i to avoid revive unused-parameter warning
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		u, err := url.Parse(href)
		if err != nil {
			return
		}
		abs := base.ResolveReference(u)
		links = append(links, abs.String())
	})
	return links
}
