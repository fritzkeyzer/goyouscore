package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fritzkeyzer/goyouscore"
)

const (
	cacheFile = "cache.json"
	cacheTTL  = 7 * 24 * time.Hour // cached entries expire after 7 days
)

// Ensure jsonCache implements youscore.Cache at compile time.
var _ youscore.Cache = (*jsonCache)(nil)

func main() {
	apiKeys := youscore.APIKeys{
		DataAnalytics:    os.Getenv("YOUSCORE_DATA_ANALYTICS_KEY"),
		PDFLegalEntities: os.Getenv("YOUSCORE_PDF_LEGAL_KEY"),
		PDFIndividuals:   os.Getenv("YOUSCORE_PDF_INDIVIDUALS_KEY"),
		Affiliates:       os.Getenv("YOUSCORE_AFFILIATES_KEY"),
	}
	if apiKeys.DataAnalytics == "" {
		fmt.Println("YOUSCORE_DATA_ANALYTICS_KEY environment variable is required")
		os.Exit(1)
	}
	if apiKeys.PDFLegalEntities == "" {
		fmt.Println("YOUSCORE_PDF_LEGAL_KEY environment variable is required")
		os.Exit(1)
	}
	if apiKeys.PDFIndividuals == "" {
		fmt.Println("YOUSCORE_PDF_INDIVIDUALS_KEY environment variable is required")
		os.Exit(1)
	}
	if apiKeys.Affiliates == "" {
		fmt.Println("YOUSCORE_AFFILIATES_KEY environment variable is required")
		os.Exit(1)
	}

	cache := newJSONCache()

	// Create a client with per-endpoint API key authentication and file-backed caching.
	cl, err := youscore.NewClientWithResponses(youscore.ServerURL,
		youscore.WithAPIKeys(apiKeys),
		youscore.WithCache(cache),
	)
	if err != nil {
		fmt.Println("error creating client:", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Example: Look up registration data (USR) for a company by its EDRPOU code
	contractorCode := "08215600" // example EDRPOU code
	usrResp, err := cl.GetV1UsrContractorCodeWithResponse(ctx, contractorCode, &youscore.GetV1UsrContractorCodeParams{})
	if err != nil {
		fmt.Println("error fetching USR data:", err)
		os.Exit(1)
	}

	fmt.Println("USR response status:", usrResp.Status())
	buf, _ := json.MarshalIndent(usrResp.JSON200, "", "  ")
	fmt.Println("USR response:", string(buf))

	// Example: Check express analysis for a company
	expressAnalysisResp, err := cl.GetV1ExpressAnalysisContractorCodeWithResponse(ctx, contractorCode, nil)
	if err != nil {
		fmt.Println("error fetching express analysis:", err)
		os.Exit(1)
	}

	fmt.Println("Express analysis status:", expressAnalysisResp.Status())
	buf, _ = json.MarshalIndent(expressAnalysisResp.JSON200, "", "  ")
	fmt.Println("Express analysis:", string(buf))

	//// Example: Check sanctions for a company
	//sanctionsResp, err := cl.GetV1SanctionsWithResponse(ctx, &youscore.GetV1SanctionsParams{
	//	ContractorCode: &contractorCode,
	//})
	//if err != nil {
	//	fmt.Println("error fetching sanctions:", err)
	//	os.Exit(1)
	//}
	//
	//fmt.Println("Sanctions response status:", sanctionsResp.Status())
	//buf, _ = json.MarshalIndent(sanctionsResp.JSON200, "", "  ")
	//fmt.Println("Sanctions response:", string(buf))

	// Example: Check rate limits (for all non-blank keys - using the custom utility)
	rateLimitsResp, err := youscore.CheckRateLimits(ctx, apiKeys)
	if err != nil {
		fmt.Println("error fetching rate limits:", err)
		os.Exit(1)
	}
	buf, _ = json.MarshalIndent(rateLimitsResp, "", "  ")
	fmt.Println("Rate limits response:", string(buf))
}

// NOTE: This file-backed JSON cache is a dumb example implementation.
// A realistic application should use a more sophisticated caching mechanism
// (e.g. Redis, SQLite, bbolt, or an in-memory LRU cache with proper eviction).
// Writing the entire cache to a JSON file on every update is computationally
// wasteful and does not scale, but it ensures the cache is persisted even if
// the program exits, errors, or is cancelled.

// jsonCache is a map-based cache that persists to a JSON file on every write.
// It implements the youscore.Cache interface.
type jsonCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

// newJSONCache creates a new jsonCache, loading existing entries from the cache file.
func newJSONCache() *jsonCache {
	c := &jsonCache{
		entries: make(map[string]cacheEntry),
	}
	c.load()
	return c
}

// cacheEntry represents a single cached HTTP response with metadata.
type cacheEntry struct {
	URL       string                  `json:"url"`
	Key       string                  `json:"key"`
	Timestamp time.Time               `json:"timestamp"`
	Response  youscore.CachedResponse `json:"response"`
}

// load reads the cache file from disk into memory.
func (c *jsonCache) load() {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return // file doesn't exist yet or can't be read — start with empty cache
	}

	var entries map[string]cacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return // corrupted file — start fresh
	}

	c.entries = entries
}

// save writes the entire cache map to the JSON file.
func (c *jsonCache) save() {
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cache: failed to marshal: %v\n", err)
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "cache: failed to write file: %v\n", err)
	}
}

// Get retrieves a cached response if it exists and is not expired.
func (c *jsonCache) Get(url string, key string) (youscore.CachedResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		fmt.Println("cache miss:", key)
		return youscore.CachedResponse{}, false
	}

	if time.Since(entry.Timestamp) > cacheTTL {
		fmt.Println("cache miss:", key)
		delete(c.entries, key)
		c.save()
		return youscore.CachedResponse{}, false
	}

	fmt.Println("cache hit:", key)
	return entry.Response, true
}

// Set stores a response in the cache and immediately persists to disk.
func (c *jsonCache) Set(url string, key string, resp youscore.CachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// skip non-200 responses
	if resp.StatusCode != 200 {
		return
	}

	c.entries[key] = cacheEntry{
		URL:       url,
		Key:       key,
		Timestamp: time.Now(),
		Response:  resp,
	}

	c.save()
}
