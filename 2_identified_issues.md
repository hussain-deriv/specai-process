# Identified Issues in Product Specification

## 1. Ambiguity in Stake vs. Payout Calculation (Ask Request)
- **Issue**: The "Contract Request Parameters" section lists `Stake` ("Premium paid") as a required parameter. However, the "Ask Response" includes `Ask Price` and `Payout`. The "Pricing Logic" mentions `ask_price` in the commission formula.
- **Ambiguity**: It is unclear whether the user:
    a) Specifies the **Stake** (cost) they are willing to pay, and the system calculates the potential **Payout**.
    b) Specifies the desired **Payout**, and the system calculates the required **Stake** (Ask Price).
- **Implication**: If the user provides the Stake, `Ask Price` in the response seems redundant (unless it includes commission adjustment). If the user provides Payout, `Stake` shouldn't be a required input parameter in the way described.

## 2. Barrier Calculation Reference for "Ask" (Proposal)
- **Issue**: The spec states: "If relative barrier is provided... calculated from symbol's entry price. Entry price is the next price after contract start time."
- **Challenge**: For an "Ask" request (pricing a proposal before purchase), the contract has not started, so there is no actual "Entry Price".
- **Question**: Should we use the **Current Spot Price** as the reference for calculating relative barriers and as the underlying price for the Black-Scholes calculation during the "Ask" phase?

## 3. Bid Request Inputs and Barrier Resolution
- **Issue**: The `GetBidRequest` (for selling/valuing an active contract) takes `OptionParameters` which includes `barrier` (relative or absolute) and `start_time`. It does not explicitly include `entry_spot` or `contract_id`.
- **Challenge**: If a contract was created with a relative barrier (e.g., "+10"), its absolute barrier is determined by the `entry_spot` which occurred just after `start_time`.
- **Question**: Does the `GetBidRequest` expect the caller to pass the resolved **absolute barrier**? Or does the service need to look up the historical `entry_spot` based on `start_time`? If the latter, does the service have access to historical tick data?

## 4. Tick Duration Conversion
- **Issue**: The spec mentions: "For tick-based durations (e.g., '5t'), the duration must be converted to an equivalent time duration based on the tick generation interval".
- **Question**: How is the "tick generation interval" determined? Is it a fixed value per symbol (e.g., in `config.yml`)?

## 5. Commission Logic details
- **Issue**: The formula `buy_commission = financialrounding('price', currency, ask_price - theo_price)` is provided.
- **Question**:
    - If `Stake` is fixed by the user, does `ask_price` = `Stake`?
    - If so, is `theo_price` derived as `Stake - commission`?
    - And is the `Payout` then calculated based on this `theo_price`?
    - Is there a corresponding "Sell Commission" for the `GetBid` (Sell) request?

## 6. External Data Sources
- **Issue**: References `BOM::MarketData`, `config.yml`, and `SD_GBPJPY.csv`.
- **Question**:
    - Is `BOM::MarketData` available as a gRPC service?
    - Are the configuration files and volatility surfaces available in the repository or need to be mocked/created?
