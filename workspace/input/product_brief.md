# New Financial Product Implementation Template

This template is designed to guide a non-technical team in defining a new financial product and to provide a structured specification for an AI agent to implement the corresponding Go microservice.

## Part 1: Product Definition

Digital Call/Put options is a binary option. For a Call option, client wins full payout if the symbol's exit price is strictly higher than the barrier. For a Put option, client wins full payout if the symbol's exit price is strictly lower than the barrier. Barrier can be defined by the client. The product accepts relative barrier, absolute barrier or null.

**Ask Request (Pre-purchase Proposal):**
- **Inputs**: Client provides `Stake`. `Payout` is not a valid input.
- **Barrier**: If relative barrier is provided, it is calculated from the **Current Spot Price**. If barrier is not provided, the **Current Spot Price** is used as the barrier.
- **Pricing**: The Ask Price returned is equal to the input `Stake`. The system calculates the potential `Payout` based on this stake and the theoretical probability + commission markup.

**Bid Request (Active/Sold Contract):**
- **Inputs**: Client must provide the resolved **Absolute Barrier** and **Start Time**. Relative barriers are not accepted for active contracts.
- **Lifecycle**:
  - **Start Condition**: The contract starts once it receives an entry price (first tick at or after start time).
  - **Winning Condition**: Call wins if exit price > barrier. Put wins if exit price < barrier.
  - **Losing Condition**: Call loses if exit price <= barrier. Put loses if exit price >= barrier.
  - **Expiry**: Contract expires at the expiry time (time-based) or after N ticks (tick-based).
  - **Early Exit**: Contract can be sold at market value using Cash-or-Nothing Black-Scholes.

### 1. Contract Request Parameters for Get & Stream endpoints (Inputs)
*Digital option parameters*
*   **Symbol**: The underlying asset (e.g. USD/JPY, BTC/USD). Required parameter.
*   **Currency**: The payout currency of the contract (e.g., USD, EUR). Required parameter.
*   **Stake**: The premium paid to enter into a contract. Required parameter.
*   **Duration**: A string that consists of duration amount and duration unit (e.g. '1m' is 1 minute, '30s' is 30 seconds, '2h' is 2 hours, '5d' is 5 days and '5t' is 5 ticks). Required parameter.
*   **Barrier**:
    - For **Ask (Proposal)**: Relative or absolute barrier. Optional. Defaults to Current Spot if null.
    - For **Bid (Active)**: **Absolute Barrier** required.
*   **Start Time**: The start time of the contract. Only required for bid request.
*Pricing time*: Pricing request epoch time. Optional parameter.

### 2. Ask Response
*   **Current Spot**: Spot price of the underlying asset.
*   **Current Spot Time**: Time associated with the spot price.
*   **Payout**: Potential payout calculated by the system.
*   **Ask Price**: Ask price (equals input Stake).
*   **Currency**: The contract quoted currency.
*   **Limits**: Contains option specific trading limits.

### 3. Bid Response
*   **Currrent Spot**: Spot price of the underlying asset.
*   **Current Spot Time**: Time associated with the spot price.
*   **Entry Spot**: Entry price of the contract.
*   **Entry Spot Time**: Time associated with the entry price.
*   **Exit Spot**: Exit price of the contract.
*   **Exit Spot Time**: Time associated with the exit price.
*   **Bid Price**: Bid price.
*   **Currency**: The contract quoted currency.
*   **Barrier**: Contract barrier.
*   **Start Time**: Contract start time.
*   **Expiry Time**: Contract end time.
*   **Is Expired**: Boolean to indicate if contract is expired.

### 4. Limits
*   **Max Payout**: Maximum payout per contract.
*   **Min Stake**: Minimum stake/ask price per contract.

### 5. Pricing Logic (The "Ask")
*How do we calculate the proposal before purchase?*
*   **Inputs**: Spot price, Barrier, Duration in years, Payout currency interest rate, Quanto drift, Volatility.
    *   **Market Data Sources**: Spot prices are fetched from `BOM::MarketData` equivalent services. Interest rates and Volatility surfaces are loaded from configuration files (e.g., `config.yml`) and benchmark CSV files (e.g., `SD_GBPJPY.csv`).
*   **Formula/Logic**:
    *   **Barrier Calculation**:
        - Relative barrier (e.g., "+10") is calculated as `Current Spot + Offset`.
        - If barrier is missing, `Current Spot` is used.
    *   **Time Calculation for Ticks**: 
        - For tick-based durations (e.g., '5t'), the expiry is **event-based**. The contract expires after the N-th valid market tick arrives after the start time.
        - Note: There is typically a maximum waiting period (e.g., 5 minutes) for tick-based contracts.
    *   **Payout Calculation**: Use the **Cash-or-Nothing Black-Scholes** formula. This pays a fixed cash amount if the option expires in-the-money, and zero otherwise.
    *   **Limits**: Min stake and Max payout are defined in **per-symbol configuration** (dynamically loaded from files like `contract_types.yml` and `contract_categories.yml`).
    *   **Commission**: 
        - Commission is a **pricing markup** included in the Ask Price.
        - `Ask Price = Theoretical Price + Commission`.
        - `Commission = Number of Contracts * (Ask Probability - Theoretical Probability)`.
        - No commission is deducted from the final payout if the contract is held to expiry.
    *   **Financial Rounding**: 
        - Amounts and prices must be rounded according to the currency's precision (e.g., 2 decimals for USD, 0 for JPY).
        - Rounding method is typically "nearest" (half-up).

### 6. Lifecycle & State Machine (The "Bid" / Active Contract)
*What happens after purchase?*
*   **Start Condition**: The contract starts once it receives an entry price.
    - **Entry Spot Determination**: The system must look up the **first valid tick at or after the Start Time** from historical market data.
*   **Update Frequency**: Contract price should be updated when it receives a new tick or 5 seconds after the previous price update. Update stops after contract expires.
*   **Winning Condition**: For a Call option, client wins when the exit price is strictly higher than the barrier. For a Put option, client wins when the exit price is strictly lower than the barrier.
*   **Losing Condition**: For a Call option, client loses when the exit prcie is lower or equal than the barrier. For a Put option, client loses when the exit price is lower or equal than the barrier.
*   **Expiry Condition**: 
    - **Time-based**: Expires at the calculated Expiry Time.
    - **Tick-based**: Expires when the N-th tick arrives (event-driven).
*   **Early Exit**: The contract can be sold at market. 
    - `Bid Price = Theoretical Price - Sell Commission`.
    - `Sell Commission = Number of Contracts * (Theoretical Probability - Bid Probability)`.

---

## Part 2: Technical Implementation Guide (For AI Agent)

*Use this section to generate the Go service code based on the definitions above.*

### 1. Service Code Structure
The service should follow standard go template.
*   Use the existing `go-templates` if available in the workspace or standard project structure.

### 2. Proto Definition

In service-pricer-digitalcallput repository, update protobuf with the following definitions.

```protobuf
syntax = "proto3";

package digitalcallput.v1;

option go_package="github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput";

service DigitalcallputService {
  // Request a single contract price (Ask)
  rpc GetAsk (GetAskRequest) returns (GetAskResponse);
  // Stream contract prices (Ask Stream)
  rpc StreamAsk (StreamAskRequest) returns (stream GetAskResponse);
  // Request value of an active contract (Bid/Sell)
  rpc GetBid (GetBidRequest) returns (GetBidResponse);
  // Stream value of an active contract
  rpc StreamBid (StreamBidRequest) returns (stream GetBidResponse);
}

enum ContractType {
  CONTRACT_TYPE_UNSPECIFIED = 0;
  CONTRACT_TYPE_CALL = 1;
  CONTRACT_TYPE_PUT = 2;
}

message OptionParameters {
  string symbol = 1;
  ContractType contract_type = 2;
  string currency = 3;
  string duration = 4; // e.g. "1m", "30s", "5t"
  optional string barrier = 5; // Relative (+/-) or absolute. Optional.
  optional int64 start_time = 6; // Required for active contract
  string stake = 7; // Premium paid. Required.
}

message GetAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2; // Epoch time
}

message StreamAskRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}

message GetAskResponse {
  string ask_price = 1;
  string currency = 2;
  string current_spot = 3;
  int64 current_spot_time = 4;
  string payout = 5;
  Limits limits = 6;
}

message Limits {
  string max_payout = 1;
  string min_stake = 2;
}

message GetBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}

message StreamBidRequest {
  OptionParameters option_parameters = 1;
  optional int64 pricing_time = 2;
}

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
