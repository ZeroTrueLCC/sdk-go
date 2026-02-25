# ZeroTrue Go SDK

Go SDK for the [ZeroTrue API](https://zerotrue.com) — AI-generated content detection service.

## Installation

```bash
go get github.com/zerotrue/sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    zerotrue "github.com/zerotrue/sdk-go"
)

func main() {
    client, err := zerotrue.NewClient("zt_your_api_key",
        zerotrue.WithBaseURL("https://api.zerotrue.com"),
    )
    if err != nil {
        log.Fatal(err)
    }

    result, err := client.AnalyzeText(context.Background(), "Text to check", nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("AI Probability: %.2f%%\n", result.AIProbability*100)
    fmt.Printf("Model: %s\n", result.MLModel)
}
```

## Features

- **Sync Analysis** — `AnalyzeFile`, `AnalyzeText`, `AnalyzeURL` via API Gateway
- **Async Flow** — `CreateCheck` + `GetCheck` with polling
- **WebSocket** — `WaitForResult` for real-time results
- **Typed Errors** — `AuthenticationError`, `RateLimitError`, `InsufficientCreditsError`, etc.
- **Retry Logic** — Automatic retry with exponential backoff for 5xx and 429 errors
- **Functional Options** — Flexible client configuration

## Client Options

```go
client, _ := zerotrue.NewClient("zt_your_api_key",
    zerotrue.WithBaseURL("https://api.zerotrue.com"),
    zerotrue.WithTimeout(2 * time.Minute),
    zerotrue.WithMaxRetries(5),
    zerotrue.WithRetryWaitMin(500 * time.Millisecond),
    zerotrue.WithRetryWaitMax(1 * time.Minute),
    zerotrue.WithHTTPClient(customHTTPClient),
)
```

## Sync Analysis (via Gateway)

```go
// Analyze text
result, err := client.AnalyzeText(ctx, "Some text", &zerotrue.AnalyzeOptions{
    IsDeepScan:    false,
    IsPrivateScan: true,
})

// Analyze file
result, err := client.AnalyzeFile(ctx, "photo.jpg", nil)

// Analyze URL
result, err := client.AnalyzeURL(ctx, "https://example.com/image.png", nil)
```

## Async Flow (via Backend)

```go
// Create check
check, err := client.CreateCheck(ctx, zerotrue.CheckInput{
    Type:  "text",
    Value: "Text to analyze",
}, &zerotrue.CheckOptions{
    IdempotencyKey: "unique-key-123",
})
// check.ID = "uuid", check.Status = "queued"

// Poll for result
result, err := client.GetCheck(ctx, check.ID)

// Or wait via WebSocket
analysisResult, err := client.WaitForResult(ctx, check.ID)
```

## Retrieve Previous Result

```go
result, err := client.GetResult(ctx, "content-id-uuid")
```

## API Information

```go
info, err := client.GetInfo(ctx)
fmt.Println(info.Name, info.Version)
```

## Error Handling

```go
import "errors"

result, err := client.AnalyzeText(ctx, text, nil)
if err != nil {
    var authErr *zerotrue.AuthenticationError
    var rateLimitErr *zerotrue.RateLimitError
    var creditsErr *zerotrue.InsufficientCreditsError
    var notFoundErr *zerotrue.NotFoundError

    switch {
    case errors.As(err, &authErr):
        log.Fatal("Invalid API key")
    case errors.As(err, &rateLimitErr):
        log.Println("Rate limited, retry later")
    case errors.As(err, &creditsErr):
        log.Fatal("Insufficient credits")
    case errors.As(err, &notFoundErr):
        log.Fatal("Not found")
    default:
        var apiErr *zerotrue.APIError
        if errors.As(err, &apiErr) {
            log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
        }
    }
}
```

## Result Fields

```go
result.AIProbability       // float64: 0.0-1.0
result.HumanProbability    // float64: 0.0-1.0
result.CombinedProbability // float64: 0.0-1.0
result.ResultType          // string: "text_analysis", etc.
result.MLModel             // string: model name
result.SuspectedModels     // []SuspectedModel: {ModelName, ConfidencePct}
result.Segments            // []Segment: content segments with labels
result.InferenceTimeMs     // *int: processing time in milliseconds
```

## License

MIT
