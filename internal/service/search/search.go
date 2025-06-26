package search

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	"github.com/nmxmxh/master-ovasabi/internal/ai"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// EventEnvelope is the canonical, extensible wrapper for all event-driven messages in the system.
type EventEnvelope struct {
	Type          string                 `json:"type"`
	Version       string                 `json:"version,omitempty"`
	Schema        string                 `json:"schema,omitempty"`
	Payload       interface{}            `json:"payload"`  // For core flows, use canonical proto messages (e.g., *searchpb.SearchRequest, *searchpb.SearchResponse)
	Metadata      *commonpb.Metadata     `json:"metadata"` // Always use canonical commonpb.Metadata
	PrevState     interface{}            `json:"prev_state,omitempty"`
	NextState     interface{}            `json:"next_state,omitempty"`
	Patch         interface{}            `json:"patch,omitempty"`
	Intent        string                 `json:"intent,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	CausationID   string                 `json:"causation_id,omitempty"`
	ParentID      string                 `json:"parent_id,omitempty"`
	Timestamp     int64                  `json:"timestamp,omitempty"`
	Timeline      []interface{}          `json:"timeline,omitempty"`
	Signature     string                 `json:"signature,omitempty"`
	Auth          string                 `json:"auth,omitempty"`
	Provenance    string                 `json:"provenance,omitempty"`
	Score         float64                `json:"score,omitempty"`
	Feedback      interface{}            `json:"feedback,omitempty"`
	Explanation   string                 `json:"explanation,omitempty"`
	Extensions    map[string]interface{} `json:"extensions,omitempty"`
}

// SearchResult is the canonical struct for all search results.
type Result struct {
	ID       string
	Type     string
	Score    float64
	Fields   map[string]interface{}
	Source   string // e.g., "internal", "google"
	Metadata *commonpb.Metadata
}

// Adapter defines the interface for search adapters.
type Adapter interface {
	Search(ctx context.Context, req *Request) ([]*Result, error)
	Name() string
}

// Request is the canonical struct for all search queries.
type Request struct {
	Query    string
	Types    []string
	Sources  []string // e.g., ["internal", "google", "wikipedia"]
	Metadata *commonpb.Metadata
}

type Service struct {
	searchpb.UnimplementedSearchServiceServer
	log          *zap.Logger
	repo         *Repository
	Cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	provider     *service.Provider // Canonical provider for DI/event orchestration
	adapters     map[string]Adapter
}

// NewService creates a new SearchService instance with event bus and provider support (canonical pattern).
func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool, provider *service.Provider) searchpb.SearchServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		provider:     provider,
		adapters:     make(map[string]Adapter),
	}
}

// Event-Driven Search Orchestration Pattern (2025)
// ------------------------------------------------
// This file implements the canonical event-driven search pattern for the OVASABI platform.
// All search actions can be triggered by events (search.requested) and emit results (search.completed)
// via the Nexus event bus, using the canonical Payload and Metadata patterns.
// For more, see docs/amadeus/amadeus_context.md (Event-Driven Orchestration Standard).
//
// All orchestration, caching, and audit flows use the graceful orchestration config.
//
// This pattern is additive: gRPC/REST APIs remain fully supported.
// ------------------------------------------------

// HandleSearchRequestedEvent processes a search.requested event and emits search.completed.
func (s *Service) HandleSearchRequestedEvent(ctx context.Context, event *nexusv1.EventResponse) {
	// Unmarshal event payload to SearchRequest
	if event == nil || event.Payload == nil {
		s.log.Warn("search.requested event missing payload")
		return
	}
	var req searchpb.SearchRequest
	if event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
			if err != nil {
				s.log.Warn("failed to unmarshal search request from event payload", zap.Error(err))
				return
			}
		}
	}
	// Reuse core search logic (do not emit duplicate events)
	query := req.GetQuery()
	page := int(req.GetPageNumber())
	pageSize := int(req.GetPageSize())
	meta := req.GetMetadata()
	types := req.GetTypes()
	if len(types) == 0 {
		types = []string{"content"}
	}
	results, total, err := s.repo.SearchAllEntities(ctx, types, query, meta, req.GetCampaignId(), page, pageSize)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "event-driven search failed", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return
	}
	protos := make([]*searchpb.SearchResult, 0, len(results))
	for _, r := range results {
		protos = append(protos, &searchpb.SearchResult{
			Id:         r.ID,
			EntityType: r.EntityType,
			Score:      float32(r.Score),
			Metadata:   r.Metadata,
		})
	}
	resp := &searchpb.SearchResponse{
		Results:    protos,
		Total:      utils.ToInt32(total),
		PageNumber: utils.ToInt32(page),
		PageSize:   utils.ToInt32(pageSize),
	}
	// Marshal response as Payload
	b, err := protojson.Marshal(resp)
	if err != nil {
		s.log.Error("failed to marshal search response", zap.Error(err))
		return
	}
	respStruct, err := structpb.NewStruct(map[string]interface{}{})
	if err == nil {
		if err := protojson.Unmarshal(b, respStruct); err != nil {
			s.log.Error("failed to unmarshal search response", zap.Error(err))
			return
		}
	}
	// Emit search.completed event
	if s.eventEmitter != nil && s.eventEnabled {
		_, _ = s.eventEmitter.EmitEventWithLogging(ctx, s, s.log, "search.completed", query, meta)
	}
	// Orchestrate success
	success := graceful.WrapSuccess(ctx, codes.OK, "event-driven search completed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     "search:" + query,
		CacheValue:   resp,
		CacheTTL:     5 * time.Minute,
		Metadata:     meta,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "search.completed",
		EventID:      query,
		PatternType:  "search",
		PatternID:    query,
		PatternMeta:  meta,
	})
}

// Search implements robust multi-entity, FTS, and metadata filtering search.
// Supports searching across multiple entity types as specified in req.Types.
func (s *Service) WithinSearch(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	// Validate request
	if err := s.validateSearchRequest(ctx, req); err != nil {
		return nil, err
	}

	// Emit search.requested event for audit/traceability
	if s.eventEmitter != nil && s.eventEnabled {
		_, _ = s.eventEmitter.EmitEventWithLogging(ctx, s, s.log, "search.requested", req.GetQuery(), req.GetMetadata())
	}

	query := req.GetQuery()
	page := int(req.GetPageNumber())
	pageSize := int(req.GetPageSize())
	meta := req.GetMetadata()
	types := req.GetTypes()
	if len(types) == 0 {
		types = []string{"content"} // default to content if not specified
	}

	results, total, err := s.repo.SearchAllEntities(ctx, types, query, meta, req.GetCampaignId(), page, pageSize)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "search failed", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}

	protos := make([]*searchpb.SearchResult, 0, len(results))
	for _, r := range results {
		protos = append(protos, &searchpb.SearchResult{
			Id:         r.ID,
			EntityType: r.EntityType,
			Score:      float32(r.Score),
			Metadata:   r.Metadata,
		})
	}

	resp := &searchpb.SearchResponse{
		Results:    protos,
		Total:      utils.ToInt32(total),
		PageNumber: utils.ToInt32(page),
		PageSize:   utils.ToInt32(pageSize),
	}

	// After search, emit search.completed event for real-time updates
	if s.eventEmitter != nil && s.eventEnabled {
		_, _ = s.eventEmitter.EmitEventWithLogging(ctx, s, s.log, "search.completed", req.GetQuery(), req.GetMetadata())
	}

	success := graceful.WrapSuccess(ctx, codes.OK, "search completed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     "search:" + query,
		CacheValue:   resp,
		CacheTTL:     5 * time.Minute,
		Metadata:     meta,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "search.completed",
		EventID:      query,
		PatternType:  "search",
		PatternID:    query,
		PatternMeta:  meta,
	})

	return resp, nil
}

func (s *Service) ListSearchableFields(_ context.Context, _ *emptypb.Empty) (*searchpb.ListSearchableFieldsResponse, error) {
	resp := &searchpb.ListSearchableFieldsResponse{
		Entities: make(map[string]*searchpb.SearchableFields),
	}
	for entity, fields := range SearchFieldRegistry {
		protoFields := &searchpb.SearchableFields{}
		for _, f := range fields {
			protoFields.Fields = append(protoFields.Fields, &searchpb.SearchableField{
				Name: f.Name,
				Type: f.Type,
			})
		}
		resp.Entities[entity] = protoFields
	}
	return resp, nil
}

// RegisterAdapter registers a new search adapter.
func (s *Service) RegisterAdapter(adapter Adapter) {
	s.adapters[adapter.Name()] = adapter
}

// Search performs a federated, async, cache-enabled search across all requested sources.
// This logic is now handled by FederatedSearch or WithinSearch. If you need federated search, use FederatedSearch or WithinSearch instead.
// Note: EventEmitter interface linter error is unrelated to adapters and should be fixed separately.

// --- Example Adapters ---

// InternalDBAdapter implements Adapter for internal database search.
type InternalDBAdapter struct {
	repo *Repository
}

func (a *InternalDBAdapter) Name() string { return "internal" }

func (a *InternalDBAdapter) Search(ctx context.Context, req *Request) ([]*Result, error) {
	results, _, err := a.repo.SearchAllEntities(ctx, req.Types, req.Query, req.Metadata, 0, 0, 20)
	if err != nil {
		return nil, err
	}

	searchResults := make([]*Result, len(results))
	for i, r := range results {
		searchResults[i] = &Result{
			ID:       r.ID,
			Type:     r.EntityType,
			Score:    r.Score,
			Fields:   map[string]interface{}{"title": r.Title, "snippet": r.Snippet},
			Source:   "internal",
			Metadata: r.Metadata,
		}
	}
	return searchResults, nil
}

// GoogleSearchAdapter implements Adapter for Google Custom Search API.
type GoogleSearchAdapter struct {
	apiKey     string
	cx         string // Custom Search Engine ID
	httpClient *http.Client
}

func NewGoogleSearchAdapter(apiKey, cx string) *GoogleSearchAdapter {
	return &GoogleSearchAdapter{
		apiKey:     apiKey,
		cx:         cx,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *GoogleSearchAdapter) Name() string { return "google" }

// Helper function to safely read response body.
func readResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (a *GoogleSearchAdapter) Search(ctx context.Context, req *Request) ([]*Result, error) {
	if a.apiKey == "" || a.cx == "" {
		return nil, fmt.Errorf("google search api key or cx not configured")
	}

	// Construct Google Custom Search API URL
	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Set("key", a.apiKey)
	params.Set("cx", a.cx)
	params.Set("q", req.Query)
	if len(req.Types) > 0 {
		params.Set("type", req.Types[0]) // Use first type if specified
	}

	// Create request with context
	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, err := readResponseBody(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to read Google search response: %w", err)
		}
		return nil, fmt.Errorf("google search api error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var result struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
			Pagemap struct {
				Metatags []map[string]string `json:"metatags"`
			} `json:"pagemap"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to SearchResult format
	results := make([]*Result, 0, len(result.Items))
	for _, item := range result.Items {
		// Create search result
		result := &Result{
			ID:     item.Link,
			Type:   "web_page",
			Score:  1.0,
			Fields: make(map[string]interface{}),
			Source: "google",
		}

		// Add fields
		result.Fields["title"] = item.Title
		result.Fields["snippet"] = item.Snippet
		if len(item.Pagemap.Metatags) > 0 {
			result.Fields["metatags"] = item.Pagemap.Metatags[0]
		}

		results = append(results, result)
	}

	return results, nil
}

// WikipediaSearchAdapter implements Adapter for Wikipedia API.
type WikipediaSearchAdapter struct {
	httpClient *http.Client
	lang       string // Wikipedia language code (e.g., "en")
}

func NewWikipediaSearchAdapter(lang string) *WikipediaSearchAdapter {
	if lang == "" {
		lang = "en" // Default to English
	}
	return &WikipediaSearchAdapter{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		lang:       lang,
	}
}

func (a *WikipediaSearchAdapter) Name() string { return "wikipedia" }

func (a *WikipediaSearchAdapter) Search(ctx context.Context, req *Request) ([]*Result, error) {
	// Construct Wikipedia API URL
	baseURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php", a.lang)
	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "search")
	params.Set("srsearch", req.Query)
	params.Set("format", "json")
	params.Set("srlimit", "10")
	params.Set("srprop", "snippet|timestamp|pageid")

	// Make request to Wikipedia API
	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := readResponseBody(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to read Wikipedia search response: %w", err)
		}
		return nil, fmt.Errorf("wikipedia api error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var result struct {
		Query struct {
			Search []struct {
				PageID    int    `json:"pageid"`
				Title     string `json:"title"`
				Snippet   string `json:"snippet"`
				Timestamp string `json:"timestamp"`
			} `json:"search"`
		} `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to SearchResult format
	results := make([]*Result, 0, len(result.Query.Search))
	for _, item := range result.Query.Search {
		// Create Wikipedia URL
		wikiURL := fmt.Sprintf("https://%s.wikipedia.org/wiki/%s", a.lang, url.QueryEscape(item.Title))

		// Create search result
		result := &Result{
			ID:    fmt.Sprintf("%d", item.PageID),
			Type:  "wikipedia_article",
			Score: 1.0,
			Fields: map[string]interface{}{
				"title":     item.Title,
				"snippet":   item.Snippet,
				"url":       wikiURL,
				"timestamp": item.Timestamp,
				"page_id":   item.PageID,
			},
			Source: "wikipedia",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"title":     structpb.NewStringValue(item.Title),
						"timestamp": structpb.NewStringValue(item.Timestamp),
						"page_id":   structpb.NewNumberValue(float64(item.PageID)),
					},
				},
			},
		}
		results = append(results, result)
	}

	return results, nil
}

// DuckDuckGoSearchAdapter implements Adapter for DuckDuckGo Instant Answer API.
type DuckDuckGoSearchAdapter struct {
	httpClient *http.Client
	region     string // Region for localized results (e.g., "wt-wt" for worldwide)
}

func NewDuckDuckGoSearchAdapter(region string) *DuckDuckGoSearchAdapter {
	if region == "" {
		region = "wt-wt" // Default to worldwide
	}
	return &DuckDuckGoSearchAdapter{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		region:     region,
	}
}

func (a *DuckDuckGoSearchAdapter) Name() string { return "duckduckgo" }

func (a *DuckDuckGoSearchAdapter) Search(ctx context.Context, req *Request) ([]*Result, error) {
	// Construct DuckDuckGo API URL
	baseURL := "https://api.duckduckgo.com/"
	params := url.Values{}
	params.Set("q", req.Query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")
	params.Set("region", a.region)

	// Make request to DuckDuckGo API
	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := readResponseBody(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to read DuckDuckGo search response: %w", err)
		}
		return nil, fmt.Errorf("duckduckgo api error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var result struct {
		Abstract      string `json:"abstract"`
		AbstractText  string `json:"abstract_text"`
		AbstractURL   string `json:"abstract_url"`
		Answer        string `json:"answer"`
		RelatedTopics []struct {
			Text     string `json:"text"`
			FirstURL string `json:"first_url"`
			Icon     struct {
				URL string `json:"url"`
			} `json:"icon"`
		} `json:"related_topics"`
		Results []struct {
			Text     string `json:"text"`
			FirstURL string `json:"first_url"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to SearchResult format
	results := make([]*Result, 0)

	// Add main result if available
	if result.AbstractText != "" {
		results = append(results, &Result{
			ID:    result.AbstractURL,
			Type:  "instant_answer",
			Score: 1.0,
			Fields: map[string]interface{}{
				"abstract":      result.Abstract,
				"abstract_text": result.AbstractText,
				"url":           result.AbstractURL,
				"answer":        result.Answer,
			},
			Source: "duckduckgo",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"abstract": structpb.NewStringValue(result.Abstract),
						"answer":   structpb.NewStringValue(result.Answer),
					},
				},
			},
		})
	}

	// Add related topics
	for _, topic := range result.RelatedTopics {
		results = append(results, &Result{
			ID:    topic.FirstURL,
			Type:  "related_topic",
			Score: 0.8,
			Fields: map[string]interface{}{
				"text": topic.Text,
				"url":  topic.FirstURL,
			},
			Source: "duckduckgo",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"text": structpb.NewStringValue(topic.Text),
					},
				},
			},
		})
	}

	// Add additional results
	for _, res := range result.Results {
		results = append(results, &Result{
			ID:    res.FirstURL,
			Type:  "web_result",
			Score: 0.6,
			Fields: map[string]interface{}{
				"text": res.Text,
				"url":  res.FirstURL,
			},
			Source: "duckduckgo",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"text": structpb.NewStringValue(res.Text),
					},
				},
			},
		})
	}

	return results, nil
}

// PinterestSearchAdapter implements Adapter for Pinterest API.
type PinterestSearchAdapter struct {
	httpClient  *http.Client
	apiKey      string
	accessToken string
}

func NewPinterestSearchAdapter(apiKey, accessToken string) *PinterestSearchAdapter {
	return &PinterestSearchAdapter{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		apiKey:      apiKey,
		accessToken: accessToken,
	}
}

func (a *PinterestSearchAdapter) Name() string { return "pinterest" }

func (a *PinterestSearchAdapter) Search(ctx context.Context, req *Request) ([]*Result, error) {
	if a.apiKey == "" || a.accessToken == "" {
		return nil, fmt.Errorf("pinterest api key or access token not configured")
	}

	// Construct Pinterest API URL
	baseURL := "https://api.pinterest.com/v5/pins/search"
	params := url.Values{}
	params.Set("query", req.Query)
	params.Set("page_size", "10")
	if len(req.Types) > 0 {
		params.Set("ad_account_id", req.Types[0])
	}

	// Make request to Pinterest API
	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.accessToken))
	httpReq.Header.Set("x-api-key", a.apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := readResponseBody(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to read Pinterest search response: %w", err)
		}
		return nil, fmt.Errorf("pinterest api error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var result struct {
		Items []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Link        string `json:"link"`
			Image       struct {
				URL string `json:"url"`
			} `json:"image"`
			Board struct {
				Name string `json:"name"`
			} `json:"board"`
			CreatedAt string `json:"created_at"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to SearchResult format
	results := make([]*Result, 0, len(result.Items))
	for _, item := range result.Items {
		// Create search result
		result := &Result{
			ID:    item.ID,
			Type:  "pinterest_pin",
			Score: 1.0,
			Fields: map[string]interface{}{
				"title":       item.Title,
				"description": item.Description,
				"link":        item.Link,
				"image_url":   item.Image.URL,
				"board_name":  item.Board.Name,
				"created_at":  item.CreatedAt,
			},
			Source: "pinterest",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"title":       structpb.NewStringValue(item.Title),
						"description": structpb.NewStringValue(item.Description),
						"board_name":  structpb.NewStringValue(item.Board.Name),
						"created_at":  structpb.NewStringValue(item.CreatedAt),
					},
				},
			},
		}
		results = append(results, result)
	}

	return results, nil
}

// LinkedInSearchAdapter implements Adapter for LinkedIn API.
type LinkedInSearchAdapter struct {
	httpClient  *http.Client
	apiKey      string
	accessToken string
}

func NewLinkedInSearchAdapter(apiKey, accessToken string) *LinkedInSearchAdapter {
	return &LinkedInSearchAdapter{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		apiKey:      apiKey,
		accessToken: accessToken,
	}
}

func (a *LinkedInSearchAdapter) Name() string { return "linkedin" }

func (a *LinkedInSearchAdapter) Search(ctx context.Context, req *Request) ([]*Result, error) {
	if a.apiKey == "" || a.accessToken == "" {
		return nil, fmt.Errorf("linkedin api key or access token not configured")
	}

	// Construct LinkedIn API URL
	baseURL := "https://api.linkedin.com/v2/search"
	params := url.Values{}
	params.Set("q", req.Query)
	params.Set("count", "10")
	params.Set("start", "0")

	// Make request to LinkedIn API
	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.accessToken))
	httpReq.Header.Set("x-api-key", a.apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := readResponseBody(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to read LinkedIn search response: %w", err)
		}
		return nil, fmt.Errorf("linkedin api error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var result struct {
		Elements []struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			Title       string `json:"title"`
			Description string `json:"description"`
			URL         string `json:"url"`
			Author      struct {
				Name string `json:"name"`
			} `json:"author"`
			PublishedAt string `json:"published_at"`
			Stats       struct {
				Views    int `json:"views"`
				Likes    int `json:"likes"`
				Comments int `json:"comments"`
				Shares   int `json:"shares"`
			} `json:"stats"`
		} `json:"elements"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to SearchResult format
	results := make([]*Result, 0, len(result.Elements))
	for _, item := range result.Elements {
		// Create search result
		result := &Result{
			ID:    item.ID,
			Type:  item.Type,
			Score: 1.0,
			Fields: map[string]interface{}{
				"title":        item.Title,
				"description":  item.Description,
				"url":          item.URL,
				"author":       item.Author.Name,
				"published_at": item.PublishedAt,
				"stats":        item.Stats,
			},
			Source: "linkedin",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"title":        structpb.NewStringValue(item.Title),
						"description":  structpb.NewStringValue(item.Description),
						"author":       structpb.NewStringValue(item.Author.Name),
						"published_at": structpb.NewStringValue(item.PublishedAt),
					},
				},
			},
		}
		results = append(results, result)
	}

	return results, nil
}

// AcademicsSearchAdapter implements Adapter for academic sources (Google Scholar, Semantic Scholar, arXiv).
type AcademicsSearchAdapter struct {
	httpClient     *http.Client
	semanticAPIKey string
	arxivAPIKey    string
}

func NewAcademicsSearchAdapter(semanticAPIKey, arxivAPIKey string) *AcademicsSearchAdapter {
	return &AcademicsSearchAdapter{
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		semanticAPIKey: semanticAPIKey,
		arxivAPIKey:    arxivAPIKey,
	}
}

func (a *AcademicsSearchAdapter) Name() string { return "academics" }

func (a *AcademicsSearchAdapter) Search(ctx context.Context, req *Request) ([]*Result, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]*Result, 0)
	errs := make([]error, 0)

	// Create a child context with timeout for each search
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Search Semantic Scholar
	wg.Add(1)
	go func() {
		defer wg.Done()
		semanticResults, err := a.searchSemanticScholar(ctx, req)
		if err != nil {
			if ctx.Err() != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("semantic scholar: request cancelled: %w", ctx.Err()))
				mu.Unlock()
				return
			}
			mu.Lock()
			errs = append(errs, fmt.Errorf("semantic scholar: %w", err))
			mu.Unlock()
			return
		}
		mu.Lock()
		results = append(results, semanticResults...)
		mu.Unlock()
	}()

	// Search arXiv
	wg.Add(1)
	go func() {
		defer wg.Done()
		arxivResults, err := a.searchArxiv(ctx, req)
		if err != nil {
			if ctx.Err() != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("arxiv: request cancelled: %w", ctx.Err()))
				mu.Unlock()
				return
			}
			mu.Lock()
			errs = append(errs, fmt.Errorf("arxiv: %w", err))
			mu.Unlock()
			return
		}
		mu.Lock()
		results = append(results, arxivResults...)
		mu.Unlock()
	}()

	wg.Wait()

	if len(errs) > 0 {
		return results, fmt.Errorf("academic search errors: %v", errs)
	}

	return results, nil
}

func (a *AcademicsSearchAdapter) searchSemanticScholar(ctx context.Context, req *Request) ([]*Result, error) {
	if a.semanticAPIKey == "" {
		return nil, fmt.Errorf("semantic scholar api key not configured")
	}

	// Construct Semantic Scholar API URL
	baseURL := "https://api.semanticscholar.org/graph/v1/paper/search"
	params := url.Values{}
	params.Set("query", req.Query)
	params.Set("limit", "10")
	params.Set("fields", "title,abstract,url,year,authors,venue,citationCount")

	// Make request to Semantic Scholar API
	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("x-api-key", a.semanticAPIKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := readResponseBody(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to read academics search response: %w", err)
		}
		return nil, fmt.Errorf("semantic scholar api error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var result struct {
		Data []struct {
			PaperID  string `json:"paper_id"`
			Title    string `json:"title"`
			Abstract string `json:"abstract"`
			URL      string `json:"url"`
			Year     int    `json:"year"`
			Authors  []struct {
				Name string `json:"name"`
			} `json:"authors"`
			Venue         string `json:"venue"`
			CitationCount int    `json:"citation_count"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to SearchResult format
	results := make([]*Result, 0, len(result.Data))
	for _, item := range result.Data {
		// Create search result
		result := &Result{
			ID:    item.URL,
			Type:  "academic_paper",
			Score: 1.0,
			Fields: map[string]interface{}{
				"title":          item.Title,
				"abstract":       item.Abstract,
				"url":            item.URL,
				"year":           item.Year,
				"authors":        item.Authors,
				"venue":          item.Venue,
				"citation_count": item.CitationCount,
			},
			Source: "semantic_scholar",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"year":           structpb.NewNumberValue(float64(item.Year)),
						"venue":          structpb.NewStringValue(item.Venue),
						"citation_count": structpb.NewNumberValue(float64(item.CitationCount)),
					},
				},
			},
		}
		results = append(results, result)
	}

	return results, nil
}

func (a *AcademicsSearchAdapter) searchArxiv(ctx context.Context, req *Request) ([]*Result, error) {
	if a.arxivAPIKey == "" {
		return nil, fmt.Errorf("arxiv api key not configured")
	}

	// Construct arXiv API URL
	baseURL := "http://export.arxiv.org/api/query"
	params := url.Values{}
	params.Set("search_query", fmt.Sprintf("all:%s", req.Query))
	params.Set("start", "0")
	params.Set("max_results", "10")
	params.Set("sortBy", "relevance")
	params.Set("sortOrder", "descending")

	// Make request to arXiv API
	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.arxivAPIKey))

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := readResponseBody(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to read academics search response: %w", err)
		}
		return nil, fmt.Errorf("arxiv api error: %s - %s", resp.Status, string(body))
	}

	// Parse XML response
	decoder := xml.NewDecoder(resp.Body)
	var feed struct {
		XMLName xml.Name `xml:"feed"`
		Entries []struct {
			ID        string   `xml:"id"`
			Title     string   `xml:"title"`
			Summary   string   `xml:"summary"`
			Published string   `xml:"published"`
			Authors   []string `xml:"author>name"`
			Links     []struct {
				Href string `xml:"href,attr"`
				Rel  string `xml:"rel,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}

	if err := decoder.Decode(&feed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to SearchResult format
	results := make([]*Result, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		// Find PDF link
		var pdfURL string
		for _, link := range entry.Links {
			if link.Rel == "related" && strings.HasSuffix(link.Href, ".pdf") {
				pdfURL = link.Href
				break
			}
		}

		// Create search result
		result := &Result{
			ID:    entry.ID,
			Type:  "arxiv_paper",
			Score: 1.0,
			Fields: map[string]interface{}{
				"title":     entry.Title,
				"summary":   entry.Summary,
				"url":       entry.ID,
				"pdf_url":   pdfURL,
				"authors":   entry.Authors,
				"published": entry.Published,
			},
			Source: "arxiv",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"published": structpb.NewStringValue(entry.Published),
					},
				},
			},
		}
		results = append(results, result)
	}

	return results, nil
}

// --- AI/ML Post-Processing Layer ---

// AIProcessResults processes search results with AI enrichment.
func AIProcessResults(ctx context.Context, results []*Result, eventEmitter events.EventEmitter, log *zap.Logger) []*Result {
	seen := make(map[string]*Result)
	observer := ai.NewObserverAI()
	for _, r := range results {
		if r == nil || r.ID == "" {
			continue
		}
		if _, ok := seen[r.ID]; !ok {
			if title, ok := r.Fields["title"].(string); ok {
				emb, err := observer.Embedding.Embed([]byte(title))
				if err != nil {
					log.Error("Failed to generate embedding",
						zap.Error(err),
						zap.String("title", title))
					continue
				}
				summary, err := observer.Summarize([]byte(title))
				if err != nil {
					log.Error("Failed to generate summary",
						zap.Error(err),
						zap.String("title", title))
					continue
				}
				r.Fields["embedding"] = emb
				r.Fields["ai_summary"] = summary

				if eventEmitter != nil {
					payload, err := json.Marshal(r)
					if err != nil {
						log.Error("Failed to marshal response",
							zap.Error(err),
							zap.String("id", r.ID))
						continue
					}
					eventID, emitted := eventEmitter.EmitRawEventWithLogging(ctx, log, "search.ai_enriched", r.ID, payload)
					if !emitted {
						log.Warn("Failed to emit AI enrichment event",
							zap.String("event_id", eventID),
							zap.String("id", r.ID))
					}
				}
			}
			seen[r.ID] = r
		}
	}
	return results
}

// --- AIEnrichmentEventEmitter ---
// Emits AI-enriched search results to the event bus and WebSocket (production: inject real bus/ws).
type AIEnrichmentEventEmitter struct {
	EventBus events.EventEmitter
	Log      *zap.Logger
}

func (a *AIEnrichmentEventEmitter) EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool) {
	if a.EventBus != nil {
		return a.EventBus.EmitRawEventWithLogging(ctx, log, eventType, eventID, payload)
	}
	if log != nil {
		log.Info("AIEnrichmentEventEmitter fallback emit", zap.String("event_type", eventType), zap.String("event_id", eventID))
	}
	return eventID, false
}

func (a *AIEnrichmentEventEmitter) EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	if a.EventBus != nil {
		return a.EventBus.EmitEventWithLogging(ctx, emitter, log, eventType, eventID, meta)
	}
	return eventID, false
}

// --- Federated, Concurrent Search Orchestration ---

// FederatedSearch performs a federated search across multiple sources.
func (s *Service) FederatedSearch(ctx context.Context, req *Request) ([]*Result, error) {
	errMsgs := make([]string, 0, len(s.adapters))
	results := make([]*Result, 0, len(s.adapters)*10) // Pre-allocate for better performance

	for _, adapter := range s.adapters {
		adapterResults, err := adapter.Search(ctx, req)
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", adapter.Name(), err))
			continue
		}
		results = append(results, adapterResults...)
	}

	if len(results) == 0 && len(errMsgs) > 0 {
		return nil, fmt.Errorf("all adapters failed: %s", strings.Join(errMsgs, "; "))
	}

	return results, nil
}

// RegisterDefaultAdapters registers all production-grade external adapters.
func (s *Service) RegisterDefaultAdapters() {
	s.RegisterAdapter(&InternalDBAdapter{})
	s.RegisterAdapter(&GoogleSearchAdapter{})
	s.RegisterAdapter(&WikipediaSearchAdapter{})
	s.RegisterAdapter(&DuckDuckGoSearchAdapter{})
	s.RegisterAdapter(&PinterestSearchAdapter{})
	s.RegisterAdapter(&LinkedInSearchAdapter{})
	s.RegisterAdapter(&AcademicsSearchAdapter{})
}

// --- Generic API Adapter ---

type GenericAPIAdapter struct {
	AdapterName string
	Endpoint    string
	QueryKey    string
	Headers     map[string]string
	ParseFunc   func([]byte) ([]*Result, error)
}

func (a *GenericAPIAdapter) Search(ctx context.Context, req *Request) ([]*Result, error) {
	q := url.QueryEscape(req.Query)
	endpoint := a.Endpoint
	if a.QueryKey != "" {
		if strings.Contains(endpoint, "?") {
			endpoint += "&" + a.QueryKey + "=" + q
		} else {
			endpoint += "?" + a.QueryKey + "=" + q
		}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read generic API response: %w", err)
	}
	return a.ParseFunc(body)
}

func (a *GenericAPIAdapter) Name() string { return a.AdapterName }

// --- ParseFunc Implementations for Tier 1 ---

// Wikipedia API ParseFunc.
func parseWikipediaResults(body []byte) ([]*Result, error) {
	var resp struct {
		Query struct {
			Search []struct {
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
				PageID  int    `json:"pageid"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, s := range resp.Query.Search {
		wikiURL := "https://en.wikipedia.org/?curid=" + fmt.Sprint(s.PageID)
		results = append(results, &Result{
			ID:       wikiURL,
			Type:     "wikipedia",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": s.Title, "snippet": s.Snippet, "url": wikiURL},
			Source:   "wikipedia",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// DuckDuckGo API ParseFunc.
func parseDuckDuckGoResults(body []byte) ([]*Result, error) {
	var result struct {
		Abstract      string `json:"abstract"`
		AbstractText  string `json:"abstract_text"`
		AbstractURL   string `json:"abstract_url"`
		Answer        string `json:"answer"`
		RelatedTopics []struct {
			Text     string `json:"text"`
			FirstURL string `json:"first_url"`
			Icon     struct {
				URL string `json:"url"`
			} `json:"icon"`
		} `json:"related_topics"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, t := range result.RelatedTopics {
		results = append(results, &Result{
			ID:       t.FirstURL,
			Type:     "duckduckgo",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": t.Text, "url": t.FirstURL, "icon": t.Icon.URL},
			Source:   "duckduckgo",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// arXiv API ParseFunc (returns XML, so use a simple regex for demo; production should use encoding/xml).
func parseArxivResults(body []byte) ([]*Result, error) {
	// For brevity, this is a simple string search; production should use encoding/xml
	results := []*Result{}
	str := string(body)
	entries := strings.Split(str, "<entry>")
	for _, entry := range entries[1:] {
		titleStart := strings.Index(entry, "<title>")
		titleEnd := strings.Index(entry, "</title>")
		linkStart := strings.Index(entry, "<id>")
		linkEnd := strings.Index(entry, "</id>")
		if titleStart == -1 || titleEnd == -1 || linkStart == -1 || linkEnd == -1 {
			continue
		}
		title := strings.TrimSpace(entry[titleStart+len("<title>") : titleEnd])
		paperURL := strings.TrimSpace(entry[linkStart+len("<id>") : linkEnd])
		results = append(results, &Result{
			ID:       paperURL,
			Type:     "arxiv",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": title, "url": paperURL},
			Source:   "arxiv",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// Semantic Scholar API ParseFunc.
func parseSemanticScholarResults(body []byte) ([]*Result, error) {
	var resp struct {
		Data []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Authors []struct {
				Name string `json:"name"`
			} `json:"authors"`
			Year     int    `json:"year"`
			Abstract string `json:"abstract"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, s := range resp.Data {
		results = append(results, &Result{
			ID:       s.URL,
			Type:     "semanticscholar",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": s.Title, "url": s.URL, "authors": s.Authors, "year": s.Year, "abstract": s.Abstract},
			Source:   "semanticscholar",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// Open Library API ParseFunc.
func parseOpenLibraryResults(body []byte) ([]*Result, error) {
	var resp struct {
		Docs []struct {
			Title            string   `json:"title"`
			Key              string   `json:"key"`
			AuthorName       []string `json:"author_name"`
			FirstPublishYear int      `json:"first_publish_year"`
		} `json:"docs"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, d := range resp.Docs {
		bookURL := "https://openlibrary.org" + d.Key
		results = append(results, &Result{
			ID:       bookURL,
			Type:     "openlibrary",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": d.Title, "url": bookURL, "authors": d.AuthorName, "year": d.FirstPublishYear},
			Source:   "openlibrary",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// Unsplash/Openverse API ParseFunc (Openverse is open, Unsplash requires API key).
func parseOpenverseResults(body []byte) ([]*Result, error) {
	var resp struct {
		Results []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			Creator   string `json:"creator"`
			License   string `json:"license"`
			Thumbnail string `json:"thumbnail"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, r := range resp.Results {
		results = append(results, &Result{
			ID:       r.ID,
			Type:     "openverse",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": r.Title, "url": r.URL, "creator": r.Creator, "license": r.License, "thumbnail": r.Thumbnail},
			Source:   "openverse",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// Internet Archive API ParseFunc.
func parseInternetArchiveResults(body []byte) ([]*Result, error) {
	var resp struct {
		Response struct {
			Docs []struct {
				Title      string `json:"title"`
				Identifier string `json:"identifier"`
				Year       string `json:"year"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, d := range resp.Response.Docs {
		archiveURL := "https://archive.org/details/" + d.Identifier
		results = append(results, &Result{
			ID:       archiveURL,
			Type:     "internetarchive",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": d.Title, "url": archiveURL, "year": d.Year},
			Source:   "internetarchive",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// NewsAPI ParseFunc.
func parseNewsAPIResults(body []byte) ([]*Result, error) {
	var result struct {
		Status       string `json:"status"`
		TotalResults int    `json:"total_results"`
		Articles     []struct {
			Source struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"source"`
			Author      string `json:"author"`
			Title       string `json:"title"`
			Description string `json:"description"`
			URL         string `json:"url"`
			URLToImage  string `json:"url_to_image"`
			PublishedAt string `json:"published_at"`
			Content     string `json:"content"`
		} `json:"articles"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, a := range result.Articles {
		results = append(results, &Result{
			ID:       a.URL,
			Type:     "newsapi",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": a.Title, "url": a.URL, "source": a.Source.Name, "published_at": a.PublishedAt, "description": a.Description},
			Source:   "newsapi",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// --- Register Tier 1 Adapters ---

func (s *Service) RegisterTier1Adapters() {
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "wikipedia",
		Endpoint:    "https://en.wikipedia.org/w/api.php?action=query&list=search&format=json",
		QueryKey:    "srsearch",
		ParseFunc:   parseWikipediaResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "duckduckgo",
		Endpoint:    "https://api.duckduckgo.com/?format=json",
		QueryKey:    "q",
		ParseFunc:   parseDuckDuckGoResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "arxiv",
		Endpoint:    "http://export.arxiv.org/api/query",
		QueryKey:    "search_query",
		ParseFunc:   parseArxivResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "semanticscholar",
		Endpoint:    "https://api.semanticscholar.org/graph/v1/paper/search?fields=title,url,authors,year,abstract",
		QueryKey:    "query",
		ParseFunc:   parseSemanticScholarResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "openlibrary",
		Endpoint:    "https://openlibrary.org/search.json",
		QueryKey:    "q",
		ParseFunc:   parseOpenLibraryResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "openverse",
		Endpoint:    "https://api.openverse.engineering/v1/images",
		QueryKey:    "q",
		ParseFunc:   parseOpenverseResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "internetarchive",
		Endpoint:    "https://archive.org/advancedsearch.php?output=json&fl[]=identifier&fl[]=title&fl[]=year",
		QueryKey:    "q",
		ParseFunc:   parseInternetArchiveResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "newsapi",
		Endpoint:    "https://newsapi.org/v2/everything",
		QueryKey:    "q",
		ParseFunc:   parseNewsAPIResults,
		// Note: NewsAPI requires an API key in headers for production use
	})
}

// --- Usage Example ---
// svc := NewSearchService(redisCache)
// svc.RegisterAdapter(&InternalDBAdapter{})
// svc.RegisterAdapter(&GoogleSearchAdapter{})
// req := &SearchRequest{Query: "AI orchestration", Types: []string{"content"}, Sources: []string{"internal", "google"}}
// results, err := svc.Search(ctx, req)
// validateSearchRequest validates the search request.
func (s *Service) validateSearchRequest(_ context.Context, req *searchpb.SearchRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if req.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	return nil
}
