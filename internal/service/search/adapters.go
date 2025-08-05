package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
)

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

	// Special handling for ConceptNet (path-based query)
	if a.AdapterName == "conceptnet" {
		// ConceptNet expects the query as a path, e.g., /c/en/word
		word := strings.ReplaceAll(strings.ToLower(req.Query), " ", "_")
		endpoint = "https://api.conceptnet.io/c/en/" + word
	} else if a.AdapterName == "gutendex" {
		// Gutendex expects ?search=...
		endpoint = "https://gutendex.com/books/?search=" + q
	} else if a.AdapterName == "wikidata" {
		// Wikidata expects SPARQL query in 'query' param
		// Use recommended label search SPARQL
		sparql := fmt.Sprintf("SELECT ?item ?itemLabel WHERE { ?item rdfs:label '%s'@en. SERVICE wikibase:label { bd:serviceParam wikibase:language 'en'. } } LIMIT 10", req.Query)
		endpoint = "https://query.wikidata.org/sparql?format=json&query=" + url.QueryEscape(sparql)
	} else if a.AdapterName == "openlibrary" {
		// OpenLibrary expects ?q=... and requires User-Agent header
		endpoint = "https://openlibrary.org/search.json?q=" + q
	} else if a.QueryKey != "" {
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
	// Add recommended User-Agent for OpenLibrary
	if a.AdapterName == "openlibrary" {
		httpReq.Header.Set("User-Agent", "Sage-Search/1.0 (contact: hello@ovasabi.com)")
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
			CoverID          int      `json:"cover_i"`
			EditionCount     int      `json:"edition_count"`
			Subject          []string `json:"subject"`
			Language         []string `json:"language"`
			PublishDate      []string `json:"publish_date"`
			Description      string   `json:"description"`
		} `json:"docs"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, d := range resp.Docs {
		bookURL := "https://openlibrary.org" + d.Key
		coverImage := ""
		if d.CoverID > 0 {
			coverImage = "https://covers.openlibrary.org/b/id/" + fmt.Sprint(d.CoverID) + "-L.jpg"
		}
		fields := map[string]interface{}{
			"title":         d.Title,
			"url":           bookURL,
			"authors":       d.AuthorName,
			"year":          d.FirstPublishYear,
			"cover_image":   coverImage,
			"edition_count": d.EditionCount,
			"subjects":      d.Subject,
			"languages":     d.Language,
			"publish_dates": d.PublishDate,
			"description":   d.Description,
		}
		results = append(results, &Result{
			ID:       bookURL,
			Type:     "openlibrary",
			Score:    1.0,
			Fields:   fields,
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
			ID      string `json:"id"`
			Title   string `json:"title"`
			URL     string `json:"url"`
			Creator string `json:"creator"`
			License string `json:"license"`
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
			Fields:   map[string]interface{}{"title": r.Title, "url": r.URL, "creator": r.Creator, "license": r.License},
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

// Wikidata SPARQL ParseFunc (returns JSON, focus on entity label and URI)
func parseWikidataResults(body []byte) ([]*Result, error) {
	var resp struct {
		Results struct {
			Bindings []struct {
				Item struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"item"`
				ItemLabel struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"itemLabel"`
			} `json:"bindings"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, b := range resp.Results.Bindings {
		results = append(results, &Result{
			ID:       b.Item.Value,
			Type:     "wikidata",
			Score:    1.0,
			Fields:   map[string]interface{}{"label": b.ItemLabel.Value, "uri": b.Item.Value},
			Source:   "wikidata",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// Gutendex API ParseFunc (Project Gutenberg)
func parseGutendexResults(body []byte) ([]*Result, error) {
	var resp struct {
		Results []struct {
			Title   string `json:"title"`
			Authors []struct {
				Name string `json:"name"`
			} `json:"authors"`
			DownloadCount int               `json:"download_count"`
			Formats       map[string]string `json:"formats"`
			ID            int               `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, r := range resp.Results {
		var author string
		if len(r.Authors) > 0 {
			author = r.Authors[0].Name
		}
		var txtURL string
		for k, v := range r.Formats {
			if strings.HasSuffix(k, "text/plain") {
				txtURL = v
				break
			}
		}
		bookURL := "https://www.gutenberg.org/ebooks/" + fmt.Sprint(r.ID)
		results = append(results, &Result{
			ID:       bookURL,
			Type:     "gutendex",
			Score:    1.0,
			Fields:   map[string]interface{}{"title": r.Title, "author": author, "url": bookURL, "txt_url": txtURL},
			Source:   "gutendex",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

// ConceptNet API ParseFunc
func parseConceptNetResults(body []byte) ([]*Result, error) {
	var resp struct {
		Edges []struct {
			Start struct {
				Label string `json:"label"`
				Term  string `json:"term"`
			} `json:"start"`
			End struct {
				Label string `json:"label"`
				Term  string `json:"term"`
			} `json:"end"`
			Rel struct {
				Label string `json:"label"`
				Term  string `json:"term"`
			} `json:"rel"`
			Weight float64 `json:"weight"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	results := []*Result{}
	for _, e := range resp.Edges {
		results = append(results, &Result{
			ID:       e.Rel.Term,
			Type:     "conceptnet",
			Score:    e.Weight,
			Fields:   map[string]interface{}{"start": e.Start.Label, "end": e.End.Label, "relation": e.Rel.Label},
			Source:   "conceptnet",
			Metadata: &commonpb.Metadata{},
		})
	}
	return results, nil
}

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
	// --- New Adapters ---
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "wikidata",
		Endpoint:    "https://query.wikidata.org/sparql?format=json",
		QueryKey:    "query",
		ParseFunc:   parseWikidataResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "gutendex",
		Endpoint:    "https://gutendex.com/books/",
		QueryKey:    "search",
		ParseFunc:   parseGutendexResults,
	})
	s.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "conceptnet",
		Endpoint:    "https://api.conceptnet.io/c/en/",
		QueryKey:    "", // ConceptNet uses path-based query, so this is a placeholder
		ParseFunc:   parseConceptNetResults,
	})
}

// RegisterAllAdapters registers all adapters (internal, external, generic) in the service.
func RegisterAllAdapters(svc *Service) {
	svc.RegisterAdapter(&InternalDBAdapter{repo: svc.repo})
	svc.RegisterAdapter(NewWikipediaSearchAdapter("en"))
	svc.RegisterAdapter(NewDuckDuckGoSearchAdapter("wt-wt"))
	// Exclude adapters that require API keys (Pinterest, LinkedIn, Google, Academics)

	// --- Register generic/knowledge adapters ---

	// Wikidata: SPARQL endpoint, label search
	svc.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "wikidata",
		Endpoint:    "https://query.wikidata.org/sparql?format=json",
		QueryKey:    "query", // SPARQL query param
		ParseFunc:   parseWikidataResults,
	})

	// Gutendex: Project Gutenberg API, search param
	svc.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "gutendex",
		Endpoint:    "https://gutendex.com/books/",
		QueryKey:    "search", // ?search=...
		ParseFunc:   parseGutendexResults,
	})

	// ConceptNet: path-based query, e.g. /c/en/word
	svc.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "conceptnet",
		Endpoint:    "https://api.conceptnet.io/c/en/",
		QueryKey:    "", // path-based, handled in adapter logic
		ParseFunc:   parseConceptNetResults,
	})

	// OpenLibrary: book search, requires User-Agent header
	svc.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "openlibrary",
		Endpoint:    "https://openlibrary.org/search.json",
		QueryKey:    "q", // ?q=...
		ParseFunc:   parseOpenLibraryResults,
		Headers:     map[string]string{"User-Agent": "Sage-Search/1.0 (contact: hello@ovasabi.com)"},
	})

	// arXiv: scientific papers, search_query param
	svc.RegisterAdapter(&GenericAPIAdapter{
		AdapterName: "arxiv",
		Endpoint:    "http://export.arxiv.org/api/query",
		QueryKey:    "search_query", // ?search_query=...
		ParseFunc:   parseArxivResults,
	})
}
