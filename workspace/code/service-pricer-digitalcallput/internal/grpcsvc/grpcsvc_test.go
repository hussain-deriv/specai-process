package grpcsvc

import (
	"context"
	"testing"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// Mock Calculator for testing
type mockCalculator struct{}

func (m *mockCalculator) CalculateAsk(ctx context.Context, params *pb.OptionParameters) (*pricing.AskQuote, error) {
	return &pricing.AskQuote{
		AskPrice:    100.0,
		Currency:    "USD",
		CurrentSpot: 1.08523,
		Payout:      196.0,
		Limits: &pricing.TradingLimits{
			MinStake:  1.0,
			MaxPayout: 50000.0,
		},
	}, nil
}

func (m *mockCalculator) CalculateBid(ctx context.Context, params *pb.OptionParameters) (*pricing.BidQuote, error) {
	return &pricing.BidQuote{
		BidPrice:    145.0,
		IsExpired:   false,
		CurrentSpot: 1.08600,
		EntrySpot:   1.08523,
		Barrier:     1.08573,
		Currency:    "USD",
	}, nil
}

func (m *mockCalculator) StreamAsk(ctx context.Context, params *pb.OptionParameters) (*pricing.AskSubscription, error) {
	sub := &pricing.AskSubscription{}
	// Create a closed subscription for testing
	ch := make(chan *pricing.AskQuote)
	close(ch)
	return sub, nil
}

func (m *mockCalculator) StreamBid(ctx context.Context, params *pb.OptionParameters) (*pricing.BidSubscription, error) {
	sub := &pricing.BidSubscription{}
	// Create a closed subscription for testing
	ch := make(chan *pricing.BidQuote)
	close(ch)
	return sub, nil
}

func TestService(t *testing.T) {
	calculator := &mockCalculator{}

	s := New(calculator)
	if s == nil {
		t.Error("Expected a non-nil service")
	}
}

func TestGetAsk(t *testing.T) {
	calculator := &mockCalculator{}
	s := New(calculator)

	ctx := context.Background()
	req := &pb.GetAskRequest{
		OptionParameters: &pb.OptionParameters{
			Symbol:       "EUR/USD",
			ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
			Currency:     "USD",
			Stake:        "100.00",
			Duration:     "5m",
		},
	}

	resp, err := s.GetAsk(ctx, req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if resp == nil {
		t.Error("Expected non-nil response")
	}
	if resp != nil && resp.AskPrice != "100.00" {
		t.Errorf("Expected ask price 100.00, got %s", resp.AskPrice)
	}
}
