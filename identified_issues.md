# Identified Issues in Product Specification

## 1. Missing Data Source Definitions
The "Pricing Logic" section lists inputs such as "Spot price", "Payout currency interest rate", "Quanto drift", and "Volatility". However, the API definitions (`GetAskRequest`, `StreamAskRequest`) do not include these as parameters. It is undefined how the service will access this market data (e.g., via an external Feed service, Redis, or database).

## 2. Ambiguity in Tick-based Pricing
The specification allows duration in ticks (e.g., "5t"). The Black-Scholes pricing model relies on time to expiry (T) as a continuous variable. It is unclear how "ticks" should be converted to a time value for the pricing formula, or if a different model should be used for tick-based contracts.

## 3. Configuration Storage
The spec states that limits (Max Payout, Min Stake) and commission are "defined by symbol". There is no definition of where this configuration resides or how the service should access it.

## 4. Commission Calculation Specifics
The statement "Commission is deducted from potential payout" is ambiguous. It does not specify if the commission is a fixed amount, a percentage of the stake, or a percentage of the payout. A precise formula is required.

## 5. Black-Scholes Model Specifics
"Standard black & scholes" is mentioned. For binary options, there are specific variations (e.g., Cash-or-Nothing Call/Put). The spec should explicitly state the formula or variant to ensure parity with the legacy system.

## 6. Environment Specific Instructions
The "Technical Implementation Guide" includes a hardcoded path (`cd /Users/junbon/Project`) and assumes a specific local environment state (`check if go-templates command exists`). These should be generalized for the build environment.
