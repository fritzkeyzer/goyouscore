package youscore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Cache defines the interface for caching HTTP responses.
// Implementations can use any backing store (in-memory, DB, Redis, disk, etc.).
// The implementation controls cache TTLs, which endpoints to cache, etc.
type Cache interface {
	// Get retrieves a cached response for the given cache key.
	// url is the raw request URL, provided for custom per-route handling (e.g. different TTLs).
	// key is a hash derived from the full request (method, URL, headers, body).
	// If the key is not found, ok must be false.
	Get(url string, key string) (resp CachedResponse, ok bool)

	// Set stores a response for the given cache key.
	// url is the raw request URL, provided for custom per-route handling.
	// key is a hash derived from the full request (method, URL, headers, body).
	Set(url string, key string, resp CachedResponse)
}

// CachedResponse holds the data needed to reconstruct an HTTP response from cache.
type CachedResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// WithCache returns a ClientOption that wraps the underlying HTTP client
// with a caching layer. All requests are passed to the Cache implementation,
// which decides what to cache via its Get/Set methods.
func WithCache(cache Cache) ClientOption {
	return func(c *Client) error {
		inner := c.Client
		if inner == nil {
			inner = &http.Client{}
		}
		c.Client = &cachingDoer{
			inner: inner,
			cache: cache,
		}
		return nil
	}
}

type cachingDoer struct {
	inner HttpRequestDoer
	cache Cache
}

// cacheKey builds a deterministic hash from method, URL, and body.
func cacheKey(req *http.Request) (string, []byte, error) {
	h := sha256.New()
	h.Write([]byte(req.Method))
	h.Write([]byte(req.URL.String()))

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return "", nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		h.Write(bodyBytes)
	}

	return hex.EncodeToString(h.Sum(nil)), bodyBytes, nil
}

// sensitiveQueryParams lists query parameter names that may contain secrets
// and must be stripped from URLs before passing to the Cache interface.
var sensitiveQueryParams = []string{"apikey", "api_key", "api-key", "token", "access_token", "authorization"}

// sanitizeURL removes sensitive query parameters (e.g. API keys) from a URL
// so that secrets are not leaked to Cache implementations.
func sanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	for param := range q {
		if isSensitiveParam(param) {
			q.Del(param)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func isSensitiveParam(name string) bool {
	lower := strings.ToLower(name)
	for _, s := range sensitiveQueryParams {
		if lower == s {
			return true
		}
	}
	return false
}

func (d *cachingDoer) Do(req *http.Request) (*http.Response, error) {
	key, _, err := cacheKey(req)
	if err != nil {
		return nil, err
	}

	rawURL := sanitizeURL(req.URL.String())

	if cached, ok := d.cache.Get(rawURL, key); ok {
		return &http.Response{
			StatusCode: cached.StatusCode,
			Header:     cached.Header,
			Body:       io.NopCloser(bytes.NewReader(cached.Body)),
			Request:    req,
		}, nil
	}

	resp, err := d.inner.Do(req)
	if err != nil {
		return resp, err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return resp, err
	}

	skipCache := strings.Contains(req.URL.String(), "/rateLimits")

	if !skipCache {
		d.cache.Set(sanitizeURL(req.URL.String()), key, CachedResponse{
			StatusCode: resp.StatusCode,
			Header:     resp.Header.Clone(),
			Body:       bytes.Clone(body),
		})
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}
