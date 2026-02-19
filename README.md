# goyouscore

A Go client library for the [YouScore API](https://youscore.com.ua/), primarily generated from the official [Swagger documentation](https://api.youscore.com.ua/swagger/index.html#/).

## About

This is a community-maintained library with no affiliation to YouScore.

This client is generated from the [Translated and fixed](./spec/swagger_en_processed.json) open-api spec,
using github.com/oapi-codegen/oapi-codegen/v2. 

With a couple of useful additions:
- Automatic API key assignment (different endpoints use different keys, this library handles that)
- Unified utility for fetching rate limits for all keys
- Utility for caching API responses (see below)
- Translated swagger docs to English [Translated](./spec/swagger_en.json)

## Usage

```go
import "github.com/fritzkeyzer/goyouscore"

// Create a client with per-endpoint API key authentication
cl, err := youscore.NewClientWithResponses(youscore.ServerURL,
    youscore.WithAPIKeys(youscore.APIKeys{
        DataAnalytics:    os.Getenv("YOUSCORE_DATA_ANALYTICS_KEY"),
        PDFLegalEntities: os.Getenv("YOUSCORE_PDF_LEGAL_KEY"),
        PDFIndividuals:   os.Getenv("YOUSCORE_PDF_INDIVIDUALS_KEY"),
        Affiliates:       os.Getenv("YOUSCORE_AFFILIATES_KEY"),
    }),
)

// OR - if you want to bring your own cache (preventing duplicate requests)
cl, err := youscore.NewClientWithResponses(youscore.ServerURL,
    youscore.WithAPIKeys(apiKeys),
    youscore.WithCache(customCache), // where customCache is your own implementation of the youscore.Cache interface
)

// Example: Look up registration data (USR) for a company by its EDRPOU code
usrResp, err := cl.GetV1UsrContractorCodeWithResponse(ctx, "08215600", &youscore.GetV1UsrContractorCodeParams{
    ShowCurrentData: ptr(true),
})
if err != nil {
    log.Fatal("ERROR: get USR:", err)
}
log.Println("Get USR status:", usrResp.StatusCode())
if usrResp.StatusCode() == http.StatusOK {
    log.Println("Get USR:", usrResp.JSON200)
}
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/).

## License

[MIT](LICENSE)
