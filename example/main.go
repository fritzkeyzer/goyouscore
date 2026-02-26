package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fritzkeyzer/goyouscore"
)

func main() {
	ctx := context.Background()

	apiKeys := youscore.APIKeys{
		DataAnalytics:    os.Getenv("YOUSCORE_DATA_ANALYTICS_KEY"),
		PDFLegalEntities: os.Getenv("YOUSCORE_PDF_LEGAL_KEY"),
		PDFIndividuals:   os.Getenv("YOUSCORE_PDF_INDIVIDUALS_KEY"),
		Affiliates:       os.Getenv("YOUSCORE_AFFILIATES_KEY"),
	}
	if apiKeys.DataAnalytics == "" || apiKeys.PDFLegalEntities == "" || apiKeys.PDFIndividuals == "" || apiKeys.Affiliates == "" {
		log.Fatal("ERROR: missing required API keys:", apiKeys)
	}

	// create a silly file based cache (for the sake of this demo)
	cache := newDumbFileCache()

	// create simple usage tracker
	totalUsage := new(usage)
	usageTracker := newSimpleUsageTracker(totalUsage)
	defer totalUsage.Print()

	// create client with per-endpoint API key auth and a cache
	cl, err := youscore.NewClientWithResponses(youscore.ServerURL,
		youscore.WithAPIKeys(apiKeys),
		youscore.WithCache(cache),
		youscore.WithUsageTracking(usageTracker),
	)
	if err != nil {
		log.Fatal("ERROR: create client")
	}

	// example EDRPOU code
	contractorCode := "08215600"

	// Example: Look up registration data (USR) for a company by its EDRPOU code
	usrResp, err := cl.GetV1UsrContractorCodeWithResponse(ctx, contractorCode, &youscore.GetV1UsrContractorCodeParams{
		ShowCurrentData: ptr(true),
	})
	if err != nil {
		log.Fatal("ERROR: get USR:", err)
	}
	log.Println("Get USR status:", usrResp.StatusCode())
	if usrResp.StatusCode() == http.StatusOK {
		//log.Println("Get USR:", toJSUtil(usrResp.JSON200))
		dumpJSUtil(contractorCode+"_usr", usrResp.JSON200)
	}

	// Example: Look up ownership
	usrOwnershipResp, err := cl.GetV1UsrDocumentsUsrOwnershipStructureFileWithResponse(ctx, &youscore.GetV1UsrDocumentsUsrOwnershipStructureFileParams{
		Code: &contractorCode,
	})
	if err != nil {
		log.Fatal("ERROR: get ownership:", err)
	}
	log.Println("Get ownership status:", usrOwnershipResp.StatusCode())
	if usrOwnershipResp.StatusCode() == http.StatusOK {
		//log.Println("Get ownership:", toJSUtil(usrOwnershipResp.JSON200))
		dumpJSUtil(contractorCode+"_ownership", usrOwnershipResp.JSON200)
	}

	// Example: Look up shareholders
	yes := true
	shareholdersResp, err := cl.GetV1ShareholdersContractorCodeWithResponse(ctx, contractorCode, &youscore.GetV1ShareholdersContractorCodeParams{
		AddHistory: &yes,
	})
	if err != nil {
		log.Fatal("ERROR: get shareholders:", err)
	}
	log.Println("Get shareholders status:", shareholdersResp.StatusCode())
	if shareholdersResp.StatusCode() == http.StatusOK {
		//log.Println("Get shareholders:", toJSUtil(shareholdersResp.JSON200))
		dumpJSUtil(contractorCode+"_shareholders", shareholdersResp.JSON200)
	}

	// Example: Look up history
	historyResp, err := cl.GetV1HistoryContractorCodeWithResponse(ctx, contractorCode)
	if err != nil {
		log.Fatal("ERROR: get history:", err)
	}
	log.Println("Get history status:", historyResp.StatusCode())
	if historyResp.StatusCode() == http.StatusOK {
		//log.Println("Get history:", toJSUtil(historyResp.JSON200))
		dumpJSUtil(contractorCode+"_history", historyResp.JSON200)
	}

	// Example: Look up status
	usrStatutResp, err := cl.GetV1UsrDocumentsUsrStatutFileWithResponse(ctx, &youscore.GetV1UsrDocumentsUsrStatutFileParams{
		Code: &contractorCode,
	})
	if err != nil {
		log.Fatal("ERROR: get usr statut:", err)
	}
	log.Println("Get usr statut status:", usrStatutResp.StatusCode())
	if usrStatutResp.StatusCode() == http.StatusOK {
		//log.Println("Get usr statut:", toJSUtil(usrStatutResp.JSON200))
		dumpJSUtil(contractorCode+"_usrStat", usrStatutResp.JSON200)
	}

	// Example: Look up admin services
	usrAdminResp, err := cl.GetV1UsrAdministrativeServicesResultsCodeWithResponse(ctx, contractorCode)
	if err != nil {
		log.Fatal("ERROR: get usr administrative services:", err)
	}
	log.Println("Get usr administrative services status:", usrAdminResp.StatusCode())
	if usrAdminResp.StatusCode() == http.StatusOK {
		//log.Println("Get usr administrative services:", toJSUtil(usrAdminResp.JSON200))
		dumpJSUtil(contractorCode+"_usrAdministrativeServices", usrAdminResp.JSON200)
	}

	// Example: Check express analysis for a company
	expressAnalysisResp, err := cl.GetV1ExpressAnalysisContractorCodeWithResponse(ctx, contractorCode, &youscore.GetV1ExpressAnalysisContractorCodeParams{
		ShowCurrentData: &yes,
		ShowPrompt:      &yes,
	})
	if err != nil {
		log.Fatal("ERROR: get express analysis:", err)
	}
	log.Println("Get express analysis status:", expressAnalysisResp.StatusCode())
	if expressAnalysisResp.StatusCode() == http.StatusOK {
		//log.Println("Get express analysis:", toJSUtil(expressAnalysisResp.JSON200))
		dumpJSUtil(contractorCode+"_expressAnalysis", expressAnalysisResp.JSON200)
	}

	// Example: Check express analysis finmon for a company
	expressAnalysisFinMonResp, err := cl.GetV1ExpressAnalysisFinmonContractorCodeWithResponse(ctx, contractorCode, &youscore.GetV1ExpressAnalysisFinmonContractorCodeParams{
		ShowCurrentData: &yes,
		ShowPrompt:      &yes,
	})
	if err != nil {
		log.Fatal("ERROR: get express analysis finmon:", err)
	}
	log.Println("Get express analysis finmon status:", expressAnalysisFinMonResp.StatusCode())
	if expressAnalysisFinMonResp.StatusCode() == http.StatusOK {
		//log.Println("Get express analysis finmon:", toJSUtil(expressAnalysisFinMonResp.JSON200))
		dumpJSUtil(contractorCode+"_expressAnalysisFinMon", expressAnalysisFinMonResp.JSON200)
	}

	//// Example: Check sanctions for a company
	//sanctionsResp, err := cl.GetV1SanctionsWithResponse(ctx, &youscore.GetV1SanctionsParams{
	//	ContractorCode: &contractorCode,
	//})
	//if err != nil {
	//	log.Fatal("ERROR: get sanctions:", err)
	//}
	//log.Println("Get sanctions status:", sanctionsResp.StatusCode())
	//if sanctionsResp.StatusCode() == http.StatusOK {
	//	log.Println("Get sanctions:", toJSUtil(sanctionsResp.JSON200))
	//	dumpJSUtil(contractorCode+"_sanctions", sanctionsResp.JSON200)
	//}

	// Example: Check rate limits (for all non-blank keys - using the custom utility)
	rateLimitsResp, err := youscore.CheckRateLimits(ctx, apiKeys)
	if err != nil {
		log.Fatal("ERROR: check rate limits:", err)
	}
	log.Println("Rate limits:", toJSUtil(rateLimitsResp))
	dumpJSUtil("rate_limits", rateLimitsResp)
}

// ---------------------------
// --- SILLY CACHE EXAMPLE ---
// ---------------------------

// NOTE: This file-backed JSON cache is a dumb example implementation.
// A realistic application should use a more sophisticated caching mechanism
// (e.g. Redis, SQLite, bbolt, or an in-memory LRU cache with proper eviction).
// Writing the entire cache to a JSON file on every update is computationally
// wasteful and does not scale, but it ensures the cache is persisted even if
// the program exits, errors, or is cancelled.

const (
	cacheFile = "cache.json"
	cacheTTL  = 7 * 24 * time.Hour // cached entries expire after 7 days
)

// dumbFileCache is a map-based cache that persists to a JSON file on every write.
// It implements the youscore.Cache interface.
type dumbFileCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

// newDumbFileCache creates a new dumbFileCache, loading existing entries from the cache file.
func newDumbFileCache() *dumbFileCache {
	c := &dumbFileCache{
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
func (c *dumbFileCache) load() {
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
func (c *dumbFileCache) save() {
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
func (c *dumbFileCache) Get(url string, key string) (youscore.CachedResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		log.Println("cache miss:", key)
		return youscore.CachedResponse{}, false
	}

	if time.Since(entry.Timestamp) > cacheTTL {
		log.Println("cache miss:", key)
		delete(c.entries, key)
		c.save()
		return youscore.CachedResponse{}, false
	}

	log.Println("cache hit:", key)
	return entry.Response, true
}

// Set stores a response in the cache and immediately persists to disk.
func (c *dumbFileCache) Set(url string, key string, resp youscore.CachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// skip 202 responses
	if resp.StatusCode == 202 {
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

// ----------------------------
// --- Simple Usage tracker ---
// ----------------------------

type usage struct {
	PerType map[string]int
	Calls   []string
}

func (u *usage) Print() {
	fmt.Println("Total usage:", toJSUtil(u))
}

func newSimpleUsageTracker(usage *usage) func(ctx context.Context, apiType youscore.APIType, path string) {
	if usage.PerType == nil {
		usage.PerType = make(map[string]int)
	}
	return func(ctx context.Context, apiType youscore.APIType, path string) {
		usage.PerType[string(apiType)]++
		usage.Calls = append(usage.Calls, fmt.Sprintf("%s: %s", apiType, path))
	}
}

// -----------------
// --- UTILITIES ---
// -----------------

func dumpJSUtil(name string, v any) {
	os.WriteFile("debug/"+name+".json", []byte(toJSUtil(v)), os.ModePerm)
}

func toJSUtil(a any) string {
	buf, _ := json.MarshalIndent(a, "", "  ")
	return string(buf)
}

func ptr[T any](v T) *T {
	return &v
}
