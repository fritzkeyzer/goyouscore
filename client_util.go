package youscore

import (
	"context"
	"net/http"
	"strings"
)

const ServerURL = "https://api.youscore.com.ua"

// APIKeys holds the four API keys for different endpoint categories.
type APIKeys struct {
	// DataAnalytics is used for data and analytics endpoints (default for most endpoints).
	DataAnalytics string
	// PDFLegalEntities is used for the legal entity PDF report endpoint (/v1/contractors/pdf-file/).
	PDFLegalEntities string
	// PDFIndividuals is used for the individual PDF report endpoints (/v1/individuals/pdf-reports).
	PDFIndividuals string
	// Affiliates is used for the affiliates endpoints (/v1/affiliates).
	Affiliates string
}

// WithBearerAuth returns a ClientOption that sets the Authorization header
// with the given API key on every request.
func WithBearerAuth(apiKey string) ClientOption {
	return WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "bearer "+apiKey)
		return nil
	})
}

// WithAPIKeys returns a ClientOption that selects the correct API key based on
// the request path. The four key categories are:
//   - DataAnalytics: used for all data and analytics endpoints (default)
//   - PDFLegalEntities: used for /v1/contractors/pdf-file/ endpoints
//   - PDFIndividuals: used for /v1/individuals/pdf-reports endpoints
//   - Affiliates: used for /v1/affiliates endpoints
func WithAPIKeys(keys APIKeys) ClientOption {
	return WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		key := apiKeyForPath(keys, req.URL.Path)
		req.Header.Set("Authorization", "bearer "+key)
		return nil
	})
}

// apiKeyForPath returns the appropriate API key for the given request path.
func apiKeyForPath(keys APIKeys, path string) string {
	path = strings.TrimPrefix(path, "/")

	switch {
	case strings.HasPrefix(path, "v1/contractors/pdf-file/"):
		return keys.PDFLegalEntities
	case strings.HasPrefix(path, "v1/individuals/pdf-reports"):
		return keys.PDFIndividuals
	case strings.HasPrefix(path, "v1/affiliates"):
		return keys.Affiliates
	default:
		return keys.DataAnalytics
	}
}
