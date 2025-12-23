# Product Requirements Document (PRD)
# Digital Call/Put Options Pricing Service

**Document Version**: 1.1
**Last Updated**: 2025-12-23
**Service Name**: digitalcallput
**Complexity Score**: 6/10 (Standard)

---

## 1. Executive Summary

### 1.1 Product Overview
The Digital Call/Put Options Pricing Service is a stateless gRPC microservice that provides real-time pricing for binary options contracts. The service calculates ask prices (pre-purchase proposals) and bid prices (active contract valuations) using the Black-Scholes pricing model, with support for both single requests and streaming updates.

### 1.2 Business Objectives
- Provide accurate, real-time pricing for digital call and put options
- Support both proposal (ask) and active contract (bid) pricing
- Enable streaming price updates for dynamic market conditions
- Ensure compliance with financial trading service standards
- Maintain stateless architecture for scalability

### 1.3 Target Users
- Trading platforms and applications
- Automated trading systems
- Financial service integrators
- Internal trading services

### 1.4 Success Metrics
- Pricing accuracy (Black-Scholes implementation correctness)
- Response latency (< 100ms for single requests)
- Stream update frequency (on tick arrival or 5-second fallback for time-based durations)
- Service availability (99.9% uptime)
- Error rate (< 0.1% for valid requests)

---

## 2. Product Scope

### 2.1 In Scope
- **Pricing Endpoints**:
  - GetAsk: Single ask price calculation
  - StreamAsk: Continuous ask price updates
  - GetBid: Single bid price calculation for active contracts
  - StreamBid: Continuous bid price updates for active contracts

- **Contract Types**:
  - Digital Call options
  - Digital Put options

- **Barrier Types**:
  - Relative barriers (e.g., "+50", "-100")
  - Absolute barriers (e.g., "1.2345")
  - No barrier (defaults to entry price)

- **Duration Formats**:
  - Seconds (e.g., "30s")
  - Minutes (e.g., "5m")
  - Hours (e.g., "2h")
  - Days (e.g., "5d")
  - Ticks (e.g., "10t")

- **Pricing Features**:
  - Black-Scholes pricing model
  - Commission deduction (2%, hidden from responses)
  - Trading limits validation (min stake, max payout)
  - Real-time spot price integration via service-feed

### 2.2 Out of Scope
- User authentication and authorization
- Contract purchase/execution
- Payment processing
- Historical price storage
- Database operations (stateless service)
- Multi-leg or exotic option types
- Dynamic volatility calculation (uses fixed 10%)
- Risk management and position tracking

### 2.3 Future Considerations
- Per-symbol volatility configuration
- Dynamic commission rates
- Advanced option types (Asian, Barrier, etc.)
- Historical pricing analytics
- A/B testing for pricing strategies

---

## 3. User Stories & Use Cases

### 3.1 Core User Stories

**US-1: Request Single Ask Price**
- **As a** trading platform
- **I want to** request a single ask price for a digital option
- **So that** I can display the proposal to my users
- **Acceptance Criteria**:
  - Service returns ask price, payout, current spot, and limits
  - Response time < 100ms
  - Validates all input parameters
  - Returns appropriate errors for invalid inputs

**US-2: Stream Ask Prices**
- **As a** trading platform
- **I want to** receive continuous ask price updates
- **So that** users see real-time pricing as market moves
- **Acceptance Criteria**:
  - Stream updates on every new tick
  - Fallback to 5-second updates if no ticks
  - Stream continues until client closes connection
  - Each update includes current spot and timestamp

**US-3: Request Single Bid Price**
- **As a** trading platform
- **I want to** get the current value of an active contract
- **So that** users can see their contract's market value
- **Acceptance Criteria**:
  - Service returns bid price, entry/exit spots, barrier, expiry
  - Calculates value based on current market conditions
  - Indicates if contract has expired
  - Validates contract start time is provided

**US-4: Stream Bid Prices**
- **As a** trading platform
- **I want to** receive continuous bid price updates for active contracts
- **So that** users see real-time contract valuations
- **Acceptance Criteria**:
  - Stream updates on every new tick
  - Fallback to 5-second updates if no ticks
  - Stream terminates on client close OR contract expiry
  - Final update shows expiry status and settlement

**US-5: Validate Trading Limits**
- **As a** risk management system
- **I want to** ensure all contracts respect trading limits
- **So that** exposure is controlled per symbol
- **Acceptance Criteria**:
  - Rejects stakes below minimum
  - Returns max payout limits in responses
  - Limits loaded from configuration files
  - Clear error messages for limit violations

### 3.2 Use Case Scenarios

**Scenario 1: Pre-Purchase Proposal**
1. Client requests ask price for EUR/USD call option
2. Parameters: stake=100 USD, duration=5m, barrier=+50
3. Service fetches current spot from service-feed
4. Service calculates barrier from spot + 50 pips
5. Service applies Black-Scholes with 10% volatility
6. Service deducts 2% commission from payout
7. Service validates stake against min/max limits
8. Service returns ask price and payout to client

**Scenario 2: Active Contract Monitoring**
1. Client purchases contract and starts StreamBid
2. Contract starts when first tick arrives (entry spot)
3. Service calculates expiry time from start + duration
4. Service streams bid price updates on each tick
5. Client sees real-time contract value changes
6. Contract expires at expiry time
7. Final update shows settlement (win/loss)
8. Stream terminates automatically

**Scenario 3: Barrier Calculation**
- **Relative Barrier**: Client sends "+100", spot=1.2000 → barrier=1.2100
- **Absolute Barrier**: Client sends "1.2500" → barrier=1.2500
- **No Barrier**: Client sends null → barrier=entry_spot (determined at contract start)

---

## 4. Functional Requirements

### 4.1 API Endpoints

#### 4.1.1 GetAsk
**Purpose**: Calculate single ask price for a digital option proposal

**Request Parameters**:
- `option_parameters` (required):
  - `symbol` (string, required): Underlying asset (e.g., "EUR/USD", "BTC/USD")
  - `contract_type` (enum, required): CALL or PUT
  - `currency` (string, required): Payout currency (e.g., "USD", "EUR")
  - `stake` (string, required): Premium amount (decimal string)
  - `duration` (string, required): Contract duration (e.g., "5m", "30s", "10t")
  - `barrier` (string, optional): Relative ("+50") or absolute ("1.2345")
  - `start_time` (int64, optional): Not used for ask requests
- `pricing_time` (int64, optional): Epoch timestamp for pricing (defaults to current time)

**Response Fields**:
- `ask_price` (string): Price to purchase the contract
- `currency` (string): Contract currency
- `current_spot` (string): Current market spot price
- `current_spot_time` (int64): Timestamp of spot price
- `payout` (string): Potential payout if contract wins
- `limits` (object):
  - `max_payout` (string): Maximum payout allowed
  - `min_stake` (string): Minimum stake required

**Business Rules**:
- Ask price = Payout × Probability - Commission
- Commission = 2% of payout (hidden from response)
- Probability calculated using Black-Scholes
- Volatility = 10% (global configuration)
- Interest rate = 0%
- Quanto drift = 0

**Validation Rules**:
- Symbol must be valid and supported
- Contract type must be CALL or PUT
- Currency must be valid
- Stake must be positive decimal
- Stake must be >= min_stake
- Duration must match format: `\d+[smhdt]`
- Barrier format: optional, relative ("+/-\d+") or absolute ("\d+\.?\d*")

**Error Conditions**:
- `INVALID_ARGUMENT`: Invalid symbol, duration format, currency
- `FAILED_PRECONDITION`: Stake below minimum
- `OUT_OF_RANGE`: Negative stake
- `UNAVAILABLE`: service-feed unavailable

#### 4.1.2 StreamAsk
**Purpose**: Stream continuous ask price updates

**Request Parameters**: Same as GetAsk

**Response Stream**: Continuous stream of GetAskResponse messages

**Stream Behavior**:
- Updates on every new tick from service-feed
- Fallback: Update every 5 seconds if no ticks received
- Continues until client closes connection
- No automatic termination

**Business Rules**: Same as GetAsk

#### 4.1.3 GetBid
**Purpose**: Calculate single bid price for an active contract

**Request Parameters**:
- `option_parameters` (required):
  - All fields from GetAsk
  - `start_time` (int64, required): Contract start timestamp

**Response Fields**:
- `bid_price` (string): Current market value of contract
- `is_expired` (bool): Whether contract has expired
- `current_spot` (string): Current market spot price
- `current_spot_time` (int64): Timestamp of current spot
- `entry_spot` (string): Contract entry price
- `entry_spot_time` (int64): Timestamp of entry
- `exit_spot` (string): Contract exit price (if expired)
- `exit_spot_time` (int64): Timestamp of exit (if expired)
- `barrier` (string): Calculated barrier value
- `start_time` (int64): Contract start time
- `expiry_time` (int64): Contract expiry time
- `currency` (string): Contract currency

**Business Rules**:
- Entry spot = First tick after start_time
- **Expiry time calculation**:
  - Time-based durations (s/m/h/d): start_time + duration
  - Tick-based durations (t): After N ticks received (no time-based expiry)
- Exit spot = First tick at or after expiry_time (for time-based) or Nth tick (for tick-based)
- Bid price calculated using Black-Scholes with time remaining
- If expired:
  - Call wins if exit_spot > barrier
  - Put wins if exit_spot < barrier
  - Bid price = payout (if win) or 0 (if loss)

**Validation Rules**: Same as GetAsk, plus:
- start_time must be provided
- start_time must be in the past

#### 4.1.4 StreamBid
**Purpose**: Stream continuous bid price updates for active contract

**Request Parameters**: Same as GetBid

**Response Stream**: Continuous stream of GetBidResponse messages

**Stream Behavior**:
- Updates on every new tick from service-feed
- **Fallback for time-based durations only**: Update every 5 seconds if no ticks received
- **Tick-based durations**: No time-based fallback, updates only on tick arrivals
- Terminates when:
  - Client closes connection, OR
  - Contract expires (automatic termination)
- Final update includes expiry status and settlement

**Business Rules**: Same as GetBid

### 4.2 Pricing Logic

#### 4.2.1 Black-Scholes Implementation
**Formula**: Standard Black-Scholes for binary options

**Inputs**:
- S: Current spot price
- K: Barrier price
- T: Time to expiry (in years)
- σ: Volatility (10%)
- r: Interest rate (0%)
- q: Quanto drift (0%)

**Outputs**:
- Probability of winning
- Payout amount
- Ask/Bid price

**Calculation Steps**:
1. Parse duration to seconds
2. Convert to years: T = seconds / (365.25 × 24 × 3600)
3. Calculate d1 and d2 using Black-Scholes formula
4. Calculate probability using cumulative normal distribution
5. Payout = stake / probability (before commission)
6. Commission = payout × 0.02
7. Final payout = payout - commission
8. Ask price = stake
9. Bid price = payout × probability (time-adjusted)

#### 4.2.2 Barrier Calculation
**Relative Barrier**:
- Format: "+\d+" or "-\d+"
- Calculation: barrier = entry_spot + relative_value
- Example: entry_spot=1.2000, barrier="+50" → 1.2050

**Absolute Barrier**:
- Format: "\d+\.?\d*"
- Calculation: barrier = absolute_value
- Example: barrier="1.2500" → 1.2500

**No Barrier**:
- Format: null or empty
- Calculation: barrier = entry_spot
- Determined when contract starts

**Validation**:
- No range limits on relative barriers (accept any value)
- Absolute barriers must be positive
- Invalid format returns gRPC error

#### 4.2.3 Duration Parsing
**Supported Units**:
- `s`: Seconds (e.g., "30s" = 30 seconds)
- `m`: Minutes (e.g., "5m" = 300 seconds)
- `h`: Hours (e.g., "2h" = 7200 seconds)
- `d`: Days (e.g., "1d" = 86400 seconds)
- `t`: Ticks (e.g., "10t" = 10 ticks)

**Tick Duration**:
- Duration in ticks requires counting tick arrivals
- Contract expires ONLY after N ticks received
- **No time-based fallback**: If no ticks arrive, contract remains active indefinitely
- Contract must wait for actual tick arrivals to complete

**Validation**:
- Format: `^\d+[smhdt]$`
- Amount must be positive integer
- Invalid format returns `INVALID_ARGUMENT` error

### 4.3 Data Integration

#### 4.3.1 service-feed Integration
**Purpose**: Real-time market data feed

**Repository**: `github.com/junbon-deriv/service-feed`

**API Definition**: `proto/grpcfeed/v1/ticks.proto`

**Client Implementation**: `client/client.go`

**Usage**:
- Subscribe to symbol ticks
- Receive real-time spot prices
- Use for ask/bid calculations
- Handle connection failures gracefully

**Error Handling**:
- Retry on connection loss
- Return `UNAVAILABLE` if feed unavailable
- Log all feed errors

#### 4.3.2 Configuration Files
**Purpose**: Trading limits and pricing parameters

**Location**: Service configuration directory

**Format**: YAML or JSON

**Contents**:
```yaml
global:
  volatility: 0.10
  commission: 0.02
  interest_rate: 0.0
  quanto_drift: 0.0

limits:
  default:
    min_stake: "1.00"
    max_payout: "50000.00"
```

**Loading**:
- Load at service startup
- Requires service restart for changes
- Validate on load

### 4.4 Validation & Error Handling

#### 4.4.1 Input Validation
**Symbol Validation**:
- Must be non-empty string
- Must be supported by service-feed
- Error: `INVALID_ARGUMENT` with message "Invalid symbol: {symbol}"

**Contract Type Validation**:
- Must be CALL or PUT
- Error: `INVALID_ARGUMENT` with message "Invalid contract type"

**Currency Validation**:
- Must be non-empty string
- Must be valid currency code
- Error: `INVALID_ARGUMENT` with message "Invalid currency: {currency}"

**Stake Validation**:
- Must be valid decimal string
- Must be positive
- Must be >= min_stake
- Errors:
  - Negative: `OUT_OF_RANGE` with message "Stake must be positive"
  - Below min: `FAILED_PRECONDITION` with message "Stake below minimum: {min_stake}"

**Duration Validation**:
- Must match format `^\d+[smhdt]$`
- Amount must be positive
- Error: `INVALID_ARGUMENT` with message "Invalid duration format: {duration}"

**Barrier Validation**:
- Optional field
- If provided, must be relative ("+/-\d+") or absolute ("\d+\.?\d*")
- No range limits on values
- Error: `INVALID_ARGUMENT` with message "Invalid barrier format: {barrier}"

**Start Time Validation** (Bid requests only):
- Must be provided
- Must be in the past
- Error: `INVALID_ARGUMENT` with message "Start time required for bid requests"

#### 4.4.2 Error Response Format
**gRPC Status Codes**:
- `INVALID_ARGUMENT`: Invalid input parameters
- `FAILED_PRECONDITION`: Business rule violation (e.g., stake too low)
- `OUT_OF_RANGE`: Value out of acceptable range
- `UNAVAILABLE`: External dependency unavailable
- `INTERNAL`: Unexpected server error

**Error Message Structure**:
- Clear, actionable error messages
- Include parameter name and expected format
- No sensitive information in errors

**Error Examples**:
```
INVALID_ARGUMENT: Invalid symbol: INVALID_SYM
INVALID_ARGUMENT: Invalid duration format: 5x (expected format: \d+[smhdt])
FAILED_PRECONDITION: Stake below minimum: 1.00
OUT_OF_RANGE: Stake must be positive
UNAVAILABLE: Market data feed unavailable
```

---

## 5. Non-Functional Requirements

### 5.1 Performance Requirements
- **Response Latency**:
  - GetAsk: < 100ms (p95)
  - GetBid: < 100ms (p95)
  - Stream updates: < 50ms from tick arrival

- **Throughput**:
  - Support 1000+ concurrent streams
  - Handle 10,000+ requests per second

- **Update Frequency**:
  - Primary: On every tick arrival
  - Fallback: Every 5 seconds if no ticks (time-based durations only)
  - Tick-based durations: Updates only on tick arrivals (no time-based fallback)

### 5.2 Scalability Requirements
- Stateless service design (horizontal scaling)
- No database dependencies
- Independent service instances
- Load balancer compatible

### 5.3 Reliability Requirements
- **Availability**: 99.9% uptime
- **Error Rate**: < 0.1% for valid requests
- **Graceful Degradation**: Continue operating if service-feed has intermittent issues
- **Recovery**: Automatic reconnection to service-feed

### 5.4 Security Requirements
- **Transport Security**: TLS for all gRPC connections
- **Input Validation**: Strict validation of all inputs
- **No Authentication**: Service-level security (not user-level)
- **Audit Logging**: Log all requests and errors

### 5.5 Compliance Requirements
- **Financial Service Standards**: Accurate pricing calculations
- **Regulatory Compliance**: Commission handling (hidden but applied)
- **Audit Trail**: Comprehensive logging for regulatory review

### 5.6 Monitoring & Observability
- **Metrics**:
  - Request count by endpoint
  - Response latency (p50, p95, p99)
  - Error rate by type
  - Active stream count
  - service-feed connection status

- **Logging**:
  - All requests with parameters
  - All errors with context
  - Pricing calculations (for audit)
  - service-feed events

- **Tracing**:
  - Distributed tracing support
  - Request correlation IDs

---

## 6. Technical Constraints

### 6.1 Technology Stack
- **Language**: Go (Golang)
- **Protocol**: gRPC
- **Template**: `github.com/junbon-deriv/go-templates`
- **Service Type**: Stateless microservice

### 6.2 Dependencies
- **service-feed**: `github.com/junbon-deriv/service-feed`
  - Purpose: Real-time market data
  - API: `proto/grpcfeed/v1/ticks.proto`
  - Client: `client/client.go`

### 6.3 Architecture Constraints
- **Stateless**: No database, no persistent storage
- **Configuration**: File-based (YAML/JSON)
- **Deployment**: Containerized (Docker)
- **Service Discovery**: Compatible with standard service mesh

### 6.4 Repository Structure
- **Repository**: `service-pricer-digitalcallput`
- **Module Path**: `github.com/regentmarkets/service-pricer-digitalcallput`
- **Package Name**: `digitalcallput`

---

## 7. Data Models

### 7.1 Request Models

#### OptionParameters
```protobuf
message OptionParameters {
  string symbol = 1;              // Required: "EUR/USD", "BTC/USD"
  ContractType contract_type = 2; // Required: CALL or PUT
  string currency = 3;            // Required: "USD", "EUR"
  string duration = 4;            // Required: "5m", "30s", "10t"
  optional string barrier = 5;    // Optional: "+50", "1.2345"
  optional int64 start_time = 6;  // Required for bid requests
  string stake = 7;               // Required: "100.00"
}
```

#### GetAskRequest
```protobuf
message GetAskRequest {
  OptionParameters option_parameters = 1; // Required
  optional int64 pricing_time = 2;        // Optional: epoch timestamp
}
```

#### GetBidRequest
```protobuf
message GetBidRequest {
  OptionParameters option_parameters = 1; // Required (with start_time)
  optional int64 pricing_time = 2;        // Optional: epoch timestamp
}
```

### 7.2 Response Models

#### GetAskResponse
```protobuf
message GetAskResponse {
  string ask_price = 1;        // "100.00"
  string currency = 2;         // "USD"
  string current_spot = 3;     // "1.2345"
  int64 current_spot_time = 4; // 1703260800
  string payout = 5;           // "196.00"
  Limits limits = 6;           // Trading limits
}
```

#### GetBidResponse
```protobuf
message GetBidResponse {
  string bid_price = 1;        // "150.00"
  bool is_expired = 2;         // false
  string current_spot = 3;     // "1.2350"
  int64 current_spot_time = 4; // 1703260850
  string entry_spot = 5;       // "1.2340"
  int64 entry_spot_time = 6;   // 1703260800
  string exit_spot = 7;        // "1.2360" (if expired)
  int64 exit_spot_time = 8;    // 1703261100 (if expired)
  string barrier = 9;          // "1.2390"
  int64 start_time = 10;       // 1703260800
  int64 expiry_time = 11;      // 1703261100
  string currency = 12;        // "USD"
}
```

#### Limits
```protobuf
message Limits {
  string max_payout = 1; // "50000.00"
  string min_stake = 2;  // "1.00"
}
```

### 7.3 Enums

#### ContractType
```protobuf
enum ContractType {
  CONTRACT_TYPE_UNSPECIFIED = 0;
  CONTRACT_TYPE_CALL = 1;
  CONTRACT_TYPE_PUT = 2;
}
```

---

## 8. User Interface Requirements

**N/A** - This is a backend gRPC service with no user interface. Client applications will integrate via gRPC API.

---

## 9. Integration Requirements

### 9.1 service-feed Integration
**Type**: gRPC client

**Purpose**: Real-time market data subscription

**Implementation**:
- Use provided client implementation from `service-feed/client/client.go`
- Subscribe to required symbols on service startup
- Handle tick events for price updates
- Implement reconnection logic for resilience

**Data Flow**:
1. Service subscribes to symbols via service-feed
2. service-feed streams tick updates
3. Service uses ticks for ask/bid calculations
4. Service streams updates to clients

**Error Handling**:
- Log connection failures
- Retry with exponential backoff
- Return `UNAVAILABLE` to clients if feed unavailable
- Continue serving cached prices during brief outages

### 9.2 Configuration Integration
**Type**: File-based configuration

**Purpose**: Trading limits and pricing parameters

**Implementation**:
- Load configuration at startup
- Validate all parameters
- Fail fast if configuration invalid
- Require restart for configuration changes

**Configuration Schema**:
```yaml
global:
  volatility: 0.10        # 10%
  commission: 0.02        # 2%
  interest_rate: 0.0      # 0%
  quanto_drift: 0.0       # 0

limits:
  default:
    min_stake: "1.00"
    max_payout: "50000.00"
```

---

## 10. Testing Requirements

### 10.1 Unit Testing
- Black-Scholes calculation accuracy
- Barrier calculation logic
- Duration parsing
- Input validation
- Error handling

### 10.2 Integration Testing
- service-feed integration
- gRPC endpoint functionality
- Stream behavior
- Configuration loading

### 10.3 Performance Testing
- Load testing (10,000+ req/s)
- Concurrent stream testing (1000+ streams)
- Latency testing (p95 < 100ms)

### 10.4 Test Cases

**TC-1: Valid Ask Request**
- Input: Valid parameters
- Expected: Successful response with pricing

**TC-2: Invalid Symbol**
- Input: Unknown symbol
- Expected: `INVALID_ARGUMENT` error

**TC-3: Stake Below Minimum**
- Input: Stake < min_stake
- Expected: `FAILED_PRECONDITION` error

**TC-4: Relative Barrier Calculation**
- Input: barrier="+50", spot=1.2000
- Expected: barrier=1.2050 in response

**TC-5: Stream Termination on Expiry**
- Input: StreamBid for 30s contract
- Expected: Stream auto-closes after 30s

**TC-6: Tick-based Duration**
- Input: duration="10t"
- Expected: Contract expires ONLY after 10 ticks received (no time-based expiry)

**TC-7: No Barrier Default**
- Input: barrier=null
- Expected: barrier=entry_spot in response

---

## 11. Deployment Requirements

### 11.1 Service Instantiation
**Template**: `github.com/junbon-deriv/go-templates`

**Steps**:
1. `cd workspace/code`
2. `git clone git@github.com:junbon-deriv/go-templates.git`
3. `cd go-templates && go install`
4. `cd .. && git clone git@github.com:regentmarkets/service-pricer-digitalcallput.git`
5. `go-templates --template service --module-path github.com/regentmarkets/service-pricer-digitalcallput --module-name digitalcallput`

### 11.2 Environment Requirements
- Go 1.21+
- Docker for containerization
- Access to service-feed endpoint
- Configuration files mounted

### 11.3 Configuration Management
- Configuration files in `/config` directory
- Environment-specific configs (dev, staging, prod)
- Secrets management for TLS certificates

### 11.4 Monitoring Setup
- Prometheus metrics endpoint
- Structured logging (JSON format)
- Distributed tracing integration

---

## 12. Documentation Requirements

### 12.1 API Documentation
- gRPC service definition (protobuf)
- Request/response examples
- Error code reference
- Integration guide

### 12.2 Operational Documentation
- Deployment guide
- Configuration reference
- Monitoring and alerting setup
- Troubleshooting guide

### 12.3 Developer Documentation
- Architecture overview
- Black-Scholes implementation details
- service-feed integration guide
- Testing guide

---

## 13. Assumptions & Dependencies

### 13.1 Assumptions
- service-feed provides reliable, real-time market data
- Configuration changes are infrequent (restart acceptable)
- Clients handle stream reconnection logic
- 10% volatility is acceptable for all symbols
- 2% commission is uniform across all contracts
- No user authentication required (service-level security)

### 13.2 Dependencies
- **service-feed**: Critical dependency for market data
- **go-templates**: Required for service scaffolding
- **gRPC libraries**: Core protocol implementation
- **Configuration files**: Required for service operation

### 13.3 Risks & Mitigations
**Risk**: service-feed unavailability
- **Mitigation**: Retry logic, graceful degradation, monitoring

**Risk**: Extreme barrier values causing calculation errors
- **Mitigation**: Input validation, error handling, logging

**Risk**: High concurrent stream load
- **Mitigation**: Load testing, horizontal scaling, rate limiting

**Risk**: Configuration errors
- **Mitigation**: Validation on startup, fail-fast behavior

---

## 14. Open Questions & Future Enhancements

### 14.1 Resolved Questions
All critical questions have been resolved and documented in [`preferences.md`](workspace/output/requirements/preferences.md).

### 14.2 Future Enhancements
- Per-symbol volatility configuration
- Dynamic commission rates
- Historical pricing analytics
- Advanced option types (Asian, Barrier, Lookback)
- Real-time volatility calculation
- A/B testing framework for pricing strategies
- Database integration for audit trail
- User-level authentication and authorization

---

## 15. Approval & Sign-off

**Document Status**: Ready for Review

**Prepared By**: Product Analyst (AI)  
**Date**: 2025-12-22

**Reviewers**:
- [ ] Product Owner
- [ ] Technical Lead
- [ ] Architecture Team
- [ ] Compliance Team

**Approval Status**: Pending

---

## 16. Changelog

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-12-22 | Product Analyst | Initial PRD creation based on product brief and preferences |
| 1.1 | 2025-12-23 | Product Analyst | Corrected tick duration behavior - removed time-based fallback for tick contracts (Entry 12) |

---

## 17. References

- **Product Brief**: [`workspace/input/product_brief.md`](workspace/input/product_brief.md)
- **Preferences**: [`workspace/output/requirements/preferences.md`](workspace/output/requirements/preferences.md)
- **Workspace Preferences**: [`workspace/input/preferences.md`](workspace/input/preferences.md)
- **service-feed Repository**: `github.com/junbon-deriv/service-feed`
- **go-templates Repository**: `github.com/junbon-deriv/go-templates`
- **Protobuf Definition**: Included in product brief

---

## Appendix A: Preference References

This PRD incorporates the following preference entries:

- **Entry 1**: Complexity assessment (Score: 6/10, Standard template)
- **Entry 2**: Pricing parameters (Volatility: 10%, Commission: 2%, Interest: 0%, Quanto: 0)
- **Entry 3**: Trading limits storage (Configuration files)
- **Entry 4**: Barrier validation (Standard gRPC errors)
- **Entry 5**: Contract update timing (On tick, 5-second fallback for time-based durations)
- **Entry 6**: Service name (digitalcallput)
- **Entry 7**: Configuration management (Global configuration)
- **Entry 8**: Commission visibility (Hidden from responses)
- **Entry 9**: Barrier validation rules (No range limits)
- **Entry 10**: Stream termination (Client close or expiry)
- **Entry 11**: Error handling strategy (Comprehensive with specific codes)
- **Entry 12**: Tick duration behavior correction (No time-based fallback for tick contracts)

---

**End of Document**
