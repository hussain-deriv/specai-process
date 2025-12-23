# API Specification: Digital Call/Put Options Pricing Service
# Public gRPC API

**Document Version**: 1.0
**Last Updated**: 2025-12-23
**Service Name**: digitalcallput
**Protocol**: gRPC (proto3)

---

## 1. Overview

The Digital Call/Put Options Pricing Service provides a gRPC API for calculating real-time prices for binary options contracts. The API supports both synchronous (unary) and streaming operations for ask and bid price calculations.

### Target Consumers
- Trading platforms
- Automated trading systems
- Financial service integrators
- Internal trading services

### API Characteristics
| Aspect | Value |
|--------|-------|
| Protocol | gRPC over HTTP/2 |
| Serialization | Protocol Buffers (proto3) |
| Authentication | None (service-level security) |
| Versioning | Package versioning (v1) |

---

## 2. Base Configuration

### Service Definition
```
Package: digitalcallput.v1
Service: PricingService
Proto File: digitalcallput/v1/pricing.proto
```

### Connection Details
| Environment | Endpoint |
|-------------|----------|
| Development | `localhost:50051` |
| Staging | `digitalcallput-staging.internal:50051` |
| Production | `digitalcallput.internal:50051` |

### Protocol Settings
- **Max Message Size**: 4MB (default)
- **Keep-Alive**: Enabled (30s interval)
- **TLS**: Required in production

---

## 3. Authentication & Authorization

This service operates without user-level authentication. Security is handled at the service/network level:

- **Network Security**: Service mesh / VPN / internal network only
- **TLS**: Required for all production traffic
- **No API Keys**: No user authentication required
- **Rate Limiting**: Applied at load balancer level

---

## 4. Table of Endpoints

| ID | Method | RPC Type | Path | Summary |
|----|--------|----------|------|---------|
| API-DC-A1K | GetAsk | Unary | `digitalcallput.v1.PricingService/GetAsk` | Calculate single ask price |
| API-DC-B2L | StreamAsk | Server Streaming | `digitalcallput.v1.PricingService/StreamAsk` | Stream continuous ask prices |
| API-DC-C3M | GetBid | Unary | `digitalcallput.v1.PricingService/GetBid` | Calculate single bid price |
| API-DC-D4N | StreamBid | Server Streaming | `digitalcallput.v1.PricingService/StreamBid` | Stream continuous bid prices |

---

## 5. Endpoints & Methods

### 5.1 GetAsk (API-DC-A1K)

**Purpose**: Calculate a single ask price for a digital option proposal

**RPC Definition**:
```protobuf
rpc GetAsk(GetAskRequest) returns (GetAskResponse);
```

**Request**: `GetAskRequest`
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| option_parameters | OptionParameters | Yes | Option contract specification |
| pricing_time | int64 | No | Epoch timestamp for pricing (defaults to current time) |

**Response**: `GetAskResponse`
| Field | Type | Description |
|-------|------|-------------|
| ask_price | string | Price to purchase the contract (decimal string) |
| currency | string | Contract currency |
| current_spot | string | Current market spot price |
| current_spot_time | int64 | Timestamp of spot price (epoch) |
| payout | string | Potential payout if contract wins |
| limits | Limits | Applicable trading limits |

**Business Rules**:
- Ask price equals the stake amount
- Payout is calculated using Black-Scholes with 2% commission (hidden)
- Validates stake against minimum stake requirement

**Error Responses**:
| gRPC Code | Condition | Example Message |
|-----------|-----------|-----------------|
| INVALID_ARGUMENT | Invalid symbol | "Invalid symbol: INVALID_SYM" |
| INVALID_ARGUMENT | Invalid duration | "Invalid duration format: 5x" |
| INVALID_ARGUMENT | Invalid currency | "Invalid currency: XXX" |
| INVALID_ARGUMENT | Invalid barrier | "Invalid barrier format: abc" |
| FAILED_PRECONDITION | Stake too low | "Stake below minimum: 1.00" |
| OUT_OF_RANGE | Negative stake | "Stake must be positive" |
| UNAVAILABLE | Feed down | "Market data feed unavailable" |

**Example Request**:
```json
{
  "option_parameters": {
    "symbol": "EUR/USD",
    "contract_type": "CONTRACT_TYPE_CALL",
    "currency": "USD",
    "stake": "100.00",
    "duration": "5m",
    "barrier": "+50"
  }
}
```

**Example Response**:
```json
{
  "ask_price": "100.00",
  "currency": "USD",
  "current_spot": "1.08523",
  "current_spot_time": 1703318400,
  "payout": "196.00",
  "limits": {
    "max_payout": "50000.00",
    "min_stake": "1.00"
  }
}
```

**PRD Reference**: [Section 4.1.1](../requirements/prd.md:175)

---

### 5.2 StreamAsk (API-DC-B2L)

**Purpose**: Stream continuous ask price updates for a digital option proposal

**RPC Definition**:
```protobuf
rpc StreamAsk(GetAskRequest) returns (stream GetAskResponse);
```

**Request**: Same as GetAsk (`GetAskRequest`)

**Response Stream**: Continuous stream of `GetAskResponse` messages

**Stream Behavior**:
| Aspect | Behavior |
|--------|----------|
| Update Trigger | On every new tick from market feed |
| Fallback Update | Every 5 seconds if no ticks received |
| Termination | Client closes connection |
| Auto-Terminate | No |

**Business Rules**:
- Same pricing logic as GetAsk
- Each response reflects current market conditions
- Stream continues indefinitely until client disconnects

**Error Responses**: Same as GetAsk (errors terminate stream)

**PRD Reference**: [Section 4.1.2](../requirements/prd.md:222)

---

### 5.3 GetBid (API-DC-C3M)

**Purpose**: Calculate a single bid price for an active contract

**RPC Definition**:
```protobuf
rpc GetBid(GetBidRequest) returns (GetBidResponse);
```

**Request**: `GetBidRequest`
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| option_parameters | OptionParameters | Yes | Option contract specification (must include start_time) |
| pricing_time | int64 | No | Epoch timestamp for pricing (defaults to current time) |

**Response**: `GetBidResponse`
| Field | Type | Description |
|-------|------|-------------|
| bid_price | string | Current market value of contract |
| is_expired | bool | Whether contract has expired |
| current_spot | string | Current market spot price |
| current_spot_time | int64 | Timestamp of current spot |
| entry_spot | string | Contract entry price |
| entry_spot_time | int64 | Timestamp of entry |
| exit_spot | string | Contract exit price (if expired) |
| exit_spot_time | int64 | Timestamp of exit (if expired) |
| barrier | string | Calculated barrier value |
| start_time | int64 | Contract start time |
| expiry_time | int64 | Contract expiry time |
| currency | string | Contract currency |

**Business Rules**:
- `start_time` is required in `option_parameters`
- Entry spot is first tick after start_time
- Expiry calculation differs for time-based vs tick-based durations:
  - Time-based: expiry_time = start_time + duration
  - Tick-based: Expires after N ticks received
- If expired, bid_price = payout (win) or 0 (loss)
- Win condition: CALL wins if exit_spot > barrier; PUT wins if exit_spot < barrier

**Error Responses**:
| gRPC Code | Condition | Example Message |
|-----------|-----------|-----------------|
| INVALID_ARGUMENT | Missing start_time | "Start time required for bid requests" |
| INVALID_ARGUMENT | Future start_time | "Start time must be in the past" |
| (All errors from GetAsk also apply) | | |

**Example Request**:
```json
{
  "option_parameters": {
    "symbol": "EUR/USD",
    "contract_type": "CONTRACT_TYPE_CALL",
    "currency": "USD",
    "stake": "100.00",
    "duration": "5m",
    "barrier": "+50",
    "start_time": 1703318400
  }
}
```

**Example Response (Active Contract)**:
```json
{
  "bid_price": "145.50",
  "is_expired": false,
  "current_spot": "1.08600",
  "current_spot_time": 1703318520,
  "entry_spot": "1.08523",
  "entry_spot_time": 1703318401,
  "barrier": "1.08573",
  "start_time": 1703318400,
  "expiry_time": 1703318700,
  "currency": "USD"
}
```

**Example Response (Expired - Win)**:
```json
{
  "bid_price": "196.00",
  "is_expired": true,
  "current_spot": "1.08650",
  "current_spot_time": 1703318705,
  "entry_spot": "1.08523",
  "entry_spot_time": 1703318401,
  "exit_spot": "1.08650",
  "exit_spot_time": 1703318700,
  "barrier": "1.08573",
  "start_time": 1703318400,
  "expiry_time": 1703318700,
  "currency": "USD"
}
```

**PRD Reference**: [Section 4.1.3](../requirements/prd.md:238)

---

### 5.4 StreamBid (API-DC-D4N)

**Purpose**: Stream continuous bid price updates for an active contract

**RPC Definition**:
```protobuf
rpc StreamBid(GetBidRequest) returns (stream GetBidResponse);
```

**Request**: Same as GetBid (`GetBidRequest`)

**Response Stream**: Continuous stream of `GetBidResponse` messages

**Stream Behavior**:
| Aspect | Time-Based Duration | Tick-Based Duration |
|--------|---------------------|---------------------|
| Update Trigger | On every new tick | On every new tick |
| Fallback Update | Every 5 seconds | None (tick arrivals only) |
| Termination | Client closes OR contract expires | Client closes OR contract expires |
| Auto-Terminate | Yes, on expiry | Yes, on expiry |
| Final Message | Includes is_expired=true, exit_spot | Includes is_expired=true, exit_spot |

**Critical Note**: Tick-based durations (e.g., "10t") have NO time-based fallback. The contract remains active until the specified number of ticks are received.

**Business Rules**:
- Same pricing logic as GetBid
- Stream automatically terminates when contract expires
- Final message contains settlement information

**PRD Reference**: [Section 4.1.4](../requirements/prd.md:275)

---

## 6. Data Models & Schemas

### 6.1 OptionParameters
```protobuf
message OptionParameters {
  string symbol = 1;              // Required: Underlying asset (e.g., "EUR/USD")
  ContractType contract_type = 2; // Required: CALL or PUT
  string currency = 3;            // Required: Payout currency (e.g., "USD")
  string duration = 4;            // Required: Contract duration (e.g., "5m", "10t")
  optional string barrier = 5;    // Optional: Relative ("+50") or absolute ("1.2345")
  optional int64 start_time = 6;  // Required for bid requests: Contract start timestamp
  string stake = 7;               // Required: Premium amount (decimal string)
}
```

**Validation Rules**:
| Field | Validation | Error Code |
|-------|------------|------------|
| symbol | Non-empty, exists in market feed | INVALID_ARGUMENT |
| contract_type | Must be CALL or PUT | INVALID_ARGUMENT |
| currency | Valid currency code | INVALID_ARGUMENT |
| duration | Format: `^\d+[smhdt]$` | INVALID_ARGUMENT |
| barrier | Optional; if present: `^[+-]?\d+\.?\d*$` | INVALID_ARGUMENT |
| start_time | Required for bid; must be past | INVALID_ARGUMENT |
| stake | Positive decimal; >= min_stake | OUT_OF_RANGE / FAILED_PRECONDITION |

### 6.2 ContractType
```protobuf
enum ContractType {
  CONTRACT_TYPE_UNSPECIFIED = 0;
  CONTRACT_TYPE_CALL = 1;
  CONTRACT_TYPE_PUT = 2;
}
```

### 6.3 GetAskRequest
```protobuf
message GetAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}
```

### 6.4 GetAskResponse
```protobuf
message GetAskResponse {
  string ask_price = 1;
  string currency = 2;
  string current_spot = 3;
  int64 current_spot_time = 4;
  string payout = 5;
  Limits limits = 6;
}
```

### 6.5 GetBidRequest
```protobuf
message GetBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}
```

### 6.6 GetBidResponse
```protobuf
message GetBidResponse {
  string bid_price = 1;
  bool is_expired = 2;
  string current_spot = 3;
  int64 current_spot_time = 4;
  string entry_spot = 5;
  int64 entry_spot_time = 6;
  string exit_spot = 7;
  int64 exit_spot_time = 8;
  string barrier = 9;
  int64 start_time = 10;
  int64 expiry_time = 11;
  string currency = 12;
}
```

### 6.7 Limits
```protobuf
message Limits {
  string max_payout = 1;
  string min_stake = 2;
}
```

---

## 7. Error Handling

### 7.1 Error Response Format

All errors follow standard gRPC error conventions:

```protobuf
// Standard gRPC Status
message Status {
  int32 code = 1;      // gRPC status code
  string message = 2;  // Human-readable message
  repeated Any details = 3; // Optional error details
}
```

### 7.2 Error Code Catalog

| ID | gRPC Code | HTTP Equivalent | Condition |
|----|-----------|-----------------|-----------|
| ERR-DC-A1K | INVALID_ARGUMENT (3) | 400 | Invalid symbol |
| ERR-DC-B2L | INVALID_ARGUMENT (3) | 400 | Invalid duration format |
| ERR-DC-C3M | INVALID_ARGUMENT (3) | 400 | Invalid currency |
| ERR-DC-D4N | INVALID_ARGUMENT (3) | 400 | Invalid barrier format |
| ERR-DC-E5P | INVALID_ARGUMENT (3) | 400 | Invalid contract type |
| ERR-DC-F6Q | INVALID_ARGUMENT (3) | 400 | Missing start_time (bid) |
| ERR-DC-G7R | INVALID_ARGUMENT (3) | 400 | Future start_time (bid) |
| ERR-DC-H8S | FAILED_PRECONDITION (9) | 400 | Stake below minimum |
| ERR-DC-I9T | OUT_OF_RANGE (11) | 400 | Negative stake |
| ERR-DC-J1U | UNAVAILABLE (14) | 503 | Market data feed unavailable |
| ERR-DC-K2V | INTERNAL (13) | 500 | Unexpected server error |

### 7.3 Error Message Format

Error messages follow this pattern:
```
{Field/Component}: {Specific issue}. {Expected format or guidance}
```

**Examples**:
- `"Invalid symbol: INVALID_SYM"`
- `"Invalid duration format: 5x. Expected format: \\d+[smhdt]"`
- `"Stake below minimum: 1.00"`
- `"Market data feed unavailable"`

---

## 8. Duration Format Reference

| Format | Example | Description |
|--------|---------|-------------|
| Seconds | `30s` | 30 seconds |
| Minutes | `5m` | 5 minutes (300 seconds) |
| Hours | `2h` | 2 hours (7200 seconds) |
| Days | `1d` | 1 day (86400 seconds) |
| Ticks | `10t` | 10 market ticks |

**Important**: Tick-based durations (`t`) behave differently:
- No time-based expiry
- Contract expires ONLY after N ticks received
- No 5-second fallback for stream updates

---

## 9. Barrier Format Reference

| Type | Format | Example | Calculation |
|------|--------|---------|-------------|
| Relative Positive | `+{digits}` | `+50` | entry_spot + value (pips) |
| Relative Negative | `-{digits}` | `-100` | entry_spot - value (pips) |
| Absolute | `{decimal}` | `1.08550` | Exact barrier value |
| None | omit field | - | barrier = entry_spot |

**Notes**:
- No range limits on relative barriers
- Absolute barriers must be positive
- Barrier is resolved when contract starts (using entry_spot)

---

## 10. Integration Guide

### 10.1 Getting Started

1. **Obtain Proto Files**:
   ```bash
   # Clone or copy the proto file
   cp digitalcallput/v1/pricing.proto ./proto/
   ```

2. **Generate Client Code**:
   ```bash
   # For Go
   protoc --go_out=. --go-grpc_out=. proto/pricing.proto
   
   # For Python
   python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. proto/pricing.proto
   ```

3. **Connect to Service**:
   ```go
   // Go example
   conn, err := grpc.Dial("digitalcallput.internal:50051", grpc.WithInsecure())
   client := digitalcallputv1.NewPricingServiceClient(conn)
   ```

### 10.2 Best Practices

1. **Connection Management**:
   - Reuse connections (expensive to create)
   - Enable keep-alive for long-lived connections
   - Handle reconnection gracefully

2. **Stream Handling**:
   - Always handle stream errors
   - Implement proper cancellation (context)
   - Process messages asynchronously if high volume

3. **Error Handling**:
   - Check gRPC status codes
   - Parse error messages for actionable information
   - Implement retry logic for UNAVAILABLE errors

### 10.3 Example: Complete Flow

```go
package main

import (
    "context"
    "log"
    
    pb "github.com/regentmarkets/service-pricer-digitalcallput/proto/digitalcallput/v1"
    "google.golang.org/grpc"
)

func main() {
    // 1. Connect
    conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
    defer conn.Close()
    client := pb.NewPricingServiceClient(conn)
    
    // 2. Get Ask Price
    askResp, err := client.GetAsk(context.Background(), &pb.GetAskRequest{
        OptionParameters: &pb.OptionParameters{
            Symbol:       "EUR/USD",
            ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
            Currency:     "USD",
            Stake:        "100.00",
            Duration:     "5m",
            Barrier:      strPtr("+50"),
        },
    })
    if err != nil {
        log.Fatalf("GetAsk failed: %v", err)
    }
    log.Printf("Ask: %s, Payout: %s", askResp.AskPrice, askResp.Payout)
    
    // 3. Stream Bid (after purchase)
    stream, _ := client.StreamBid(context.Background(), &pb.GetBidRequest{
        OptionParameters: &pb.OptionParameters{
            Symbol:       "EUR/USD",
            ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
            Currency:     "USD",
            Stake:        "100.00",
            Duration:     "5m",
            Barrier:      strPtr("+50"),
            StartTime:    int64Ptr(time.Now().Unix()),
        },
    })
    
    for {
        resp, err := stream.Recv()
        if err != nil {
            break
        }
        log.Printf("Bid: %s, Expired: %v", resp.BidPrice, resp.IsExpired)
        if resp.IsExpired {
            break
        }
    }
}

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }
```

---

## 11. Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-12-23 | Initial API specification |

---

## 12. PRD Traceability

| API Element | PRD Section |
|-------------|-------------|
| GetAsk | [4.1.1](../requirements/prd.md:175) |
| StreamAsk | [4.1.2](../requirements/prd.md:222) |
| GetBid | [4.1.3](../requirements/prd.md:238) |
| StreamBid | [4.1.4](../requirements/prd.md:275) |
| OptionParameters | [7.1](../requirements/prd.md:566) |
| GetAskResponse | [7.2](../requirements/prd.md:596) |
| GetBidResponse | [7.2](../requirements/prd.md:609) |
| ContractType | [7.3](../requirements/prd.md:635) |
| Error Codes | [4.4](../requirements/prd.md:410) |
| Duration Format | [4.2.3](../requirements/prd.md:343) |
| Barrier Format | [4.2.2](../requirements/prd.md:322) |

---

**End of Document**
