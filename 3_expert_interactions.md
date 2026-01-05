# Expert Interactions

## 1. Commission and Payout
**Question:** How is commission calculated for Digital Call/Put options? Does it reduce the final Payout amount, or is it purely a markup? The spec mentions "Commission is deducted from the potential payout" but also gives a formula based on ask_price - theo_price.

**Expert Answer:**
Commission is implemented as a **pricing markup**, not an explicit deduction from the payout at expiry.

*   **Buying (Ask):** `commission = number_of_contracts * (ask_probability - theo_probability)`. The client pays this markup upfront in the Ask Price (Stake).
*   **Selling (Bid):** `commission = number_of_contracts * (theo_probability - bid_probability)`. This is the cost for early exit.
*   **Expiry:** No commission is deducted from the payout if held to expiry.

## 2. Tick Duration and Expiry
**Question:** For tick-based durations (e.g., '5t'), how is the expiry determined? Does the contract expire exactly at `Start Time + (5 * tick_interval)` (time-based approximation), or does it wait for 5 actual market ticks to arrive (event-based)?

**Expert Answer:**
The expiry is **event-based**. The system waits for the N-th tick after start to arrive.
*   "We wait for the n-th tick to settle tick expiry contract."
*   There is a maximum waiting period (e.g., 5 minutes).
*   It is not a time-based approximation using the interval.

## 3. Entry Spot for Active Contracts
**Question:** For active contracts (GetBidRequest), how does the system determine the 'Entry Spot' and 'Entry Spot Time' if the request is stateless and only provides 'Start Time' and 'Barrier'? Does it need to query historical ticks to find the first tick after Start Time?

**Expert Answer:**
Yes, the system must query **historical market data**.
*   **Entry Spot:** The first valid underlying spot price at or after the contract's `start_time`.
*   **Entry Spot Time:** The epoch time of that entry spot.
*   If the request is stateless, the system must look up this tick using the `start_time`.

## 4. Financial Rounding
**Question:** What are the specific rules for 'financialrounding'? How many decimal places, what rounding method (half-up, floor, etc.), and does it vary by currency?

**Expert Answer:**
*   **Precision:** Varies by currency (e.g., JPY=0, USD=2).
*   **Method:** Most likely "nearest" (half-up).
*   **Context:** Used for ensuring amounts and commissions align with currency precision.

## 5. Limits Configuration
**Question:** How are 'Limits' (Max Payout, Min Stake) and 'Commission Rates' configured? Are they static values in a config file (like 'config.yml'), or dynamically loaded? The spec mentions 'per-symbol configuration'.

**Expert Answer:**
Limits are **dynamically loaded** from per-symbol configuration files (YAML).
*   Values like `min_stake_per_contract` and `max_payout_per_contract` are specific to the symbol and contract category.
*   They are not hardcoded but sourced from files like `contract_types.yml` and `contract_categories.yml`.
