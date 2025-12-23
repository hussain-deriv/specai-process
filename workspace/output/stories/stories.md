# User Stories
# Digital Call/Put Options Pricing Service

**Document Version**: 1.0
**Last Updated**: 2025-12-23
**Service Name**: digitalcallput

---

## Executive Summary

The Digital Call/Put Options Pricing Service is a **backend gRPC microservice** with no direct human users. All interactions are machine-to-machine via gRPC API. The "users" of this service are client systems that integrate with the pricing API.

This document captures user stories from the perspective of these **system consumers**, focusing on the functional requirements they need the service to fulfill.

---

## 1. User Types

### 1.1 Trading Platform (UT-TP)

| Attribute | Description |
|-----------|-------------|
| **Name** | Trading Platform |
| **Description** | Web or mobile trading applications that display pricing to end-users and need real-time ask/bid prices for digital options. These platforms present proposals to traders and show active contract values. |
| **Key Characteristics** | Requires low-latency responses, needs streaming updates, displays pricing to human traders |
| **Access Level** | Full API access (all 4 endpoints) |
| **Communication Pattern** | Both unary and streaming RPCs |

### 1.2 Automated Trading System (UT-AT)

| Attribute | Description |
|-----------|-------------|
| **Name** | Automated Trading System |
| **Description** | Algorithmic trading bots and automated systems that programmatically request pricing for decision-making without human intervention. |
| **Key Characteristics** | High-frequency requests, programmatic decision-making, low-latency critical |
| **Access Level** | Full API access (all 4 endpoints) |
| **Communication Pattern** | Primarily unary RPCs for quick decisions |

### 1.3 Financial Service Integrator (UT-FI)

| Attribute | Description |
|-----------|-------------|
| **Name** | Financial Service Integrator |
| **Description** | Third-party systems that aggregate pricing from multiple sources or build composite products on top of the pricing service. |
| **Key Characteristics** | May batch requests, aggregates data from multiple symbols, builds derivative products |
| **Access Level** | Full API access (all 4 endpoints) |
| **Communication Pattern** | Mix of unary and streaming based on use case |

---

## 2. User Stories by Type

### 2.1 Trading Platform Stories

#### US-DC-A1K: Request Single Ask Price
**As a** trading platform,  
**I want to** request a single ask price for a digital option,  
**So that** I can display the proposal price and payout to my users.

**Acceptance Criteria**:
- [ ] Given valid OptionParameters, the service returns AskQuote within 100ms
- [ ] Response includes: ask_price, payout, current_spot, current_spot_time, limits
- [ ] Ask price equals the stake amount submitted
- [ ] Payout reflects Black-Scholes calculation with 2% commission applied (hidden)
- [ ] Trading limits (min_stake, max_payout) are included in response
- [ ] gRPC status OK (0) returned for valid requests

**Service Hint**: (pricing)  
**PRD Reference**: [4.1.1](../requirements/prd.md:175)

---

#### US-DC-B2L: Stream Ask Price Updates
**As a** trading platform,  
**I want to** receive continuous ask price updates,  
**So that** my users see real-time pricing as market conditions change.

**Acceptance Criteria**:
- [ ] Stream initiates successfully and sends first response immediately
- [ ] Updates sent on every new tick from market feed
- [ ] If no ticks received, fallback update sent every 5 seconds
- [ ] Each update includes current_spot and current_spot_time
- [ ] Stream continues until client closes connection
- [ ] Payout recalculated with each new spot price

**Service Hint**: (pricing)  
**PRD Reference**: [4.1.2](../requirements/prd.md:222)

---

#### US-DC-C3M: Request Single Bid Price
**As a** trading platform,  
**I want to** request the current value of an active contract,  
**So that** I can show my users what their contract is worth now.

**Acceptance Criteria**:
- [ ] Given valid OptionParameters with start_time, service returns BidQuote within 100ms
- [ ] Response includes: bid_price, entry_spot, barrier, expiry_time, is_expired
- [ ] Entry spot is the first tick after start_time
- [ ] Barrier resolved from relative/absolute/none specification
- [ ] If contract expired: bid_price = payout (win) or 0 (loss)
- [ ] Win condition: CALL wins if exit_spot > barrier; PUT wins if exit_spot < barrier

**Service Hint**: (pricing)  
**PRD Reference**: [4.1.3](../requirements/prd.md:238)

---

#### US-DC-D4N: Stream Bid Price Updates
**As a** trading platform,  
**I want to** receive continuous bid price updates for an active contract,  
**So that** my users see real-time contract valuation until expiry.

**Acceptance Criteria**:
- [ ] Stream initiates and sends first response with current bid
- [ ] Updates sent on every new tick from market feed
- [ ] For time-based durations: 5-second fallback if no ticks
- [ ] For tick-based durations: NO fallback, updates only on tick arrival
- [ ] Stream automatically terminates when contract expires
- [ ] Final response includes is_expired=true, exit_spot, and settlement amount

**Service Hint**: (pricing)  
**PRD Reference**: [4.1.4](../requirements/prd.md:275)

---

#### US-DC-E5P: Display Trading Limits
**As a** trading platform,  
**I want to** receive trading limits with every ask response,  
**So that** I can validate user inputs before submission.

**Acceptance Criteria**:
- [ ] AskQuote includes limits object with min_stake and max_payout
- [ ] Limits reflect global configuration values
- [ ] Values are decimal strings (e.g., "1.00", "50000.00")

**Service Hint**: (pricing)  
**PRD Reference**: [4.1.1](../requirements/prd.md:175)

---

### 2.2 Automated Trading System Stories

#### US-DC-F6Q: Validate Symbol Before Trading
**As an** automated trading system,  
**I want to** receive a clear error when I request pricing for an invalid symbol,  
**So that** I can handle the error and try a valid symbol.

**Acceptance Criteria**:
- [ ] Invalid symbol returns gRPC INVALID_ARGUMENT (code 3)
- [ ] Error message includes: "Invalid symbol: {symbol}"
- [ ] Valid symbols are those available in service-feed

**Service Hint**: (pricing)  
**PRD Reference**: [4.4.1](../requirements/prd.md:413)

---

#### US-DC-G7R: Validate Duration Format
**As an** automated trading system,  
**I want to** receive a clear error for invalid duration formats,  
**So that** I can fix my request parameters.

**Acceptance Criteria**:
- [ ] Invalid duration returns gRPC INVALID_ARGUMENT (code 3)
- [ ] Error message includes: "Invalid duration format: {duration}"
- [ ] Valid formats: `^\d+[smhdt]$` (30s, 5m, 2h, 1d, 10t)

**Service Hint**: (pricing)  
**PRD Reference**: [4.2.3](../requirements/prd.md:343)

---

#### US-DC-H8S: Validate Stake Amount
**As an** automated trading system,  
**I want to** receive appropriate errors for invalid stake amounts,  
**So that** I can adjust my trading parameters.

**Acceptance Criteria**:
- [ ] Negative stake returns gRPC OUT_OF_RANGE (code 11) with message "Stake must be positive"
- [ ] Stake below minimum returns gRPC FAILED_PRECONDITION (code 9) with message "Stake below minimum: {min_stake}"
- [ ] Valid stake is positive and >= min_stake from configuration

**Service Hint**: (pricing)  
**PRD Reference**: [4.4.1](../requirements/prd.md:428)

---

#### US-DC-I9T: Handle Market Feed Unavailability
**As an** automated trading system,  
**I want to** receive a clear error when market data is unavailable,  
**So that** I can implement retry logic or fail gracefully.

**Acceptance Criteria**:
- [ ] service-feed unavailability returns gRPC UNAVAILABLE (code 14)
- [ ] Error message: "Market data feed unavailable"
- [ ] Error occurs for all endpoints when feed is down

**Service Hint**: (pricing)  
**PRD Reference**: [4.3.1](../requirements/prd.md:378)

---

#### US-DC-J1U: Calculate Relative Barriers
**As an** automated trading system,  
**I want to** specify barriers relative to entry spot,  
**So that** I can create contracts with dynamic strike prices.

**Acceptance Criteria**:
- [ ] Relative barrier "+50" calculates as entry_spot + 50 pips
- [ ] Relative barrier "-100" calculates as entry_spot - 100 pips
- [ ] No range limits on relative barrier values
- [ ] Barrier resolved when contract starts (at entry spot determination)

**Service Hint**: (pricing)  
**PRD Reference**: [4.2.2](../requirements/prd.md:322)

---

#### US-DC-K2V: Calculate Absolute Barriers
**As an** automated trading system,  
**I want to** specify exact barrier values,  
**So that** I can create contracts with specific strike prices.

**Acceptance Criteria**:
- [ ] Absolute barrier "1.2345" sets barrier exactly to 1.2345
- [ ] Absolute barriers must be positive
- [ ] Invalid format returns gRPC INVALID_ARGUMENT

**Service Hint**: (pricing)  
**PRD Reference**: [4.2.2](../requirements/prd.md:322)

---

#### US-DC-L3W: Default Barrier to Entry Spot
**As an** automated trading system,
**I want to** omit the barrier parameter,
**So that** the contract uses entry spot as the barrier (at-the-money).

**Acceptance Criteria**:
- [ ] When barrier is null/omitted, barrier = entry_spot
- [ ] Barrier value determined when first tick after start_time arrives
- [ ] BidQuote response shows resolved barrier value

**Service Hint**: (pricing)
**PRD Reference**: [4.2.2](../requirements/prd.md:335)

---

#### US-DC-T1D: Validate Contract Type
**As an** automated trading system,
**I want to** receive a clear error for invalid contract types,
**So that** I can correct my request to use only CALL or PUT.

**Acceptance Criteria**:
- [ ] Invalid contract type returns gRPC INVALID_ARGUMENT (code 3)
- [ ] Error message includes: "Invalid contract type: {type}"
- [ ] Only "CALL" and "PUT" are valid values
- [ ] Case sensitivity: must be uppercase

**Service Hint**: (pricing)
**PRD Reference**: [4.4.1](../requirements/prd.md:413)

---

#### US-DC-U2E: Validate Currency Code
**As an** automated trading system,
**I want to** receive a clear error for invalid currency codes,
**So that** I can use a supported currency for my stake.

**Acceptance Criteria**:
- [ ] Invalid currency returns gRPC INVALID_ARGUMENT (code 3)
- [ ] Error message includes: "Invalid currency: {currency}"
- [ ] Valid currencies are 3-letter ISO codes (e.g., USD, EUR, GBP)

**Service Hint**: (pricing)
**PRD Reference**: [4.4.1](../requirements/prd.md:413)

---

#### US-DC-V3F: Validate Start Time for Bid Requests
**As an** automated trading system,
**I want to** receive a clear error when start_time is missing or invalid for bid requests,
**So that** I can provide valid contract start information.

**Acceptance Criteria**:
- [ ] Missing start_time for GetBid/StreamBid returns gRPC INVALID_ARGUMENT (code 3)
- [ ] Error message: "start_time is required for bid requests"
- [ ] Future start_time returns gRPC INVALID_ARGUMENT with message "start_time cannot be in the future"
- [ ] start_time must be ISO 8601 UTC format

**Service Hint**: (pricing)
**PRD Reference**: [4.4.1](../requirements/prd.md:413)

---

### 2.3 Financial Service Integrator Stories

#### US-DC-M4X: Request Tick-Based Contract Pricing
**As a** financial service integrator,  
**I want to** request pricing for tick-based duration contracts,  
**So that** I can offer tick-expiry products to my users.

**Acceptance Criteria**:
- [ ] Duration "10t" creates contract expiring after 10 ticks
- [ ] Expiry time is NOT calculated from start_time + duration
- [ ] Contract expires ONLY when specified number of ticks received
- [ ] StreamBid has NO 5-second fallback for tick durations
- [ ] If no ticks arrive, contract remains active indefinitely

**Service Hint**: (pricing)  
**PRD Reference**: [4.2.3](../requirements/prd.md:350), Entry 12

---

#### US-DC-N5Y: Request Time-Based Contract Pricing
**As a** financial service integrator,  
**I want to** request pricing for time-based duration contracts,  
**So that** I can offer standard time-expiry products.

**Acceptance Criteria**:
- [ ] Duration "30s" expires at start_time + 30 seconds
- [ ] Duration "5m" expires at start_time + 300 seconds
- [ ] Duration "2h" expires at start_time + 7200 seconds
- [ ] Duration "1d" expires at start_time + 86400 seconds
- [ ] StreamBid has 5-second fallback update if no ticks

**Service Hint**: (pricing)  
**PRD Reference**: [4.2.3](../requirements/prd.md:343)

---

#### US-DC-P6Z: Understand Settlement Logic
**As a** financial service integrator,  
**I want to** understand exactly when and how contracts settle,  
**So that** I can correctly reconcile outcomes with my users.

**Acceptance Criteria**:
- [ ] CALL contract wins when exit_spot > barrier
- [ ] PUT contract wins when exit_spot < barrier
- [ ] Winning contract: bid_price = payout
- [ ] Losing contract: bid_price = 0
- [ ] Exit spot is first tick at or after expiry_time (time-based) or Nth tick (tick-based)

**Service Hint**: (pricing)  
**PRD Reference**: [4.1.3](../requirements/prd.md:266)

---

#### US-DC-Q7A: Verify Black-Scholes Pricing Accuracy
**As a** financial service integrator,  
**I want to** understand the pricing model parameters,  
**So that** I can validate pricing against my own calculations.

**Acceptance Criteria**:
- [ ] Volatility (σ) = 10% fixed for all symbols
- [ ] Interest rate (r) = 0%
- [ ] Quanto drift (q) = 0
- [ ] Commission = 2% of payout (applied, but hidden from response)
- [ ] Payout = stake / probability (before commission)
- [ ] Final payout = payout - (payout × 0.02)

**Service Hint**: (pricing)  
**PRD Reference**: [4.2.1](../requirements/prd.md:295), Entry 2

---

---

## 3. Cross-User Type Stories

### US-DC-R8B: Consistent Error Handling Across All Consumers
**As any** API consumer,  
**I want to** receive consistent, standard gRPC error codes,  
**So that** I can implement unified error handling logic.

**Acceptance Criteria**:
- [ ] All validation errors use appropriate gRPC codes (see table below)
- [ ] Error messages are clear and actionable
- [ ] No custom error structures - only standard gRPC status

| Condition | gRPC Code | Example Message |
|-----------|-----------|-----------------|
| Invalid symbol | INVALID_ARGUMENT (3) | "Invalid symbol: XXX" |
| Invalid duration | INVALID_ARGUMENT (3) | "Invalid duration format: 5x" |
| Invalid barrier | INVALID_ARGUMENT (3) | "Invalid barrier format: abc" |
| Stake below min | FAILED_PRECONDITION (9) | "Stake below minimum: 1.00" |
| Negative stake | OUT_OF_RANGE (11) | "Stake must be positive" |
| Feed unavailable | UNAVAILABLE (14) | "Market data feed unavailable" |

**Service Hint**: (pricing)  
**PRD Reference**: [4.4.2](../requirements/prd.md:452)

---

### US-DC-S9C: Low Latency for All Request Types
**As any** API consumer,  
**I want to** receive responses within acceptable latency bounds,  
**So that** I can provide real-time pricing to my downstream systems.

**Acceptance Criteria**:
- [ ] Unary RPC (GetAsk, GetBid): < 100ms p95
- [ ] Stream updates: < 50ms from tick arrival
- [ ] First stream response: < 100ms from request
- [ ] Service supports 10,000+ requests per second
- [ ] Service supports 1000+ concurrent streams

**Service Hint**: (pricing)  
**PRD Reference**: [5.1](../requirements/prd.md:477)

---

## 4. Story Coverage Matrix

| PRD Section | Feature | Story ID(s) | User Types |
|-------------|---------|-------------|------------|
| 4.1.1 | GetAsk endpoint | US-DC-A1K, US-DC-E5P | UT-TP, UT-AT, UT-FI |
| 4.1.2 | StreamAsk endpoint | US-DC-B2L | UT-TP, UT-FI |
| 4.1.3 | GetBid endpoint | US-DC-C3M, US-DC-P6Z | UT-TP, UT-AT, UT-FI |
| 4.1.4 | StreamBid endpoint | US-DC-D4N | UT-TP, UT-FI |
| 4.2.1 | Black-Scholes | US-DC-Q7A | UT-FI |
| 4.2.2 | Barrier calculation | US-DC-J1U, US-DC-K2V, US-DC-L3W | UT-AT |
| 4.2.3 | Duration parsing | US-DC-G7R, US-DC-M4X, US-DC-N5Y | UT-AT, UT-FI |
| 4.3.1 | service-feed integration | US-DC-I9T | UT-AT |
| 4.4.1 | Input validation | US-DC-F6Q, US-DC-G7R, US-DC-H8S, US-DC-T1D, US-DC-U2E, US-DC-V3F | UT-AT |
| 4.4.2 | Error handling | US-DC-R8B | All |
| 5.1 | Performance | US-DC-S9C | All |

---

## 5. Service Distribution Summary

All stories map to a single service: **digitalcallput**

| Service Hint | Story Count | Story IDs |
|--------------|-------------|-----------|
| pricing | 22 | All stories |

This is expected - the digitalcallput service is a single focused microservice with no internal service boundaries.

---

## 6. Quality Checklist

- [x] All user types from PRD identified (3 system consumers)
- [x] Each user type has clear description and characteristics
- [x] All PRD features have corresponding user stories (22 stories)
- [x] Stories follow standard format (As a... I want... So that...)
- [x] Story IDs are unique and properly formatted (US-DC-XXX)
- [x] Service hints assigned (all: pricing)
- [x] Cross-user interactions captured (2 cross-type stories)
- [x] Story Coverage Matrix is complete
- [x] Edge cases included (tick duration, feed unavailability)
- [x] Error handling scenarios covered (9 error types)
- [x] All validation types from PRD 4.4.1 covered (symbol, duration, stake, contract type, currency, start time)

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-12-23 | Initial user stories creation |
| 1.1 | 2025-12-23 | Added 3 missing validation stories (US-DC-T1D, US-DC-U2E, US-DC-V3F) |

---

**End of Document**
