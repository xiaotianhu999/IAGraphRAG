package web_search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aiplusall/aiplusall-kb/internal/config"
	"github.com/aiplusall/aiplusall-kb/internal/logger"
	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
	secutils "github.com/aiplusall/aiplusall-kb/internal/utils"
)

// PerplexityProvider implements web search using Perplexity.ai chat-completions API.
type PerplexityProvider struct {
	client      *http.Client
	apiURL      string
	model       string
	temperature float64
	apiKey      string
	maxTokens   int
	recency     string
}

// NewPerplexityProvider creates a new Perplexity provider.
func NewPerplexityProvider(cfg config.WebSearchProviderConfig) (interfaces.WebSearchProvider, error) {
	apiKey := strings.TrimSpace(os.Getenv("PERPLEXITY_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("PERPLEXITY_API_KEY is required for perplexity provider")
	}

	apiURL := cfg.APIURL
	if strings.TrimSpace(apiURL) == "" {
		apiURL = "https://api.perplexity.ai/chat/completions"
	}

	model := strings.TrimSpace(os.Getenv("PERPLEXITY_MODEL"))
	if model == "" {
		model = "sonar"
	}

	temperature := 0.2
	if v := strings.TrimSpace(os.Getenv("PERPLEXITY_TEMPERATURE")); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			temperature = parsed
		}
	}

	maxTokens := 512
	if v := strings.TrimSpace(os.Getenv("PERPLEXITY_MAX_TOKENS")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxTokens = parsed
		}
	}

	recency := strings.TrimSpace(os.Getenv("PERPLEXITY_RECENCY"))

	return &PerplexityProvider{
		client:      &http.Client{Timeout: 30 * time.Second},
		apiURL:      apiURL,
		model:       model,
		temperature: temperature,
		apiKey:      apiKey,
		maxTokens:   maxTokens,
		recency:     recency,
	}, nil
}

// Name returns the provider name.
func (p *PerplexityProvider) Name() string {
	return "perplexity"
}

// Search performs a web search using Perplexity.ai chat-completions endpoint.
func (p *PerplexityProvider) Search(
	ctx context.Context,
	query string,
	maxResults int,
	includeDate bool,
) ([]*types.WebSearchResult, error) {
	if p == nil {
		return nil, fmt.Errorf("perplexity provider is not initialized")
	}
	if maxResults <= 0 {
		maxResults = 5
	}

	payload := p.buildRequestPayload(query, includeDate)
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "aiplusall-kb/1.0")

	curl := fmt.Sprintf("curl -X POST '%s' -H 'Authorization: Bearer ***' -H 'Content-Type: application/json' -d '%s'",
		p.apiURL, secutils.SanitizeForLog(buf.String()))
	logger.Infof(ctx, "Perplexity request: %s", curl)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perplexity request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("perplexity API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp perplexityResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode perplexity response: %w", err)
	}

	if apiResp.Error.Message != "" {
		return nil, fmt.Errorf("perplexity API error: %s", apiResp.Error.Message)
	}

	results := p.convertResponse(query, apiResp, maxResults)
	if len(results) == 0 {
		return nil, fmt.Errorf("perplexity returned no results")
	}
	return results, nil
}

func (p *PerplexityProvider) buildRequestPayload(query string, includeDate bool) perplexityRequest {
	req := perplexityRequest{
		Model:               p.model,
		Messages:            []perplexityMessage{{Role: "user", Content: query}},
		Temperature:         p.temperature,
		MaxTokens:           p.maxTokens,
		ReturnCitations:     true,
		SearchRecencyFilter: "",
	}
	if includeDate {
		if p.recency != "" {
			req.SearchRecencyFilter = p.recency
		} else {
			req.SearchRecencyFilter = "month"
		}
	}
	return req
}

func (p *PerplexityProvider) convertResponse(query string, resp perplexityResponse, maxResults int) []*types.WebSearchResult {
	if len(resp.Choices) == 0 {
		return nil
	}
	choice := resp.Choices[0]
	summary := strings.TrimSpace(choice.Message.Content)

	results := make([]*types.WebSearchResult, 0, maxResults)
	if summary != "" && len(results) < maxResults {
		results = append(results, &types.WebSearchResult{
			Title:   "Perplexity Summary",
			URL:     fmt.Sprintf("https://www.perplexity.ai/search?q=%s", url.QueryEscape(query)),
			Snippet: summary,
			Content: summary,
			Source:  "perplexity",
		})
	}

	citations := append([]string{}, choice.Citations...)
	if len(citations) == 0 && len(resp.Citations) > 0 {
		citations = append(citations, resp.Citations...)
	}

	seen := map[string]bool{}
	for _, citation := range citations {
		if len(results) >= maxResults {
			break
		}
		c := strings.TrimSpace(citation)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		title := fallbackTitleFromURL(c)
		results = append(results, &types.WebSearchResult{
			Title:   title,
			URL:     c,
			Snippet: summary,
			Source:  "perplexity",
		})
	}
	return results
}

// fallbackTitleFromURL builds a readable title from a URL when no explicit title is available.
func fallbackTitleFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	host := u.Host
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return host
	}
	return fmt.Sprintf("%s/%s", host, path)
}

// Internal request/response shapes

type perplexityRequest struct {
	Model               string              `json:"model"`
	Messages            []perplexityMessage `json:"messages"`
	Temperature         float64             `json:"temperature,omitempty"`
	MaxTokens           int                 `json:"max_tokens,omitempty"`
	ReturnCitations     bool                `json:"return_citations,omitempty"`
	SearchRecencyFilter string              `json:"search_recency_filter,omitempty"`
}

type perplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type perplexityChoice struct {
	Index     int               `json:"index"`
	Message   perplexityMessage `json:"message"`
	Citations []string          `json:"citations"`
}

type perplexityError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type perplexityResponse struct {
	Choices   []perplexityChoice `json:"choices"`
	Citations []string           `json:"citations"`
	Error     perplexityError    `json:"error"`
}
