package pricing

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FeedSubscriber defines the interface for market data feed (Rule 5.5: Interface at Consumer)
type FeedSubscriber interface {
	GetCurrentTick(ctx context.Context, symbol string) (*Tick, error)
	Subscribe(ctx context.Context, symbol string) (<-chan *Tick, error)
	StreamWithFallback(ctx context.Context, symbol string, fallbackDuration time.Duration) (<-chan *Tick, error)
}

// Tick represents a real-time market data point from service-feed
type Tick struct {
	Symbol    string
	Price     float64
	Timestamp time.Time
}

// OptionParams represents the internal option parameters
type OptionParams struct {
	Symbol       string
	ContractType pb.ContractType
	Currency     string
	Stake        float64
	Duration     Duration
	Barrier      *Barrier
	StartTime    *time.Time
}

// AskQuote represents the result of ask price calculation
type AskQuote struct {
	AskPrice        float64
	Currency        string
	CurrentSpot     float64
	CurrentSpotTime time.Time
	Payout          float64
	Limits          *TradingLimits
}

// ToProto converts AskQuote to protobuf response
func (q *AskQuote) ToProto() *pb.GetAskResponse {
	return &pb.GetAskResponse{
		AskPrice:        formatDecimal(q.AskPrice),
		Currency:        q.Currency,
		CurrentSpot:     formatDecimal(q.CurrentSpot),
		CurrentSpotTime: q.CurrentSpotTime.Unix(),
		Payout:          formatDecimal(q.Payout),
		Limits: &pb.Limits{
			MaxPayout: formatDecimal(q.Limits.MaxPayout),
			MinStake:  formatDecimal(q.Limits.MinStake),
		},
	}
}

// BidQuote represents the result of bid price calculation
type BidQuote struct {
	BidPrice        float64
	IsExpired       bool
	CurrentSpot     float64
	CurrentSpotTime time.Time
	EntrySpot       float64
	EntrySpotTime   time.Time
	ExitSpot        *float64
	ExitSpotTime    *time.Time
	Barrier         float64
	StartTime       time.Time
	ExpiryTime      time.Time
	Currency        string
}

// ToProto converts BidQuote to protobuf response
func (q *BidQuote) ToProto() *pb.GetBidResponse {
	resp := &pb.GetBidResponse{
		BidPrice:        formatDecimal(q.BidPrice),
		IsExpired:       q.IsExpired,
		CurrentSpot:     formatDecimal(q.CurrentSpot),
		CurrentSpotTime: q.CurrentSpotTime.Unix(),
		EntrySpot:       formatDecimal(q.EntrySpot),
		EntrySpotTime:   q.EntrySpotTime.Unix(),
		Barrier:         formatDecimal(q.Barrier),
		StartTime:       q.StartTime.Unix(),
		ExpiryTime:      q.ExpiryTime.Unix(),
		Currency:        q.Currency,
	}

	if q.ExitSpot != nil {
		resp.ExitSpot = formatDecimal(*q.ExitSpot)
	}

	if q.ExitSpotTime != nil {
		resp.ExitSpotTime = q.ExitSpotTime.Unix()
	}

	return resp
}

// AskSubscription represents an active ask price stream
type AskSubscription struct {
	quotes chan *AskQuote
	err    error
	done   chan struct{}
}

// Quotes returns the read-only quote channel
func (s *AskSubscription) Quotes() <-chan *AskQuote {
	return s.quotes
}

// Err returns any error that occurred during streaming
func (s *AskSubscription) Err() error {
	return s.err
}

// Close terminates the subscription
func (s *AskSubscription) Close() {
	close(s.done)
}

// BidSubscription represents an active bid price stream
type BidSubscription struct {
	quotes chan *BidQuote
	err    error
	done   chan struct{}
}

// Quotes returns the read-only quote channel
func (s *BidSubscription) Quotes() <-chan *BidQuote {
	return s.quotes
}

// Err returns any error that occurred during streaming
func (s *BidSubscription) Err() error {
	return s.err
}

// Close terminates the subscription
func (s *BidSubscription) Close() {
	close(s.done)
}

// TradingLimits contains trading constraints
type TradingLimits struct {
	MinStake  float64
	MaxPayout float64
}

// PricingConfig contains global pricing parameters
type PricingConfig struct {
	Volatility   float64 // Annual volatility (σ)
	Commission   float64 // Commission rate
	InterestRate float64 // Risk-free rate (r)
	QuantoDrift  float64 // Quanto adjustment (q)
}

// formatDecimal formats a float64 as a decimal string with 2 decimal places for money
func formatDecimal(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

// Calculator orchestrates all pricing calculations
type Calculator struct {
	config *PricingConfig
	limits *TradingLimits
	feed   FeedSubscriber
}

// NewCalculator creates a new pricing calculator
func NewCalculator(config *PricingConfig, limits *TradingLimits, feed FeedSubscriber) *Calculator {
	return &Calculator{
		config: config,
		limits: limits,
		feed:   feed,
	}
}

// CalculateAsk calculates the ask price for a new contract proposal
func (c *Calculator) CalculateAsk(ctx context.Context, params *pb.OptionParameters) (*AskQuote, error) {
	// 1. Validate and parse parameters
	optionParams, err := c.parseAndValidateParams(params, false)
	if err != nil {
		return nil, err
	}

	// 2. Get current market tick
	tick, err := c.feed.GetCurrentTick(ctx, optionParams.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Market data feed unavailable: %v", err)
	}

	// 3. Calculate time to expiry
	timeToExpiry, err := c.calculateTimeToExpiry(optionParams.Duration, time.Now())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to calculate expiry: %v", err)
	}

	// 4. Resolve barrier
	barrier := tick.Price // Default to entry spot
	if optionParams.Barrier != nil {
		barrier, err = optionParams.Barrier.ResolveBarrier(tick.Price)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to resolve barrier: %v", err)
		}
	}

	// 5. Calculate probability using Black-Scholes
	probability := BlackScholesDigitalOption(
		tick.Price,
		barrier,
		timeToExpiry,
		c.config.Volatility,
		c.config.InterestRate,
		c.config.QuantoDrift,
		optionParams.ContractType,
	)

	// 6. Calculate payout
	payout := CalculateAskPayout(optionParams.Stake, probability, c.config.Commission)

	// 7. Build and return quote
	return &AskQuote{
		AskPrice:        optionParams.Stake, // Ask price equals stake
		Currency:        optionParams.Currency,
		CurrentSpot:     tick.Price,
		CurrentSpotTime: tick.Timestamp,
		Payout:          payout,
		Limits:          c.limits,
	}, nil
}

// CalculateBid calculates the bid price for an active contract
func (c *Calculator) CalculateBid(ctx context.Context, params *pb.OptionParameters) (*BidQuote, error) {
	// 1. Validate and parse parameters (bid requires start_time)
	optionParams, err := c.parseAndValidateParams(params, true)
	if err != nil {
		return nil, err
	}

	// 2. Get current market tick
	tick, err := c.feed.GetCurrentTick(ctx, optionParams.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Market data feed unavailable: %v", err)
	}

	// 3. Determine entry spot (for now, use start time price)
	// In a real implementation, this would fetch the first tick after start_time
	entrySpot := tick.Price // Simplified for initial implementation
	entrySpotTime := *optionParams.StartTime

	// 4. Resolve barrier based on entry spot
	barrier := entrySpot // Default
	if optionParams.Barrier != nil {
		barrier, err = optionParams.Barrier.ResolveBarrier(entrySpot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to resolve barrier: %v", err)
		}
	}

	// 5. Calculate expiry time
	var expiryTime time.Time
	if optionParams.Duration.IsTimeBased() {
		expiryTime, err = optionParams.Duration.CalculateExpiryTime(*optionParams.StartTime)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to calculate expiry: %v", err)
		}
	} else {
		// Tick-based: expiry is not time-based
		// For now, set a far future time
		expiryTime = time.Now().Add(365 * 24 * time.Hour)
	}

	// 6. Check if contract has expired
	now := time.Now()
	isExpired := now.After(expiryTime) || now.Equal(expiryTime)

	// 7. Calculate payout (same formula as ask)
	timeToExpiry := expiryTime.Sub(*optionParams.StartTime).Seconds() / (365.25 * 24 * 60 * 60)
	probability := BlackScholesDigitalOption(
		entrySpot,
		barrier,
		timeToExpiry,
		c.config.Volatility,
		c.config.InterestRate,
		c.config.QuantoDrift,
		optionParams.ContractType,
	)
	payout := CalculateAskPayout(optionParams.Stake, probability, c.config.Commission)

	// 8. Determine bid price
	var bidPrice float64
	var exitSpot *float64
	var exitSpotTime *time.Time

	if isExpired {
		// Contract expired - check win condition
		exitSpotVal := tick.Price
		exitSpot = &exitSpotVal
		exitSpotTimeVal := tick.Timestamp
		exitSpotTime = &exitSpotTimeVal

		hasWon := c.checkWinCondition(tick.Price, barrier, optionParams.ContractType)
		bidPrice = CalculateBidPrice(payout, probability, true, hasWon)
	} else {
		// Contract active - calculate current value
		remainingTime := expiryTime.Sub(now).Seconds() / (365.25 * 24 * 60 * 60)
		currentProbability := BlackScholesDigitalOption(
			tick.Price,
			barrier,
			remainingTime,
			c.config.Volatility,
			c.config.InterestRate,
			c.config.QuantoDrift,
			optionParams.ContractType,
		)
		bidPrice = CalculateBidPrice(payout, currentProbability, false, false)
	}

	// 9. Build and return quote
	return &BidQuote{
		BidPrice:        bidPrice,
		IsExpired:       isExpired,
		CurrentSpot:     tick.Price,
		CurrentSpotTime: tick.Timestamp,
		EntrySpot:       entrySpot,
		EntrySpotTime:   entrySpotTime,
		ExitSpot:        exitSpot,
		ExitSpotTime:    exitSpotTime,
		Barrier:         barrier,
		StartTime:       *optionParams.StartTime,
		ExpiryTime:      expiryTime,
		Currency:        optionParams.Currency,
	}, nil
}

// StreamAsk creates a subscription that streams ask price updates
func (c *Calculator) StreamAsk(ctx context.Context, params *pb.OptionParameters) (*AskSubscription, error) {
	// 1. Validate parameters
	optionParams, err := c.parseAndValidateParams(params, false)
	if err != nil {
		return nil, err
	}

	// 2. Create subscription
	sub := &AskSubscription{
		quotes: make(chan *AskQuote, 10),
		done:   make(chan struct{}),
	}

	// 3. Subscribe to feed with 5-second fallback
	tickChan, err := c.feed.StreamWithFallback(ctx, optionParams.Symbol, 5*time.Second)
	if err != nil {
		return nil, err
	}

	// 4. Start goroutine to handle streaming
	go func() {
		defer close(sub.quotes)

		// Send initial quote immediately
		quote, err := c.calculateAskQuote(ctx, optionParams)
		if err != nil {
			sub.err = err
			return
		}

		select {
		case sub.quotes <- quote:
		case <-ctx.Done():
			sub.err = ctx.Err()
			return
		case <-sub.done:
			return
		}

		// Stream updates on each tick
		for {
			select {
			case <-ctx.Done():
				sub.err = ctx.Err()
				return

			case <-sub.done:
				return

			case _, ok := <-tickChan:
				if !ok {
					return
				}

				// Recalculate with new market data
				quote, err := c.calculateAskQuote(ctx, optionParams)
				if err != nil {
					sub.err = err
					return
				}

				select {
				case sub.quotes <- quote:
				case <-ctx.Done():
					sub.err = ctx.Err()
					return
				case <-sub.done:
					return
				}
			}
		}
	}()

	return sub, nil
}

// StreamBid creates a subscription that streams bid price updates
func (c *Calculator) StreamBid(ctx context.Context, params *pb.OptionParameters) (*BidSubscription, error) {
	// 1. Validate parameters (bid requires start_time)
	optionParams, err := c.parseAndValidateParams(params, true)
	if err != nil {
		return nil, err
	}

	// 2. Parse duration to check if tick-based
	duration, err := ParseDuration(params.Duration)
	if err != nil {
		return nil, err
	}

	// 3. Create subscription
	sub := &BidSubscription{
		quotes: make(chan *BidQuote, 10),
		done:   make(chan struct{}),
	}

	// 4. Subscribe to feed (with or without fallback)
	var tickChan <-chan *Tick
	if duration.IsTimeBased() {
		// Time-based: use 5-second fallback
		tickChan, err = c.feed.StreamWithFallback(ctx, optionParams.Symbol, 5*time.Second)
	} else {
		// Tick-based: NO fallback, only tick arrivals
		tickChan, err = c.feed.Subscribe(ctx, optionParams.Symbol)
	}
	if err != nil {
		return nil, err
	}

	// 5. Start goroutine to handle streaming
	go func() {
		defer close(sub.quotes)

		// Send initial quote immediately
		quote, err := c.calculateBidQuote(ctx, optionParams)
		if err != nil {
			sub.err = err
			return
		}

		select {
		case sub.quotes <- quote:
		case <-ctx.Done():
			sub.err = ctx.Err()
			return
		case <-sub.done:
			return
		}

		// Check if already expired
		if quote.IsExpired {
			return
		}

		// Stream updates on each tick until expiry
		for {
			select {
			case <-ctx.Done():
				sub.err = ctx.Err()
				return

			case <-sub.done:
				return

			case _, ok := <-tickChan:
				if !ok {
					return
				}

				// Recalculate with new market data
				quote, err := c.calculateBidQuote(ctx, optionParams)
				if err != nil {
					sub.err = err
					return
				}

				select {
				case sub.quotes <- quote:
				case <-ctx.Done():
					sub.err = ctx.Err()
					return
				case <-sub.done:
					return
				}

				// Auto-terminate stream when contract expires
				if quote.IsExpired {
					return
				}
			}
		}
	}()

	return sub, nil
}

// calculateAskQuote performs ask calculation using internal OptionParams
func (c *Calculator) calculateAskQuote(ctx context.Context, params *OptionParams) (*AskQuote, error) {
	// Get current market tick
	tick, err := c.feed.GetCurrentTick(ctx, params.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Market data feed unavailable: %v", err)
	}

	// Calculate time to expiry
	timeToExpiry, err := c.calculateTimeToExpiry(params.Duration, time.Now())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to calculate expiry: %v", err)
	}

	// Resolve barrier
	barrier := tick.Price // Default to entry spot
	if params.Barrier != nil {
		barrier, err = params.Barrier.ResolveBarrier(tick.Price)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to resolve barrier: %v", err)
		}
	}

	// Calculate probability using Black-Scholes
	probability := BlackScholesDigitalOption(
		tick.Price,
		barrier,
		timeToExpiry,
		c.config.Volatility,
		c.config.InterestRate,
		c.config.QuantoDrift,
		params.ContractType,
	)

	// Calculate payout
	payout := CalculateAskPayout(params.Stake, probability, c.config.Commission)

	// Build and return quote
	return &AskQuote{
		AskPrice:        params.Stake, // Ask price equals stake
		Currency:        params.Currency,
		CurrentSpot:     tick.Price,
		CurrentSpotTime: tick.Timestamp,
		Payout:          payout,
		Limits:          c.limits,
	}, nil
}

// calculateBidQuote performs bid calculation using internal OptionParams
func (c *Calculator) calculateBidQuote(ctx context.Context, params *OptionParams) (*BidQuote, error) {
	// Get current market tick
	tick, err := c.feed.GetCurrentTick(ctx, params.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Market data feed unavailable: %v", err)
	}

	// Determine entry spot (for now, use start time price)
	// In a real implementation, this would fetch the first tick after start_time
	entrySpot := tick.Price // Simplified for initial implementation
	entrySpotTime := *params.StartTime

	// Resolve barrier based on entry spot
	barrier := entrySpot // Default
	if params.Barrier != nil {
		barrier, err = params.Barrier.ResolveBarrier(entrySpot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to resolve barrier: %v", err)
		}
	}

	// Calculate expiry time
	var expiryTime time.Time
	if params.Duration.IsTimeBased() {
		expiryTime, err = params.Duration.CalculateExpiryTime(*params.StartTime)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to calculate expiry: %v", err)
		}
	} else {
		// Tick-based: expiry is not time-based
		// For now, set a far future time
		expiryTime = time.Now().Add(365 * 24 * time.Hour)
	}

	// Check if contract has expired
	now := time.Now()
	isExpired := now.After(expiryTime) || now.Equal(expiryTime)

	// Calculate payout (same formula as ask)
	timeToExpiry := expiryTime.Sub(*params.StartTime).Seconds() / (365.25 * 24 * 60 * 60)
	probability := BlackScholesDigitalOption(
		entrySpot,
		barrier,
		timeToExpiry,
		c.config.Volatility,
		c.config.InterestRate,
		c.config.QuantoDrift,
		params.ContractType,
	)
	payout := CalculateAskPayout(params.Stake, probability, c.config.Commission)

	// Determine bid price
	var bidPrice float64
	var exitSpot *float64
	var exitSpotTime *time.Time

	if isExpired {
		// Contract expired - check win condition
		exitSpotVal := tick.Price
		exitSpot = &exitSpotVal
		exitSpotTimeVal := tick.Timestamp
		exitSpotTime = &exitSpotTimeVal

		hasWon := c.checkWinCondition(tick.Price, barrier, params.ContractType)
		bidPrice = CalculateBidPrice(payout, probability, true, hasWon)
	} else {
		// Contract active - calculate current value
		remainingTime := expiryTime.Sub(now).Seconds() / (365.25 * 24 * 60 * 60)
		currentProbability := BlackScholesDigitalOption(
			tick.Price,
			barrier,
			remainingTime,
			c.config.Volatility,
			c.config.InterestRate,
			c.config.QuantoDrift,
			params.ContractType,
		)
		bidPrice = CalculateBidPrice(payout, currentProbability, false, false)
	}

	// Build and return quote
	return &BidQuote{
		BidPrice:        bidPrice,
		IsExpired:       isExpired,
		CurrentSpot:     tick.Price,
		CurrentSpotTime: tick.Timestamp,
		EntrySpot:       entrySpot,
		EntrySpotTime:   entrySpotTime,
		ExitSpot:        exitSpot,
		ExitSpotTime:    exitSpotTime,
		Barrier:         barrier,
		StartTime:       *params.StartTime,
		ExpiryTime:      expiryTime,
		Currency:        params.Currency,
	}, nil
}

// parseAndValidateParams validates and converts protobuf parameters to internal types
func (c *Calculator) parseAndValidateParams(params *pb.OptionParameters, requireStartTime bool) (*OptionParams, error) {
	// Validate symbol
	if params.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	// Validate contract type
	if params.ContractType == pb.ContractType_CONTRACT_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "contract_type is required")
	}

	// Validate currency
	if params.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}

	// Parse and validate stake
	stake, err := strconv.ParseFloat(params.Stake, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid stake format: %s", params.Stake)
	}
	if stake <= 0 {
		return nil, status.Error(codes.OutOfRange, "Stake must be positive")
	}
	if stake < c.limits.MinStake {
		return nil, status.Errorf(codes.FailedPrecondition, "Stake below minimum: %.2f", c.limits.MinStake)
	}

	// Parse duration
	duration, err := ParseDuration(params.Duration)
	if err != nil {
		return nil, err
	}

	// Parse barrier (optional)
	var barrier *Barrier
	if params.Barrier != nil && *params.Barrier != "" {
		barrier, err = ParseBarrier(*params.Barrier)
		if err != nil {
			return nil, err
		}
	} else {
		barrier = &Barrier{Type: BarrierTypeNone}
	}

	// Validate start_time for bid requests
	var startTime *time.Time
	if requireStartTime {
		if params.StartTime == nil {
			return nil, status.Error(codes.InvalidArgument, "start_time is required for bid requests")
		}
		st := time.Unix(*params.StartTime, 0)
		if st.After(time.Now()) {
			return nil, status.Error(codes.InvalidArgument, "start_time cannot be in the future")
		}
		startTime = &st
	}

	return &OptionParams{
		Symbol:       params.Symbol,
		ContractType: params.ContractType,
		Currency:     params.Currency,
		Stake:        stake,
		Duration:     *duration,
		Barrier:      barrier,
		StartTime:    startTime,
	}, nil
}

// calculateTimeToExpiry calculates time to expiry in years
func (c *Calculator) calculateTimeToExpiry(duration Duration, fromTime time.Time) (float64, error) {
	seconds, err := duration.ToSeconds()
	if err != nil {
		return 0, err
	}

	return TimeToExpiryInYears(seconds), nil
}

// checkWinCondition determines if a contract has won
func (c *Calculator) checkWinCondition(exitSpot, barrier float64, contractType pb.ContractType) bool {
	if contractType == pb.ContractType_CONTRACT_TYPE_CALL {
		return exitSpot > barrier
	}
	return exitSpot < barrier
}
