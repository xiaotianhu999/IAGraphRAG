package web_search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aiplusall/aiplusall-kb/internal/config"
)

func TestPerplexityProvider_Name(t *testing.T) {
	t.Setenv("PERPLEXITY_API_KEY", "test-key")
	prov, err := NewPerplexityProvider(config.WebSearchProviderConfig{})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	if prov.Name() != "perplexity" {
		t.Fatalf("expected provider name perplexity, got %s", prov.Name())
	}
}

func TestPerplexityProvider_Search(t *testing.T) {
	t.Setenv("PERPLEXITY_API_KEY", "test-key")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		var req perplexityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "sonar" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		if req.SearchRecencyFilter == "" {
			t.Fatalf("expected search recency filter to be set when includeDate is true")
		}

		resp := perplexityResponse{
			Choices: []perplexityChoice{
				{
					Index:     0,
					Message:   perplexityMessage{Role: "assistant", Content: "Summary content"},
					Citations: []string{"https://example.com/a", "https://example.org/b"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	prov, err := NewPerplexityProvider(config.WebSearchProviderConfig{APIURL: ts.URL})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	pp := prov.(*PerplexityProvider)
	pp.client = &http.Client{Timeout: 5 * time.Second}

	ctx := context.Background()
	results, err := pp.Search(ctx, "weknora", 3, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "Perplexity Summary" || results[0].Source != "perplexity" {
		t.Fatalf("unexpected summary result: %+v", results[0])
	}
	if results[1].URL != "https://example.com/a" || results[2].URL != "https://example.org/b" {
		t.Fatalf("unexpected citation ordering: %+v", results)
	}
}

func TestPerplexityProvider_MissingAPIKey(t *testing.T) {
	t.Setenv("PERPLEXITY_API_KEY", "")
	if _, err := NewPerplexityProvider(config.WebSearchProviderConfig{}); err == nil {
		t.Fatalf("expected error when API key is missing")
	}
}
