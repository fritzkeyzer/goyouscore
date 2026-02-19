package youscore

import "testing"

func TestApiKeyForPath(t *testing.T) {
	keys := APIKeys{
		DataAnalytics:    "da-key",
		PDFLegalEntities: "pdf-legal-key",
		PDFIndividuals:   "pdf-ind-key",
		Affiliates:       "aff-key",
	}

	tests := []struct {
		path string
		want string
	}{
		{"/v1/usr/00032112", "da-key"},
		{"/v1/sanctions", "da-key"},
		{"/v1/rate-limits", "da-key"},
		{"/v1/court/00032112", "da-key"},
		{"/v1/contractors/pdf-file/00032112", "pdf-legal-key"},
		{"/v1/contractors/pdf-file/12345678", "pdf-legal-key"},
		{"/v1/individuals/pdf-reports", "pdf-ind-key"},
		{"/v1/individuals/pdf-reports/some-result-id", "pdf-ind-key"},
		{"/v1/affiliates/query", "aff-key"},
		{"/v1/affiliates/some-id", "aff-key"},
		// Other /v1/individuals/ endpoints use data+analytics
		{"/v1/individuals/full-name-info", "da-key"},
		{"/v1/individuals/related-persons", "da-key"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := apiKeyForPath(keys, tt.path)
			if got != tt.want {
				t.Errorf("apiKeyForPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
