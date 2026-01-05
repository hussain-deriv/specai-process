# Expert Interactions

## Question 1
**Question:** How does the legacy system source market data inputs (spot price, interest rates, volatility) required for the pricing formula?

**Answer:** 
The legacy system sources market data inputs as follows:

1. **Spot Price:**  
   Spot prices are typically fetched from underlying market data sources. In the code, this is handled by:

   ```perl
   use BOM::MarketData qw(create_underlying_db);
   use BOM::MarketData qw(create_underlying);
   ```
   [ContractPricer.pm](https://github.com/regentmarkets/bom-core/blob/master/bom/lib/BOM/Product/ContractPricer.pm)

   These functions query proprietary or third-party sources to provide current spot prices.

2. **Interest Rates:**  
   Interest rates are either defined in static config or loaded from external files or databases.  
   Example from [config.yml](https://github.com/regentmarkets/bom-core/blob/master/bom/t/BOM/Product/Pricing/Engine/config.yml):

   ```yaml
   index:
     OTC_FCHI:
       rates:
         '1': 44.34
         '10': 7.89
         # ...
   ```
   These rates can be mapped to the relevant product and used in pricing engines.

   Additionally, CSV files like `SD_GBPJPY.csv`, `SD_USDJPY.csv`, etc., contain historical or benchmark rates:
   ```
   Depo Base,0.66,0.655,...
   Depo Term,-0.163,-0.168,...
   ```

3. **Volatility:**  
   Volatility surfaces are also loaded from CSV benchmark files (e.g., [SD_GBPJPY.csv](https://github.com/regentmarkets/bom-core/blob/master/bom-quant-benchmark/t/csv/superderivatives/SD_GBPJPY.csv)):

   ```
   START VOL
   Volatility,13.481,12.45,12.799,...
   25 Delta Risk Reversal,-0.008,-0.4,-0.858,...
   25 Delta Butterfly,0.4,0.4,0.4,...
   END VOL
   ```

   These files provide volatilities, risk reversals, and butterflies needed for option pricing.

**Summary:**  
- Spot: `BOM::MarketData` modules
- Rates: Config files ([config.yml](https://github.com/regentmarkets/bom-core/blob/master/bom/t/BOM/Product/Pricing/Engine/config.yml)), and rate CSVs
- Volatility: Benchmark CSVs (e.g., [SD_GBPJPY.csv](https://github.com/regentmarkets/bom-core/blob/master/bom-quant-benchmark/t/csv/superderivatives/SD_GBPJPY.csv))

For each contract, the relevant pricing engine is chosen using configuration ([selection_config.yml](https://github.com/regentmarkets/bom-core/blob/master/bom/t/BOM/Product/Pricing/Engine/selection_config.yml)), and the engine requests the necessary market data from these sources.

## Question 2
**Question:** How does the legacy system handle 'tick' durations (e.g., '5t') in the Black-Scholes pricing formula? Is there a conversion to time, or a different model?

**Answer:**
In the legacy system, contracts with tick-based durations like 5t (5 ticks) are handled with special adjustments to match their actual time duration before passing them to the Black-Scholes pricing model.

### How Tick Durations Are Converted:

- There is a fixed tick generation interval for synthetic indices (e.g., a tick every 2 seconds).
- For contracts with durations less than 5 minutes, instead of directly using the nominal duration, the code adjusts the contract duration to match the actual number of ticks that will occur during the contract duration.
- This adjustment is done so the Black-Scholes model receives a correct time duration that reflects the actual number of monitored ticks, preventing under-pricing of in-the-money contracts.

#### Example from [`Contract.pm`](https://github.com/regentmarkets/bom-core/blob/master/bom/lib/BOM/Product/Contract.pm):

```perl
if (    $self->market->name eq 'synthetic_index'
    and $self->underlying->generation_interval->seconds > 1
    and not($self->is_atm_bet or $self->for_sale or $self->tick_expiry)
    and $time_to_expiry->minutes < 5)
{
    # Adjust start and expiry to align to tick boundaries
    my $date_start_adjustment  = $self->effective_start->epoch % 2 ? 1 : 2;
    my $date_expiry_adjustment = $self->date_expiry->epoch % 2     ? 1 : 0;
    my $actual_duration =
        $self->date_expiry->minus_time_interval($date_expiry_adjustment)->epoch -
        $self->effective_start->plus_time_interval($date_start_adjustment)->epoch;
    $time_to_expiry = Time::Duration::Concise->new(interval => $actual_duration);
}
```

### Summary

- Tick durations are converted to the equivalent time duration based on the tick interval.
- The Black-Scholes model is always used for pricing, but it is passed the actual calculated time duration that corresponds to the number of ticks for the contract.
- No separate model is used for ticks; rather, the time duration is recalibrated before applying Black-Scholes.

You can find more config mappings for tick contracts (like `5T`) using Black-Scholes in [`selection_config.yml`](https://github.com/regentmarkets/bom-core/blob/master/bom/t/BOM/Product/Pricing/Engine/selection_config.yml). Example entry:

```
ASIANU_R_25_100.00_1465693200_5T: Pricing::Engine::BlackScholes
```

**In summary:**  
Tick durations are converted to their equivalent time durations before applying Black-Scholes, rather than using a completely different model.

## Question 3
**Question:** How are 'commission', 'Max Payout', and 'Min Stake' defined and accessed in the legacy system for each symbol?

**Answer:**
Here’s how these key contract parameters are defined and accessed:

---

### 1. **Commission**

#### **Definition**
- **commission**: The commission as a decimal. Example: `0.01` means 1%.
- **commission_amount**: The commission charged in payout currency. Minimum 0.02 per contract.

**Reference**: [Multiplier.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Multiplier.pm)
```
=head2 commission_amount
Commission charged in payout currency amount. A minimum of 0.02 is imposed per contract.

=head2 commission
Commission in decimal amount. E.g. 0.01 = 1%
```
For Binary contracts, commission charged at buy is calculated as:
```perl
sub buy_commission {
    my $self = shift;
    return financialrounding('price', $self->currency, $self->ask_price - $self->theo_price);
}
```
— [Binary.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Binary.pm)

---

### 2. **Max Payout**

#### **Definition**
**Max Payout** is the maximum payout a user can receive from a contract.

**Reference**: [proposal.md](file:///home/git/regentmarkets/bom-core/binary-websocket-api/docs/endpoints/proposal.md)
| Name | Type | Description |
|------|------|-------------|
| proposal.contract_details.maximum_payout | number | Maximum payout that user can get out of a contract, contract will close automatically if payout reaches this number |

Access for binaries:
```perl
sub _build_max_payout {
    return shift->staking_limits->{max};
}
```
— [Binary.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Binary.pm)

For turbos and multipliers, similar configurations exist (under per-symbol configs).

---

### 3. **Min Stake**

#### **Definition**
**Min Stake** is the minimum allowed stake to place a contract.

- [proposal.md](file:///home/git/regentmarkets/bom-core/binary-websocket-api/docs/endpoints/proposal.md) describes:
  | proposal.contract_details.minimum_stake | string | Minimum stake that user can set to buy a contract |

- In JSON config ([receive.json](file:///home/git/regentmarkets/bom-core/binary-websocket-api/config/v3/contracts_for/receive.json)):
```json
"min_stake": {
  "description": "[Only for turbos options] Minimum contract stake",
  "type": ["null", "number"]
}
```

#### **Access/Computation**
##### For Multiplier
```perl
sub _build_min_stake {
    my $self = shift;
    my $default_min_stake = get_exchangerates_limit($self->per_symbol_config()->{min_stake_per_contract}, $self->currency);
    my $min_stake         = min($default_min_stake, $self->max_stake);
    return beautify_stake($min_stake, $self->currency, 1);
}
```
— [Multiplier.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Multiplier.pm)

##### For Binary
```perl
sub _build_min_stake {
    return shift->staking_limits->{min};
}
```
— [Binary.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Binary.pm)

---

### **Per-Symbol Config Access**
All these values (`min_stake_per_contract`, `max_stake_per_contract`, etc.) are typically accessed using:
- `$self->per_symbol_config->{parameter_name}`

They're further processed by helper functions like `get_exchangerates_limit()` for currency adjustment, and rounding/beautification functions.

---

### **Summary Table**

| Parameter    | Definition                     | Access Pattern / Calculation                    | Reference File |
|--------------|-------------------------------|--------------------------------------------------|---------------|
| Commission   | Percentage or absolute amount  | `$self->commission`, `buy_commission`           | [Multiplier.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Multiplier.pm), [Binary.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Binary.pm) |
| Max Payout   | Max contract payout            | `$self->staking_limits->{max}` or per-symbol    | [Binary.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Binary.pm) |
| Min Stake    | Minimum contract stake         | `$self->staking_limits->{min}` or per-symbol    | [Multiplier.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Multiplier.pm), [Binary.pm](file:///home/git/regentmarkets/bom-core/bom/lib/BOM/Product/Role/Binary.pm) |

Let me know if you need a concrete code translation for any of these parts.

## Question 4
**Question:** Is there a specific Black-Scholes variant (e.g., Cash-or-Nothing) used for binary options in the legacy system?

**Answer:**
Yes, the legacy system uses a specific variant of Black-Scholes for binary options: the **Cash-or-Nothing** option.

This is evidenced by the frequent use of pricing engines called `Pricing::Engine::BlackScholes` across many binary option types—such as ONETOUCH, NOTOUCH, CALL, and PUT—in the [selection_config.yml](file:///home/git/regentmarkets/bom-core/bom/t/BOM/Product/Pricing/Engine/selection_config.yml) file. In industry and financial mathematics, the "BlackScholes" engine with binary payouts (fixed cash amount if in-the-money, zero otherwise) describes the Cash-or-Nothing (digital) Black-Scholes variant.

For example:
```yaml
ONETOUCH_R_100_100.00_1465693200_1465862399_104650000_0: Pricing::Engine::BlackScholes
CALL_R_100_100.00_1465693200_1465862399_104650000_0: Pricing::Engine::BlackScholes
```

These entries show that almost all binary contracts (pays a fixed amount if a condition is met) are priced using this Black-Scholes variant.

**Summary:**  
Binary (digital) contracts in the legacy system are priced with the Cash-or-Nothing Black-Scholes formula, via the `Pricing::Engine::BlackScholes` engine.

Reference: [selection_config.yml](file:///home/git/regentmarkets/bom-core/bom/t/BOM/Product/Pricing/Engine/selection_config.yml)
