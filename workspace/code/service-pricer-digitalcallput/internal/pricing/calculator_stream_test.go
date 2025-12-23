package pricing

import (
	"context"
	"sync"
	"testing"
	"time"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
)

// mockFeedSubscriber for testing
type mockFeedSubscriber struct {
	ticks     chan *Tick
	lastTick  *Tick
	tickMutex sync.RWMutex
}

func (m *mockFeedSubscriber) GetCurrentTick(ctx context.Context, symbol string) (*Tick, error) {
	m.tickMutex.RLock()
	defer m.tickMutex.RUnlock()

	if m.lastTick != nil {
		return m.lastTick, nil
	}

	return &Tick{
		Symbol:    symbol,
		Price:     1.08523,
		Timestamp: time.Now(),
	}, nil
}

func (m *mockFeedSubscriber) Subscribe(ctx context.Context, symbol string) (<-chan *Tick, error) {
	return m.mirrorTicks(), nil
}

func (m *mockFeedSubscriber) StreamWithFallback(ctx context.Context, symbol string, fallbackDuration time.Duration) (<-chan *Tick, error) {
	return m.mirrorTicks(), nil
}

func (m *mockFeedSubscriber) mirrorTicks() <-chan *Tick {
	out := make(chan *Tick, 10)
	go func() {
		defer close(out)
		for tick := range m.ticks {
			m.tickMutex.Lock()
			m.lastTick = tick
			m.tickMutex.Unlock()
			out <- tick
		}
	}()
	return out
}

func TestStreamAsk(t *testing.T) {
	// Create mock feed
	tickChan := make(chan *Tick, 10)
	mockFeed := &mockFeedSubscriber{ticks: tickChan}

	// Create calculator
	config := &PricingConfig{
		Volatility:   0.10,
		Commission:   0.02,
		InterestRate: 0.0,
		QuantoDrift:  0.0,
	}
	limits := &TradingLimits{
		MinStake:  1.0,
		MaxPayout: 50000.0,
	}
	calc := NewCalculator(config, limits, mockFeed)

	// Create test parameters
	params := &pb.OptionParameters{
		Symbol:       "EUR/USD",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Stake:        "100.00",
		Duration:     "5m",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start streaming
	sub, err := calc.StreamAsk(ctx, params)
	if err != nil {
		t.Fatalf("StreamAsk failed: %v", err)
	}
	defer sub.Close()

	// Expect initial quote
	select {
	case quote := <-sub.Quotes():
		if quote == nil {
			t.Fatal("Expected initial quote, got nil")
		}
		if quote.AskPrice != 100.0 {
			t.Errorf("Expected AskPrice=100.0, got %v", quote.AskPrice)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for initial quote")
	}

	// Send a tick
	tickChan <- &Tick{
		Symbol:    "EUR/USD",
		Price:     1.08600,
		Timestamp: time.Now(),
	}

	// Expect updated quote
	select {
	case quote := <-sub.Quotes():
		if quote == nil {
			t.Fatal("Expected updated quote, got nil")
		}
		if quote.CurrentSpot != 1.08600 {
			t.Errorf("Expected CurrentSpot=1.08600, got %v", quote.CurrentSpot)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for updated quote")
	}

	// Close tick channel
	close(tickChan)

	// Wait for subscription to close
	select {
	case _, ok := <-sub.Quotes():
		if ok {
			t.Error("Expected quotes channel to be closed")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for quotes channel to close")
	}

	// Check for errors
	if sub.Err() != nil {
		t.Errorf("Unexpected error: %v", sub.Err())
	}
}

func TestStreamBid_TimeBasedExpiry(t *testing.T) {
	// Create mock feed
	tickChan := make(chan *Tick, 10)
	mockFeed := &mockFeedSubscriber{ticks: tickChan}

	// Create calculator
	config := &PricingConfig{
		Volatility:   0.10,
		Commission:   0.02,
		InterestRate: 0.0,
		QuantoDrift:  0.0,
	}
	limits := &TradingLimits{
		MinStake:  1.0,
		MaxPayout: 50000.0,
	}
	calc := NewCalculator(config, limits, mockFeed)

	// Create test parameters with past start time and short duration
	startTime := time.Now().Add(-10 * time.Second).Unix()
	params := &pb.OptionParameters{
		Symbol:       "EUR/USD",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Stake:        "100.00",
		Duration:     "5s", // 5 seconds duration, should already be expired
		StartTime:    &startTime,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start streaming
	sub, err := calc.StreamBid(ctx, params)
	if err != nil {
		t.Fatalf("StreamBid failed: %v", err)
	}
	defer sub.Close()

	// Expect initial quote
	select {
	case quote := <-sub.Quotes():
		if quote == nil {
			t.Fatal("Expected initial quote, got nil")
		}
		// Contract should be expired
		if !quote.IsExpired {
			t.Error("Expected contract to be expired")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for initial quote")
	}

	// Since contract is expired, stream should close immediately
	select {
	case _, ok := <-sub.Quotes():
		if ok {
			t.Error("Expected quotes channel to be closed after expiry")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for quotes channel to close after expiry")
	}

	// Check for errors
	if sub.Err() != nil {
		t.Errorf("Unexpected error: %v", sub.Err())
	}

	close(tickChan)
}

func TestStreamBid_ActiveContract(t *testing.T) {
	// Create mock feed
	tickChan := make(chan *Tick, 10)
	mockFeed := &mockFeedSubscriber{ticks: tickChan}

	// Create calculator
	config := &PricingConfig{
		Volatility:   0.10,
		Commission:   0.02,
		InterestRate: 0.0,
		QuantoDrift:  0.0,
	}
	limits := &TradingLimits{
		MinStake:  1.0,
		MaxPayout: 50000.0,
	}
	calc := NewCalculator(config, limits, mockFeed)

	// Create test parameters with recent start time and long duration
	startTime := time.Now().Add(-1 * time.Second).Unix()
	params := &pb.OptionParameters{
		Symbol:       "EUR/USD",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Stake:        "100.00",
		Duration:     "1h", // 1 hour duration, should still be active
		StartTime:    &startTime,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start streaming
	sub, err := calc.StreamBid(ctx, params)
	if err != nil {
		t.Fatalf("StreamBid failed: %v", err)
	}
	defer sub.Close()

	// Expect initial quote
	select {
	case quote := <-sub.Quotes():
		if quote == nil {
			t.Fatal("Expected initial quote, got nil")
		}
		// Contract should NOT be expired
		if quote.IsExpired {
			t.Error("Expected contract to be active")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for initial quote")
	}

	// Send a tick
	tickChan <- &Tick{
		Symbol:    "EUR/USD",
		Price:     1.08700,
		Timestamp: time.Now(),
	}

	// Expect updated quote
	select {
	case quote := <-sub.Quotes():
		if quote == nil {
			t.Fatal("Expected updated quote, got nil")
		}
		if quote.CurrentSpot != 1.08700 {
			t.Errorf("Expected CurrentSpot=1.08700, got %v", quote.CurrentSpot)
		}
		if quote.IsExpired {
			t.Error("Expected contract to still be active")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for updated quote")
	}

	// Close tick channel
	close(tickChan)

	// Wait for subscription to close
	select {
	case _, ok := <-sub.Quotes():
		if ok {
			t.Error("Expected quotes channel to be closed")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for quotes channel to close")
	}

	// Check for errors
	if sub.Err() != nil {
		t.Errorf("Unexpected error: %v", sub.Err())
	}
}

func TestStreamAsk_ContextCancellation(t *testing.T) {
	// Create mock feed
	tickChan := make(chan *Tick, 10)
	mockFeed := &mockFeedSubscriber{ticks: tickChan}

	// Create calculator
	config := &PricingConfig{
		Volatility:   0.10,
		Commission:   0.02,
		InterestRate: 0.0,
		QuantoDrift:  0.0,
	}
	limits := &TradingLimits{
		MinStake:  1.0,
		MaxPayout: 50000.0,
	}
	calc := NewCalculator(config, limits, mockFeed)

	// Create test parameters
	params := &pb.OptionParameters{
		Symbol:       "EUR/USD",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Stake:        "100.00",
		Duration:     "5m",
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start streaming
	sub, err := calc.StreamAsk(ctx, params)
	if err != nil {
		t.Fatalf("StreamAsk failed: %v", err)
	}
	defer sub.Close()

	// Expect initial quote
	select {
	case quote := <-sub.Quotes():
		if quote == nil {
			t.Fatal("Expected initial quote, got nil")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for initial quote")
	}

	// Cancel context
	cancel()

	// Wait for subscription to close
	select {
	case _, ok := <-sub.Quotes():
		if ok {
			t.Error("Expected quotes channel to be closed after context cancellation")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for quotes channel to close after context cancellation")
	}

	// Check for context cancellation error
	if sub.Err() != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", sub.Err())
	}

	close(tickChan)
}

func TestStreamAsk_InvalidParameters(t *testing.T) {
	// Create mock feed
	tickChan := make(chan *Tick, 10)
	mockFeed := &mockFeedSubscriber{ticks: tickChan}

	// Create calculator
	config := &PricingConfig{
		Volatility:   0.10,
		Commission:   0.02,
		InterestRate: 0.0,
		QuantoDrift:  0.0,
	}
	limits := &TradingLimits{
		MinStake:  1.0,
		MaxPayout: 50000.0,
	}
	calc := NewCalculator(config, limits, mockFeed)

	// Create test parameters with invalid stake
	params := &pb.OptionParameters{
		Symbol:       "EUR/USD",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Stake:        "0.50", // Below minimum stake
		Duration:     "5m",
	}

	ctx := context.Background()

	// Start streaming should fail
	_, err := calc.StreamAsk(ctx, params)
	if err == nil {
		t.Fatal("Expected StreamAsk to fail with invalid parameters")
	}

	close(tickChan)
}

func TestStreamBid_MissingStartTime(t *testing.T) {
	// Create mock feed
	tickChan := make(chan *Tick, 10)
	mockFeed := &mockFeedSubscriber{ticks: tickChan}

	// Create calculator
	config := &PricingConfig{
		Volatility:   0.10,
		Commission:   0.02,
		InterestRate: 0.0,
		QuantoDrift:  0.0,
	}
	limits := &TradingLimits{
		MinStake:  1.0,
		MaxPayout: 50000.0,
	}
	calc := NewCalculator(config, limits, mockFeed)

	// Create test parameters without start_time
	params := &pb.OptionParameters{
		Symbol:       "EUR/USD",
		ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
		Currency:     "USD",
		Stake:        "100.00",
		Duration:     "5m",
		// StartTime is missing
	}

	ctx := context.Background()

	// Start streaming should fail
	_, err := calc.StreamBid(ctx, params)
	if err == nil {
		t.Fatal("Expected StreamBid to fail without start_time")
	}

	close(tickChan)
}
