# Service-Feed Integration Setup

## Prerequisites

For private GitHub repositories, configure git to use SSH:

```bash
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

Set GOPRIVATE for private modules:

```bash
export GOPRIVATE=github.com/regentmarkets
```

## Build

This document describes the integration between `service-pricer-digitalcallput` and `service-feed`.

## Overview

The service-pricer-digitalcallput integrates with the real-time market data feed provided by `service-feed` (as specified in Architecture Section 6).

## Current Implementation

### Real gRPC Client

The [`internal/feed/client.go`](internal/feed/client.go:1) implements a gRPC client that connects to service-feed using the TickServiceClient:

- **Repository**: `github.com/regentmarkets/service-feed v0.1.2`
- **API Package**: `github.com/regentmarkets/service-feed/api`
- **Client Type**: `feedapi.TickServiceClient`
- **Methods**:
  - `Subscribe(ctx, symbol)` - Streams market ticks for a symbol via `StreamTicks` RPC
  - `GetCurrentTick(ctx, symbol)` - Retrieves current tick for a symbol
  - `Close()` - Closes the gRPC connection

### Implementation Details

The client uses the gRPC streaming API:

```go
stream, err := c.client.StreamTicks(ctx, &feedapi.StreamTicksRequest{
    Symbol: symbol,
    Time:   timestamppb.Now(),
})
```

Each response contains a batch of ticks:
```go
resp, err := stream.Recv()
for _, tick := range resp.Ticks {
    // Process tick
}
```

### Field Mapping

The service-feed tick format is mapped to the internal pricing model:

| service-feed | pricing.Tick | Description |
|-------------|--------------|-------------|
| `tick.Symbol` | `Symbol` | Trading symbol |
| `tick.Quote` | `Price` | Market price (string parsed to float64) |
| `tick.Time.AsTime()` | `Timestamp` | Timestamp |

### Configuration

The feed address can be configured via:

1. **Command-line flag**: `--feed-address`
2. **Environment variable**: `FEED_ADDRESS`
3. **Default value**: `localhost:50052`

Example:
```bash
# Using environment variable
export FEED_ADDRESS=service-feed.example.com:50052
./digitalcallput

# Using command-line flag
./digitalcallput --feed-address=service-feed.example.com:50052
```

## Development Setup

### Local Module Reference

During development, the service uses a local replace directive in `go.mod`:

```go
replace github.com/regentmarkets/service-feed => ../../../../feed/service-feed
```

This allows developing against the local service-feed repository.

### Private Module Access

Since `service-feed` is a private repository, configure Git to use SSH for GitHub:

```bash
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

## Testing

### Mock Client for Tests

A separate mock implementation is available in [`internal/feed/client_mock.go`](internal/feed/client_mock.go:1) for unit testing purposes.

To use the mock in tests:
```go
mockClient, err := feed.NewMockClient("localhost:50052")
subscriber := feed.NewSubscriber(mockClient)
```

The mock provides:
- Fixed mock price (1.08523) with random variations
- 1-second tick intervals
- Proper context cancellation handling

### Integration Testing

To test with a running service-feed:

```bash
# Build and run the service
cd workspace/code/service-pricer-digitalcallput
go build ./cmd/digitalcallput
./digitalcallput --feed-address=localhost:50052
```

The service will connect to service-feed and receive real-time tick data.

## Architecture Reference

This implementation follows the architecture specification:
- **Section 5.5**: Interface defined at consumer side (FeedSubscriber interface)
- **Section 6**: Service-feed integration details
- **Proto**: Based on service-feed's `proto/grpcfeed/v1/ticks.proto`

### Reference Implementation

This implementation follows the pattern used in `service-ohlc`:

```go
import feedapi "github.com/regentmarkets/service-feed/api"

// Client uses TickServiceClient
client feedapi.TickServiceClient

// StreamTicks method
stream, err := client.StreamTicks(ctx, &feedapi.StreamTicksRequest{
    Symbol: symbol,
    Time:   timestamppb.New(startTime),
})

// Reading ticks
resp, err := stream.Recv()
for _, tick := range resp.Ticks {
    // tick.Time.AsTime() - timestamp
    // tick.Quote - price string
}
```

## Files Modified

1. [`cmd/digitalcallput/main.go`](cmd/digitalcallput/main.go:1) - Added feed-address configuration
2. [`internal/app/app.go`](internal/app/app.go:1) - Configured to use FeedAddress from config
3. [`internal/feed/client.go`](internal/feed/client.go:1) - Implemented real gRPC TickServiceClient
4. [`internal/feed/client_mock.go`](internal/feed/client_mock.go:1) - Mock implementation for testing
5. [`internal/feed/subscriber.go`](internal/feed/subscriber.go:1) - Manages feed subscriptions with fallback
6. [`go.mod`](go.mod:1) - Added service-feed dependency with local replace directive

## Troubleshooting

### Build Errors

If you encounter build errors related to service-feed:

1. Verify the local service-feed repository exists: `ls ../../../../feed/service-feed`
2. Check the replace directive in go.mod points to the correct path
3. Run `go mod tidy` to update dependencies
4. Ensure service-feed has been built: `cd ../../../../feed/service-feed && make generate`

### Connection Errors

If the service fails to connect to service-feed:

1. Verify service-feed is running on the configured address
2. Check network connectivity: `telnet localhost 50052`
3. Review logs for detailed error messages
4. Ensure service-feed is exposing the TickService

### Field Mapping Issues

If tick data appears incorrect:

1. Verify service-feed is returning data in the expected format
2. Check field mappings in [`internal/feed/client.go`](internal/feed/client.go:1)
3. Ensure timestamp conversion is working correctly
4. Check quote parsing from string to float64

### Quote Parsing

The `Quote` field from service-feed is a string. If parsing fails:

1. Check the quote format returned by service-feed
2. Update the `parseQuote()` function if needed
3. Consider adding error handling for invalid quote formats
