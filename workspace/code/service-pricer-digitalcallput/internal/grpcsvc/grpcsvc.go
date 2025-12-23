// Package grpcsvc implements gRPC service handlers for digital call/put pricing
package grpcsvc

import (
	"context"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// PricingCalculator defines the interface for pricing calculations (Rule 5.5: Interface at Consumer)
type PricingCalculator interface {
	CalculateAsk(ctx context.Context, params *pb.OptionParameters) (*pricing.AskQuote, error)
	CalculateBid(ctx context.Context, params *pb.OptionParameters) (*pricing.BidQuote, error)
	StreamAsk(ctx context.Context, params *pb.OptionParameters) (*pricing.AskSubscription, error)
	StreamBid(ctx context.Context, params *pb.OptionParameters) (*pricing.BidSubscription, error)
}

// Service implements gRPC service PricingService
type Service struct {
	pb.UnimplementedPricingServiceServer
	calculator PricingCalculator
}

// New returns a new instance of the service
func New(calculator PricingCalculator) *Service {
	return &Service{
		calculator: calculator,
	}
}

// GetAsk calculates a single ask price for a digital option proposal
// Rule 5.1: Handler ONLY delegates to pricing module
func (s *Service) GetAsk(ctx context.Context, req *pb.GetAskRequest) (*pb.GetAskResponse, error) {
	// Delegate ALL work to the pricing module
	quote, err := s.calculator.CalculateAsk(ctx, req.OptionParameters)
	if err != nil {
		return nil, err // Return error as-is (pricing returns proper gRPC errors)
	}

	return quote.ToProto(), nil
}

// StreamAsk streams continuous ask price updates for a digital option proposal
// Rule 5.1: Handler ONLY sends quotes from pricing subscription to stream
func (s *Service) StreamAsk(req *pb.GetAskRequest, stream pb.PricingService_StreamAskServer) error {
	// Delegate subscription management to pricing module
	sub, err := s.calculator.StreamAsk(stream.Context(), req.OptionParameters)
	if err != nil {
		return err
	}
	defer sub.Close()

	// Simply iterate quotes and send to stream
	for quote := range sub.Quotes() {
		if err := stream.Send(quote.ToProto()); err != nil {
			return err
		}
	}

	return sub.Err()
}

// GetBid calculates a single bid price for an active contract
// Rule 5.1: Handler ONLY delegates to pricing module
func (s *Service) GetBid(ctx context.Context, req *pb.GetBidRequest) (*pb.GetBidResponse, error) {
	// Delegate ALL work to the pricing module
	quote, err := s.calculator.CalculateBid(ctx, req.OptionParameters)
	if err != nil {
		return nil, err // Return error as-is (pricing returns proper gRPC errors)
	}

	return quote.ToProto(), nil
}

// StreamBid streams continuous bid price updates for an active contract
// Rule 5.1: Handler ONLY sends quotes from pricing subscription to stream
func (s *Service) StreamBid(req *pb.GetBidRequest, stream pb.PricingService_StreamBidServer) error {
	// Delegate subscription management to pricing module
	sub, err := s.calculator.StreamBid(stream.Context(), req.OptionParameters)
	if err != nil {
		return err
	}
	defer sub.Close()

	// Simply iterate quotes and send to stream
	for quote := range sub.Quotes() {
		if err := stream.Send(quote.ToProto()); err != nil {
			return err
		}
	}

	return sub.Err()
}
