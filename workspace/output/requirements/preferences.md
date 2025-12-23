# Requirements Phase Preferences

This document captures all user directives, decisions, and clarifications made during the requirements phase.

---

## Entry 1: Complexity Assessment
**Type**: Assessment
**Assessment Scores**:
- Technical Complexity: 2/3 (gRPC microservice, Black-Scholes pricing, streaming, service integration)
- User Complexity: 0/2 (Single user type, no auth)
- Integration Complexity: 1/2 (One critical dependency: service-feed)
- Regulatory/Compliance: 2/2 (Financial trading service)
- Scale/Performance: 1/1 (Real-time pricing, streaming)
**Total Score**: 6/10
**Template Selected**: Standard PRD (templates/prd_standard.md)
**User Confirmation**: Approved
**Date**: 2025-12-22

---

## Entry 2: Pricing Parameters Configuration
**Type**: Directive
**User Input**: "Payout currency interest rate and quanto drift can be set to 0. Volatility set to 10%. Commission set to 2%."
**Context**: Clarification on missing pricing inputs from the product brief
**Impact**: 
- Interest rate: 0% (no interest rate consideration)
- Quanto drift: 0 (no quanto adjustment)
- Volatility: 10% (fixed volatility for all calculations)
- Commission: 2% (deducted from payout)
**Date**: 2025-12-22

---

## Entry 3: Trading Limits Storage
**Type**: Question/Answer
**Question**: "Where should trading limits (min stake, max payout) be stored?"
**User Input**: "Trading limit can be stored in configuration files."
**Context**: Determining the source of trading limits per symbol
**Impact**: Limits will be read from configuration files (not database or external service)
**Date**: 2025-12-22

---

## Entry 4: Barrier Validation
**Type**: Question/Answer
**Question**: "What should happen if an invalid barrier is provided?"
**User Input**: "Standard gRPC error will be returned to client if barrier is invalid."
**Context**: Error handling for barrier validation
**Impact**: Use standard gRPC error codes for validation failures
**Date**: 2025-12-22

---

## Entry 5: Contract Update Timing
**Type**: Question/Answer
**Question**: "How should contract price updates be timed - on tick arrival or 5-second intervals?"
**User Input**: "5-second rule is a fallback when no ticks arrive"
**Context**: Clarifying the contract lifecycle update frequency
**Impact**: 
- Primary: Update on every new tick received
- Fallback: If no tick received, update every 5 seconds
**Date**: 2025-12-22

---

## Entry 6: Service Name
**Type**: Directive
**User Input**: "${SERVICE_NAME} can be digitalcallput"
**Context**: Confirming the actual service name for repository and module naming
**Impact**: 
- Repository: service-pricer-digitalcallput
- Module: github.com/regentmarkets/service-pricer-digitalcallput
- Package: digitalcallput
**Date**: 2025-12-22

---

## Entry 7: Configuration Management
**Type**: Question/Answer
**Question**: "Should configuration be per-symbol or global?"
**User Input**: "1:B - Global configuration (same for all symbols)"
**Context**: Determining configuration scope for volatility, commission, and limits
**Impact**: Single global configuration applies to all symbols
**Date**: 2025-12-22

---

## Entry 8: Commission Visibility
**Type**: Question/Answer
**Question**: "Should commission be visible in responses?"
**User Input**: "2:B - No, just show the final ask/bid price after commission"
**Context**: Determining response structure for pricing
**Impact**: Commission is deducted internally, not exposed in API responses
**Date**: 2025-12-22

---

## Entry 9: Barrier Validation Rules
**Type**: Question/Answer
**Question**: "Are there limits on relative barrier values?"
**User Input**: "3:A - No limits - accept any relative barrier value"
**Context**: Validation rules for relative barriers
**Impact**: Accept any relative barrier value (e.g., +100, -50, etc.) without range restrictions
**Date**: 2025-12-22

---

## Entry 10: Stream Termination
**Type**: Question/Answer
**Question**: "When should streaming endpoints terminate?"
**User Input**: "4:C - Both client close and contract expiry"
**Context**: Stream lifecycle management
**Impact**: 
- StreamAsk: Terminates when client closes connection
- StreamBid: Terminates when client closes OR contract expires
**Date**: 2025-12-22

---

## Entry 11: Error Handling Strategy
**Type**: Question/Answer
**Question**: "What validation errors should return gRPC errors?"
**User Input**: "5:C - All validation errors with specific error codes for each type"
**Context**: Comprehensive error handling requirements
**Impact**: Implement specific gRPC error codes for:
- Invalid symbol
- Invalid duration format
- Stake below minimum
- Invalid currency
- Negative stake
- Invalid barrier
- Missing required fields
**Date**: 2025-12-22

---

## Summary of Key Decisions

### Technical Configuration
- Volatility: 10% (global)
- Commission: 2% (global, not visible in responses)
- Interest Rate: 0%
- Quanto Drift: 0
- Configuration Storage: Files (not database)

### Service Identity
- Service Name: digitalcallput
- Repository: service-pricer-digitalcallput
- Module Path: github.com/regentmarkets/service-pricer-digitalcallput

### Validation & Error Handling
- Barrier: No range limits, standard gRPC errors for invalid values
- Comprehensive validation with specific error codes
- Standard gRPC error responses

### Operational Behavior
- Update Timing: On tick (primary), 5-second fallback
- Stream Termination: Client close or expiry (for Bid)
- Configuration Scope: Global (all symbols)

---

## Entry 12: Tick Duration Behavior Correction
**Type**: Modification
**Current State**: Tick duration contracts (e.g., '5t') must expire ONLY after receiving the specified number of ticks. There is NO time-based fallback. If no ticks are received, the contract remains active indefinitely until the required number of ticks arrive.
**Previous State**: Incorrectly specified that tick duration contracts would fall back to 5-second intervals if no ticks received
**Context**: User identified an error in PRD section 4.2.3 where tick duration behavior was incorrectly documented with a time-based fallback
**Impact**:
- PRD section 4.2.3 (Duration Parsing) needs correction
- Tick duration contracts must wait for actual tick arrivals
- No time-based expiry for tick-based contracts
- This is a fundamental business rule that affects contract lifecycle
**Date**: 2025-12-23
