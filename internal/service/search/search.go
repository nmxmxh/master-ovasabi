package search

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

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
	handler      *graceful.Handler
}

// NewService creates a new SearchService instance with event bus and provider support (canonical pattern).
func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool, provider *service.Provider) searchpb.SearchServiceServer {
	svc := &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		provider:     provider,
		adapters:     make(map[string]Adapter),
		handler:      graceful.NewHandler(log, eventEmitter, cache, "search", "v1", eventEnabled),
	}

	// Centralized registration of all adapters
	RegisterAllAdapters(svc)
	return svc
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

// handleSearchAction is the generic business logic handler for the "search" action, used by the generic event handler.
func handleSearchAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	// Only process canonical 'requested' events, ignore 'started', 'completed', 'failed', etc.
	if !strings.HasSuffix(event.GetEventType(), ":requested") {
		s.log.Debug("[handleSearchAction] Ignoring non-requested event (only handling 'requested')", zap.String("event_type", event.GetEventType()), zap.String("event_id", event.EventId))
		return
	}

	if event == nil {
		s.log.Warn("search event is nil")
		s.handler.Error(ctx, "search", codes.InvalidArgument, "Search event is nil", nil, nil, "")
		return
	}

	// For requested events, we require payload and event_id
	if event.Payload == nil || event.EventId == "" {
		s.log.Warn("search requested event missing payload or event_id", zap.String("event_type", event.GetEventType()), zap.String("event_id", event.EventId))
		s.handler.Error(ctx, "search", codes.InvalidArgument, "Missing payload or event_id in search requested event", nil, event.Metadata, event.EventId)
		return
	}
	// Abbreviate payload and metadata for logging
	payloadPreview := "nil"
	if event.Payload != nil && event.Payload.Data != nil {
		keys := make([]string, 0, len(event.Payload.Data.Fields))
		for k := range event.Payload.Data.Fields {
			keys = append(keys, k)
		}
		payloadPreview = "fields: [" + strings.Join(keys, ",") + "]"
	}
	metaPreview := "nil"
	if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
		keys := make([]string, 0, len(event.Metadata.ServiceSpecific.Fields))
		for k := range event.Metadata.ServiceSpecific.Fields {
			keys = append(keys, k)
		}
		metaPreview = "serviceSpecific: [" + strings.Join(keys, ",") + "]"
	}
	s.log.Info("[handleSearchAction] Invoked", zap.String("event_type", event.GetEventType()), zap.String("payload", payloadPreview), zap.String("metadata", metaPreview))

	// Debug: Log the raw payload field count to see what we're receiving
	if event.Payload != nil && event.Payload.Data != nil {
		s.log.Debug("[handleSearchAction] Received payload field count",
			zap.String("event_type", event.GetEventType()),
			zap.Int("field_count", len(event.Payload.Data.Fields)))
	}

	// Define canonical event types for emission
	canonicalStarted := GetCanonicalEventType("search", "started")
	if canonicalStarted == "" {
		s.log.Warn("Failed to resolve canonical event type for search:started")
		canonicalStarted = "search:search:v1:started" // fallback
	}
	var req searchpb.SearchRequest
	if event.Payload.Data != nil {
		// Pre-process campaign_id: if string and convertible to int, convert; if not, fail gracefully
		fields := event.Payload.Data.Fields
		if v, ok := fields["campaign_id"]; ok {
			switch val := v.Kind.(type) {
			case *structpb.Value_StringValue:
				// Try to convert string to int64
				if val.StringValue == "0" {
					fields["campaign_id"] = structpb.NewNumberValue(0)
				} else if cid, err := strconv.ParseInt(val.StringValue, 10, 64); err == nil {
					fields["campaign_id"] = structpb.NewNumberValue(float64(cid))
				} else {
					s.handler.Error(ctx, "search", codes.InvalidArgument, "Invalid campaign_id: must be int64 or '0'", nil, event.Metadata, event.EventId)
					return
				}
			case *structpb.Value_NumberValue:
				// Already a number, do nothing
			}
		}

		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
			if err != nil {
				s.log.Warn("failed to unmarshal search request from event payload", zap.Error(err))
				s.handler.Error(ctx, "search", codes.InvalidArgument, "Failed to unmarshal search request: "+err.Error(), err, event.Metadata, event.EventId)
				return
			}
		}
	}

	meta := req.GetMetadata()
	if meta == nil {
		meta = event.Metadata
	}
	// Don't emit started events from within the handler - this creates a loop
	// Started events should be emitted by the orchestration layer, not the service itself

	// Default to federated search unless service_specific context says otherwise
	searchType := "federated"
	if meta != nil && meta.ServiceSpecific != nil {
		if v, ok := meta.ServiceSpecific.Fields["search_type"]; ok {
			if v.GetStringValue() != "" {
				searchType = v.GetStringValue()
			}
		}
	}

	var resp *searchpb.SearchResponse
	var err error
	switch searchType {
	case "internal":
		resp, err = s.WithinSearch(ctx, &req)
	case "within":
		resp, err = s.WithinSearch(ctx, &req)
	case "federated":
		fallthrough
	default:
		fedReq := &Request{
			Query:    req.GetQuery(),
			Types:    req.GetTypes(),
			Metadata: meta,
		}
		results, ferr := s.FederatedSearch(ctx, fedReq)
		// Always push results forward, even if partial or error
		// Removed results logging for inspection clarity
		protos := make([]*searchpb.SearchResult, 0, len(results))
		for _, r := range results {
			fields := make(map[string]interface{})
			for k, v := range r.Fields {
				fields[k] = v
			}
			// Add Source field for full adapter detail
			fields["source"] = r.Source
			protos = append(protos, &searchpb.SearchResult{
				Id:         r.ID,
				EntityType: r.Type,
				Score:      float32(r.Score),
				Fields:     metadata.NewStructFromMap(fields, nil),
				Metadata:   r.Metadata,
			})
		}
		resp = &searchpb.SearchResponse{
			Results:    protos,
			Total:      int32(len(results)),
			PageNumber: req.GetPageNumber(),
			PageSize:   req.GetPageSize(),
		}
		// Optionally, log or attach error details to metadata for frontend display
		// If you want to include error details in the response, you can add a ServiceSpecific field:
		if ferr != nil && meta != nil && meta.ServiceSpecific != nil {
			// Abbreviate the error to just the list of failing adapters for a cleaner frontend display.
			errMsg := strings.TrimPrefix(ferr.Error(), "adapter errors: ")
			meta.ServiceSpecific.Fields["adapter_errors"] = structpb.NewStringValue(errMsg)
		}
	}

	// Only call HandleServiceError for true system errors (not adapter errors)
	if err != nil {
		s.handler.Error(ctx, "search", codes.Internal, "event-driven search failed", err, meta, event.EventId)
		return
	}

	// --- Canonical Metadata Merging ---
	// Merge only the ServiceSpecific field, as backend metadata only supports ServiceSpecific
	if event.Metadata != nil && meta != nil {
		meta = metadata.MergeMetadata(meta, event.Metadata)
	}

	// --- Payload Serialization ---
	var completedPayload *structpb.Struct
	if resp != nil {
		b, err := protojson.Marshal(resp)
		if err == nil {
			completedPayload = &structpb.Struct{}
			_ = protojson.Unmarshal(b, completedPayload)
		}
	}

	// Abbreviate completedPayload and metadata for logging
	completedPreview := "nil"
	if completedPayload != nil && completedPayload.Fields != nil {
		keys := make([]string, 0, len(completedPayload.Fields))
		for k := range completedPayload.Fields {
			keys = append(keys, k)
		}
		completedPreview = "fields: [" + strings.Join(keys, ",") + "]"
	}
	metaPreview2 := "nil"
	if meta != nil && meta.ServiceSpecific != nil {
		keys := make([]string, 0, len(meta.ServiceSpecific.Fields))
		for k := range meta.ServiceSpecific.Fields {
			keys = append(keys, k)
		}
		metaPreview2 = "serviceSpecific: [" + strings.Join(keys, ",") + "]"
	}
	s.log.Info("[handleSearchAction] Emitting completed event", zap.String("event_id", event.EventId), zap.String("completedPayload", completedPreview), zap.String("metadata", metaPreview2))
	s.handler.Success(ctx, "search", codes.OK, "event-driven search completed", completedPayload, meta, event.EventId, nil)
}

// Suggest implements the gRPC endpoint for search suggestions.
func (s *Service) Suggest(ctx context.Context, req *searchpb.SuggestRequest) (*searchpb.SuggestResponse, error) {
	s.log.Info("[Suggest] Invoked", zap.String("prefix", req.GetPrefix()), zap.Any("metadata", req.GetMetadata()))
	if req.GetPrefix() == "" {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.InvalidArgument, "prefix cannot be empty", nil))
	}

	query := req.GetPrefix()
	limit := int(req.GetLimit())
	meta := req.GetMetadata()

	suggestions, err := s.repo.Suggest(ctx, query, limit)
	if err != nil {
		s.handler.Error(ctx, "suggest", codes.Internal, "suggest failed", err, meta, query)
		return nil, graceful.ToStatusError(err)
	}

	resp := &searchpb.SuggestResponse{
		Suggestions: suggestions,
	}

	s.handler.Success(ctx, "suggest", codes.OK, "suggest completed", resp, meta, query, nil)

	return resp, nil
}

// handleSuggestAction is the generic business logic handler for the "suggest" action.
func handleSuggestAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil {
		s.log.Warn("[handleSuggestAction] Missing event")
		return
	}
	// Abbreviate payload and metadata for logging
	payloadPreview := "nil"
	if event.Payload != nil && event.Payload.Data != nil {
		keys := make([]string, 0, len(event.Payload.Data.Fields))
		for k := range event.Payload.Data.Fields {
			keys = append(keys, k)
		}
		payloadPreview = "fields: [" + strings.Join(keys, ",") + "]"
	}
	metaPreview := "nil"
	if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
		keys := make([]string, 0, len(event.Metadata.ServiceSpecific.Fields))
		for k := range event.Metadata.ServiceSpecific.Fields {
			keys = append(keys, k)
		}
		metaPreview = "serviceSpecific: [" + strings.Join(keys, ",") + "]"
	}
	s.log.Info("[handleSuggestAction] Invoked", zap.String("event_type", event.GetEventType()), zap.String("payload", payloadPreview), zap.String("metadata", metaPreview))

	if event.Payload == nil || event.Payload.Data == nil {
		s.log.Warn("[handleSuggestAction] Missing or invalid event payload", zap.Any("event", event))
		return
	}

	var req searchpb.SuggestRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err != nil {
		s.log.Warn("[handleSuggestAction] Failed to marshal event payload to JSON", zap.Error(err))
		return
	}
	if err := protojson.Unmarshal(b, &req); err != nil {
		s.log.Warn("[handleSuggestAction] Failed to unmarshal payload to SuggestRequest", zap.Error(err))
		return
	}

	meta := req.GetMetadata()
	if meta == nil {
		meta = event.Metadata
	}

	suggestType := "federated"
	if meta != nil && meta.ServiceSpecific != nil {
		if v, ok := meta.ServiceSpecific.Fields["suggest_type"]; ok {
			if v.GetStringValue() != "" {
				suggestType = v.GetStringValue()
			}
		}
	}

	var resp *searchpb.SuggestResponse
	s.log.Info("[handleSuggestAction] Dispatching suggestType", zap.String("suggest_type", suggestType))
	switch suggestType {
	case "internal":
		resp, err = s.Suggest(ctx, &req)
	case "federated":
		fallthrough
	default:
		resp, err = s.Suggest(ctx, &req)
	}

	if err != nil {
		s.log.Warn("[handleSuggestAction] Suggest failed", zap.Error(err))
		s.handler.Error(ctx, "suggest", codes.Internal, "suggest failed", err, meta, event.EventId)
		return
	}

	// Abbreviate response for logging
	respPreview := "nil"
	if resp != nil && resp.Suggestions != nil {
		respPreview = "suggestions: [" + strconv.Itoa(len(resp.Suggestions)) + "]"
	}
	s.log.Info("[handleSuggestAction] Suggest succeeded", zap.String("response", respPreview))
	s.handler.Success(ctx, "suggest", codes.OK, "event-driven suggest completed", resp, meta, event.EventId, nil)
}

// Search implements robust multi-entity, FTS, and metadata filtering search.
// Supports searching across multiple entity types as specified in req.Types.
func (s *Service) WithinSearch(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	// Validate request
	if err := s.validateSearchRequest(ctx, req); err != nil {
		return nil, err
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
		s.handler.Error(ctx, "search", codes.Internal, "search failed", err, meta, query)
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
	baseURL := "https://api.duckduckgo.com/"
	params := url.Values{}
	params.Set("q", req.Query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")
	params.Set("region", a.region)

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

	var result struct {
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]*Result, 0)
	// Instant answer (text + link)
	if result.AbstractText != "" && result.AbstractURL != "" {
		results = append(results, &Result{
			ID:    result.AbstractURL,
			Type:  "instant_answer",
			Score: 1.0,
			Fields: map[string]interface{}{
				"text": result.AbstractText,
				"url":  result.AbstractURL,
			},
			Source: "duckduckgo",
		})
	}
	// Related topics (text + link)
	for _, topic := range result.RelatedTopics {
		if topic.Text != "" && topic.FirstURL != "" {
			results = append(results, &Result{
				ID:    topic.FirstURL,
				Type:  "related_topic",
				Score: 0.8,
				Fields: map[string]interface{}{
					"text": topic.Text,
					"url":  topic.FirstURL,
				},
				Source: "duckduckgo",
			})
		}
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
	// Use public endpoint, focus on text and links only
	baseURL := "https://api.semanticscholar.org/graph/v1/paper/search"
	params := url.Values{}
	params.Set("query", req.Query)
	params.Set("limit", "10")
	params.Set("fields", "title,url")

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
			return nil, fmt.Errorf("failed to read academics search response: %w", err)
		}
		return nil, fmt.Errorf("semantic scholar api error: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Data []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]*Result, 0, len(result.Data))
	for _, item := range result.Data {
		if item.Title != "" && item.URL != "" {
			results = append(results, &Result{
				ID:    item.URL,
				Type:  "academic_paper",
				Score: 1.0,
				Fields: map[string]interface{}{
					"title": item.Title,
					"url":   item.URL,
				},
				Source: "semantic_scholar",
			})
		}
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
	for _, r := range results {
		if r == nil || r.ID == "" {
			continue
		}
		if _, ok := seen[r.ID]; !ok {
			if eventEmitter != nil {
				payloadBytes, err := json.Marshal(r)
				if err != nil {
					log.Error("Failed to marshal result for AI enrichment", zap.Error(err), zap.String("id", r.ID))
					continue
				}
				// Convert payload to *structpb.Struct
				var payloadMap map[string]interface{}
				if err := json.Unmarshal(payloadBytes, &payloadMap); err != nil {
					log.Error("Failed to unmarshal result for structpb conversion", zap.Error(err), zap.String("id", r.ID))
					continue
				}
				payloadStruct := metadata.NewStructFromMap(payloadMap, nil)
				envelope := &events.EventEnvelope{
					ID:        r.ID,
					Type:      "search.ai_enrichment_requested",
					Payload:   &commonpb.Payload{Data: payloadStruct},
					Metadata:  r.Metadata,
					Timestamp: time.Now().Unix(),
				}
				eventID, err := eventEmitter.EmitEventEnvelope(ctx, envelope)
				if err != nil {
					log.Warn("Failed to emit AI enrichment request event", zap.String("event_id", eventID), zap.String("id", r.ID), zap.Error(err))
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
	// Deprecated: Use EmitEventEnvelopeWithLogging instead.
	if log != nil {
		log.Info("AIEnrichmentEventEmitter fallback emit (deprecated)", zap.String("event_type", eventType), zap.String("event_id", eventID))
	}
	return eventID, false
}

func (a *AIEnrichmentEventEmitter) EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	// Deprecated: Use EmitEventEnvelopeWithLogging instead.
	return eventID, false
}

// --- Federated, Concurrent Search Orchestration ---

// FederatedSearch performs a federated search across multiple sources.
func (s *Service) FederatedSearch(ctx context.Context, req *Request) ([]*Result, error) {
	errMsgs := make([]string, 0, len(s.adapters))
	// Track adapter order for sorting
	adapterOrder := make(map[string]int)
	idx := 0
	for name := range s.adapters {
		adapterOrder[name] = idx
		idx++
	}
	type resultWithAdapterIdx struct {
		*Result
		adapterIdx int
	}
	allResults := make([]resultWithAdapterIdx, 0, len(s.adapters)*10)
	var mu sync.Mutex
	var wg sync.WaitGroup

	s.log.Info("[FederatedSearch] Starting federated search", zap.Int("adapter_count", len(s.adapters)), zap.Strings("adapter_names", func() []string {
		names := make([]string, 0, len(s.adapters))
		for name := range s.adapters {
			names = append(names, name)
		}
		return names
	}()))

	// Set a 3s timeout for the entire federated search (increase for slow APIs)
	searchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	for _, adapter := range s.adapters {
		wg.Add(1)
		go func(adapter Adapter) {
			defer wg.Done()
			var adapterWg sync.WaitGroup
			adapterResultsCh := make(chan []*Result, 10)
			errCh := make(chan error, 10)
			for _, workerID := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
				adapterWg.Add(1)
				go func(workerID int) {
					defer adapterWg.Done()
					workerReq := *req
					switch workerID {
					case 0:
						workerReq.Query = req.Query + " definition"
					case 1:
						workerReq.Query = req.Query + " fundamentals"
					case 2:
						workerReq.Query = req.Query + " core facts"
					case 3:
						workerReq.Query = req.Query + " translation"
					case 4:
						workerReq.Query = req.Query + " regional adaptation"
					case 5:
						workerReq.Query = req.Query + " applications industry deployments"
					case 6:
						workerReq.Query = req.Query + " criticisms risks limitations ethical debates"
					case 7:
						workerReq.Query = req.Query + " demographics adoption trends user analysis"
					case 8:
						workerReq.Query = req.Query + " interdisciplinary cross-domain relevance"
					case 9:
						workerReq.Query = req.Query + " future trajectory innovation predictions"
					}
					s.log.Debug("[FederatedSearch] Adapter worker", zap.String("adapter", adapter.Name()), zap.Int("worker", workerID), zap.String("query", workerReq.Query))
					res, err := adapter.Search(searchCtx, &workerReq)
					if err != nil {
						errCh <- fmt.Errorf("worker %d: %w", workerID, err)
						return
					}
					adapterResultsCh <- res
				}(workerID)
			}
			adapterWg.Wait()
			close(adapterResultsCh)
			close(errCh)
			mu.Lock()
			for res := range adapterResultsCh {
				if len(res) > 0 {
					s.log.Info("[FederatedSearch] Adapter results", zap.String("adapter", adapter.Name()), zap.Int("result_count", len(res)), zap.Any("results", res))
					idx := adapterOrder[adapter.Name()]
					for _, r := range res {
						allResults = append(allResults, resultWithAdapterIdx{Result: r, adapterIdx: idx})
					}
				}
			}
			for err := range errCh {
				s.log.Warn("[FederatedSearch] Adapter error", zap.String("adapter", adapter.Name()), zap.Error(err))
				errMsgs = append(errMsgs, adapter.Name())
			}
			mu.Unlock()
		}(adapter)
	}
	wg.Wait()

	// Sort allResults by a weighted composite score: mix of adapterIdx (priority) and Score
	// Higher score and higher priority (lower adapterIdx) rank higher
	scoreWeight := 0.7
	priorityWeight := 0.3
	maxAdapterIdx := 0
	for _, rw := range allResults {
		if rw.adapterIdx > maxAdapterIdx {
			maxAdapterIdx = rw.adapterIdx
		}
	}
	sort.SliceStable(allResults, func(i, j int) bool {
		// Composite score: scoreWeight * Score + priorityWeight * (maxAdapterIdx - adapterIdx)
		compositeI := scoreWeight*allResults[i].Score + priorityWeight*float64(maxAdapterIdx-allResults[i].adapterIdx)
		compositeJ := scoreWeight*allResults[j].Score + priorityWeight*float64(maxAdapterIdx-allResults[j].adapterIdx)
		return compositeI > compositeJ
	})

	s.log.Info("[FederatedSearch] Aggregated results", zap.Int("total_results", len(allResults)), zap.Strings("errors", errMsgs))

	pageNumber := 1
	pageSize := 100
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		if v, ok := req.Metadata.ServiceSpecific.Fields["page_number"]; ok {
			if v.GetNumberValue() > 0 {
				pageNumber = int(v.GetNumberValue())
			}
		}
		if v, ok := req.Metadata.ServiceSpecific.Fields["page_size"]; ok {
			if v.GetNumberValue() > 0 {
				pageSize = int(v.GetNumberValue())
			}
		}
	}
	start := (pageNumber - 1) * pageSize
	end := start + pageSize
	if start > len(allResults) {
		return []*Result{}, fmt.Errorf("adapter errors: %s", strings.Join(errMsgs, "; "))
	}
	if end > len(allResults) {
		end = len(allResults)
	}

	// Deduplicate results by ID before paging
	deduped := make([]*Result, 0, len(allResults))
	seen := make(map[string]struct{})
	for _, rw := range allResults {
		if rw.Result == nil || rw.Result.ID == "" {
			continue
		}
		if _, exists := seen[rw.Result.ID]; !exists {
			seen[rw.Result.ID] = struct{}{}
			deduped = append(deduped, rw.Result)
		}
	}

	// Apply paging after deduplication
	pagedResults := make([]*Result, 0, end-start)
	if start < len(deduped) {
		if end > len(deduped) {
			end = len(deduped)
		}
		pagedResults = deduped[start:end]
	}

	if len(errMsgs) > 0 {
		return pagedResults, fmt.Errorf("adapter errors: %s", strings.Join(errMsgs, ", "))
	}
	return pagedResults, nil
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
