# Identified Issues and Ambiguities

## 1. Commission and Payout Calculation
The specification states: "Commission is deducted from the potential payout." It also defines a formula: `buy_commission = financialrounding('price', currency, ask_price - theo_price)`.
- **Ambiguity:** Since the `Ask Price` (Stake) is a fixed input from the client, `Theo Price` must be calculated. If `Commission` is the difference between Ask and Theo, it implies a markup. However, "deducted from potential payout" suggests the user receives less money upon winning.
- **Question:** Does the commission reduce the final Payout amount, or is it purely a markup that determines how much "fair value" (Theo) goes into the Black-Scholes payout calculation?

## 2. Tick Duration and Expiry
The Pricing Logic states: "For tick-based durations... convert to time duration by multiplying the number of ticks by the tick generation interval."
- **Ambiguity:** Does this conversion apply to the **Lifecycle/Expiry** of the contract as well?
- **Question:** For a '5t' contract, does it expire exactly at `Start Time + (5 * Interval)`, or does it wait for 5 actual market ticks to arrive? This is critical for the "Bid" (Active Contract) implementation and finding the Exit Price.

## 3. Stateless Bid Request and Entry Spot
The `GetBidRequest` requires `OptionParameters` including `start_time` and `barrier`. It does not include `entry_spot`.
- **Issue:** To report the `entry_spot` and `entry_spot_time` in the `GetBidResponse`, the service must know the "first tick after start time".
- **Question:** Does the service need to query a historical data source to find this past tick? Or is the "tick generation interval" approximation used here too?

## 4. Financial Rounding
- **Ambiguity:** The spec mentions `financialrounding`.
- **Question:** What are the specific rules for this rounding?

## 5. Limits and Commission Configuration
- **Question:** The spec mentions "Commission rates are defined in per-symbol configuration". Is this a fixed amount per contract, a percentage of stake, or something else?
