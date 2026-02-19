package youscore

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// mapCache is a simple in-memory cache for testing.
type mapCache struct {
	mu    sync.Mutex
	store map[string]CachedResponse
}

func newMapCache() *mapCache {
	return &mapCache{store: make(map[string]CachedResponse)}
}

func (c *mapCache) Get(_ string, key string) (CachedResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.store[key]
	return v, ok
}

func (c *mapCache) Set(_ string, key string, resp CachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = resp
}

// fakeDoer records how many times Do was called.
type fakeDoer struct {
	calls int
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.calls++
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": {"value"}},
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Request:    req,
	}, nil
}

func TestWithCache_GetRequestCached(t *testing.T) {
	fake := &fakeDoer{}
	cache := newMapCache()

	cl, err := NewClientWithResponses(ServerURL,
		WithHTTPClient(fake),
		WithCache(cache),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()

	// First call should hit the backend
	_, err = cl.GetV1AffiliatesResultIdWithResponse(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected 1 call, got %d", fake.calls)
	}

	// Second call should be served from cache
	_, err = cl.GetV1AffiliatesResultIdWithResponse(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected still 1 call after cache hit, got %d", fake.calls)
	}
}

func TestWithCache_GetRateLimitsCached(t *testing.T) {
	fake := &fakeDoer{}
	cache := newMapCache()

	cl, err := NewClientWithResponses(ServerURL,
		WithHTTPClient(fake),
		WithCache(cache),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()

	// First call should hit the backend (not cached)
	_, err = cl.GetV1RateLimitsWithResponse(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected 1 call, got %d", fake.calls)
	}

	// Second call should also hit the backend (not cached)
	_, err = cl.GetV1RateLimitsWithResponse(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 2 {
		t.Fatalf("expected 2 calls after rate limit calls, got %d", fake.calls)
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no params", "https://api.example.com/v1/data", "https://api.example.com/v1/data"},
		{"apiKey removed", "https://api.example.com/v1/data?apiKey=secret123&page=1", "https://api.example.com/v1/data?page=1"},
		{"apiKey case insensitive", "https://api.example.com/v1/data?ApiKey=secret", "https://api.example.com/v1/data"},
		{"api_key removed", "https://api.example.com/v1/data?api_key=secret", "https://api.example.com/v1/data"},
		{"api-key removed", "https://api.example.com/v1/data?api-key=secret", "https://api.example.com/v1/data"},
		{"token removed", "https://api.example.com/v1/data?token=abc", "https://api.example.com/v1/data"},
		{"access_token removed", "https://api.example.com/v1/data?access_token=abc", "https://api.example.com/v1/data"},
		{"authorization removed", "https://api.example.com/v1/data?authorization=bearer+abc", "https://api.example.com/v1/data"},
		{"safe params kept", "https://api.example.com/v1/data?page=1&limit=10", "https://api.example.com/v1/data?limit=10&page=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeURL(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestWithCache_PostRequestCached(t *testing.T) {
	fake := &fakeDoer{}
	cache := newMapCache()

	cl, err := NewClientWithResponses(ServerURL,
		WithHTTPClient(fake),
		WithCache(cache),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()

	// Same POST body twice — second should be cached
	_, err = cl.PostV1AffiliatesQueryWithResponse(ctx, PostV1AffiliatesQueryJSONRequestBody{})
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected 1 call, got %d", fake.calls)
	}

	_, err = cl.PostV1AffiliatesQueryWithResponse(ctx, PostV1AffiliatesQueryJSONRequestBody{})
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected still 1 call after cache hit, got %d", fake.calls)
	}
}
