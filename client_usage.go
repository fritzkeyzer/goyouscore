package youscore

import (
	"context"
	"net/http"
	"strings"
)

type APIType string

const (
	APITypeCustom   APIType = "custom"
	APITypeAnalysis APIType = "analysis"
	APITypeData     APIType = "data"
)

func WithUsageTracking(usageFn func(ctx context.Context, apiType APIType, path string)) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
			path := req.URL.Path
			path, _, _ = strings.Cut(path, "?")

			apiType := aPITypeForPath(path)
			usageFn(ctx, apiType, path)

			return nil
		})
		return nil
	}
}

// aPITypeForPath returns the appropriate API type for the given request path.
// This can be used to calculate pricing and implement cost tracking
func aPITypeForPath(path string) APIType {
	path = strings.TrimPrefix(path, "/")

	switch {
	// Custom (PDF reports, affiliates, auctions)
	case strings.HasPrefix(path, "v1/contractors/pdf-file/"),
		strings.HasPrefix(path, "v1/contractorsPdf/"),
		strings.HasPrefix(path, "v1/individuals/pdf-reports"),
		strings.HasPrefix(path, "v1/individualsPdfReports"),
		strings.HasPrefix(path, "v1/affiliates"),
		strings.HasPrefix(path, "v1/setam/"):
		return APITypeCustom

	// Analytics
	case strings.HasPrefix(path, "v1/history/"),
		strings.HasPrefix(path, "v1/usrAdministrativeServicesResults/"),
		strings.HasPrefix(path, "v1/usrDocuments/"),
		strings.HasPrefix(path, "v1/expressAnalysis/"),
		strings.HasPrefix(path, "v1/marketScoring/"),
		strings.HasPrefix(path, "v1/financialScoring/"),
		strings.HasPrefix(path, "v1/investigationsLegal"),
		strings.HasPrefix(path, "v1/investigationsNatural"),
		strings.HasPrefix(path, "v1/fig"),
		strings.HasPrefix(path, "v1/individualsFigCompanies"),
		strings.HasPrefix(path, "v1/courtCaseGroup/"),
		strings.HasPrefix(path, "v1/encumbrances/details/"),
		strings.HasPrefix(path, "v1/encumbrances/resultdetails/"),
		strings.HasPrefix(path, "v1/realEstate/details/"),
		strings.HasPrefix(path, "v1/realEstate/resultdetails/"),
		strings.HasPrefix(path, "v1/tenders/risks/"),
		strings.HasPrefix(path, "v1/sanctions"):
		return APITypeAnalysis
	}

	// Default: DATA (covers all remaining endpoints)
	return APITypeData
}
