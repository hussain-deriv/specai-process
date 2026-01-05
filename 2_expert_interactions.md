# Code Expert Interactions

## Interaction 1
**Question:**
Regarding the Digital Call/Put options: The spec lists 'Stake' as a required input, but also mentions calculating Payout. In the legacy Perl system, does the user strictly specify the 'Stake' (and the system calculates Payout), or can the user specify a desired 'Payout' (and the system calculates the required Stake)? If they specify Stake, is the 'Ask Price' returned simply equal to the input Stake?

**Answer:**
In the legacy Perl system for Digital Call/Put options (binary options):

- **The user strictly specifies the "Stake" as input.**
- The system **calculates the payout** based on this stake and contract parameters.
- **The user cannot specify the desired "Payout"** (there is no reverse calculation from payout to stake).

Regarding "Ask Price":
- **Ask Price is simply equal to the input Stake.**  
  This is confirmed by the code in [Binary.pm](https://github.com/regentmarkets/bom-core/blob/master/bom/lib/BOM/Product/Role/Binary.pm):

```perl
override _build_ask_price => sub {
    my $self = shift;

    return $self->_user_input_stake if defined $self->_user_input_stake;
    # ... fallback not relevant for standard case
};
```

So when the user specifies a stake, the system returns "ask price" == "stake".

**Summary:**  
- User-input: Stake  
- System: Calculates Payout  
- Ask Price: Equals input Stake

**References:**  
- [Binary.pm](https://github.com/regentmarkets/bom-core/blob/master/bom/lib/BOM/Product/Role/Binary.pm)
- [ContractPricer.pm](https://github.com/regentmarkets/bom-core/blob/master/bom/lib/BOM/Product/ContractPricer.pm)  
- [ContractValidator.pm](https://github.com/regentmarkets/bom-core/blob/master/bom/lib/BOM/Product/ContractValidator.pm)

## Interaction 2
**Question:**
For the 'Ask' request (pricing a proposal before purchase) where the contract has not started yet:
1. Does the legacy system use the *Current Spot Price* as the underlying price for the Black-Scholes calculation?
2. How are relative barriers (e.g., '+10') calculated during this 'Ask' phase? Does it use the Current Spot Price as the reference since there is no 'Entry Price' yet?

**Answer:**
1. **Spot Price Used for Black-Scholes in 'Ask' Phase**  
For the 'Ask' request before contract start (i.e., before purchase), the system uses the **current spot price** as the underlying price in the Black-Scholes calculation.

This is reflected in [`Contract.pm`](./bom/lib/BOM/Product/Contract.pm):

```perl
my $initial_spot;
if ($self->current_tick) {
    $initial_spot = $self->current_tick->quote;
} else {
    # fallback if unavailable, still aims for latest available spot
    ...
}
```
So, for proposals ('Ask'), the current spot price is always used.

2. **Calculation of Relative Barriers in 'Ask' Phase**  
During the 'Ask' phase, for contracts with relative barriers (like '+10'), the system uses the **current spot price** as the reference, since there is no contract entry price yet.

Relative barriers are resolved as:
```
Barrier = Current Spot Price + Relative Offset
```
Again, this is because there is no definitive 'entry' price yet at the proposal stage, so the current spot price is the only available reference.

**Summary:**  
- **Spot for pricing at 'Ask':** Always the current spot.
- **Relative barrier reference:** Also the current spot at proposal time.  

**References:**
- [`Contract.pm`](./bom/lib/BOM/Product/Contract.pm) - Logic on which spot is used

## Interaction 3
**Question:**
For the `GetBidRequest` (selling an active contract):
1. The spec input parameters are just `OptionParameters` and `pricing_time`. Does the legacy system require the caller to pass the resolved *absolute* barrier, or can it accept the relative barrier string?
2. If relative, does the service look up the historical `entry_spot` to calculate the absolute barrier? If so, does the system have access to historical ticks, or is the `entry_spot` expected to be passed in the request?

**Answer:**
1. The legacy system expects the caller to pass the resolved **absolute barrier** when selling an active contract.

   This is evident from [Contract.pm](../bom/lib/BOM/Product/Contract.pm):  
   ```perl
   foreach my $barrier ($self->two_barriers ? ('high_barrier', 'low_barrier') : ('barrier')) {
       if (defined $self->$barrier) {
           $opp_parameters{$barrier} = $self->$barrier->as_absolute;
           $opp_parameters{'supplied_' . $barrier} = $self->$barrier->as_absolute;
       }
   }
   ```
   The `.as_absolute` indicates that the barrier should be passed in absolute terms, regardless of how it was originally input.

2. For existing contracts, the service does **not** expect a relative barrier string. The absolute barrier—as originally recorded for the contract—is already known to the backend, since contract creation always resolves and stores the absolute value based on the `entry_spot` at that time.

   Thus:

   - The service does not need to calculate the barrier at sell time;
   - The system has access to the historical `entry_spot` for the contract, because it's tied to the contract record;
   - The `entry_spot` is **not** expected to be passed in the sell request.

**Summary:**  
For selling, the absolute barrier is used. The system already knows the `entry_spot` and the resolved barrier values for active contracts, so no special barrier calculation is needed at bid time.

*Reference*:  
- [BOM/Product/Contract.pm](../bom/lib/BOM/Product/Contract.pm)

## Interaction 4
**Question:**
Regarding tick durations (e.g., '5t'):
1. How does the legacy system convert tick duration to time duration for the Black-Scholes model?
2. Where is the 'tick generation interval' defined? Is it in a configuration file or calculated from recent tick history?

**Answer:**
1. **How does the legacy system convert tick duration to time duration for the Black-Scholes model?**

The system adjusts contract duration based on the tick generation interval, especially for contracts less than 5 minutes on volatility indices. For tick-based durations (e.g., '5t'), it estimates the real time by multiplying the number of ticks by the generation interval, ensuring the Black-Scholes input uses an accurate time duration.

Relevant code:

```perl
if (    $self->market->name eq 'synthetic_index'
    and $self->underlying->generation_interval->seconds > 1
    and not($self->is_atm_bet or $self->for_sale or $self->tick_expiry)
    and $time_to_expiry->minutes < 5)
{
    # The actual duration is adjusted based on date_start/date_expiry, considering the tick interval.
    my $actual_duration = ...; # see code for details
    $time_to_expiry = Time::Duration::Concise->new(interval => $actual_duration);
}
```
See: [Contract.pm](../bom/lib/BOM/Product/Contract.pm)

Also, for tick expiry contracts, there is a maximum tick expiry duration set to 5 minutes:

```perl
sub _max_tick_expiry_duration {
    return Time::Duration::Concise->new(interval => '5m');
}
```
See: [Contract.pm - _max_tick_expiry_duration](../bom/lib/BOM/Product/Contract.pm)

2. **Where is the 'tick generation interval' defined? Is it in a configuration file or calculated from recent tick history?**

The tick generation interval is accessed via the contract’s underlying market object:  
`$self->underlying->generation_interval->seconds`

This is typically a property of the underlying market or symbol (e.g., for 'synthetic_index'), and it is not specified in a flat configuration file but is most likely configured per underlying or market, possibly in code or in-market definitions.

There is no indication from the referenced information that it's dynamically calculated from tick history – it appears to be statically set.

See reference: [Contract.pm](../bom/lib/BOM/Product/Contract.pm) (search for `generation_interval`).

**Summary:**  
- Tick-based durations are converted to time based on a per-underlying/static tick interval, not from recent history.
- That interval is a property of the underlying or market, accessed via methods, not a config file.
