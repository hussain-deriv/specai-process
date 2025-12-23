package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// MockClient represents a mock feed client for testing
type MockClient struct {
	address string
}

// NewMockClient creates a new mock feed client
func NewMockClient(address string) (*MockClient, error) {
	if address == "" {
		return nil, fmt.Errorf("feed address is required")
	}

	return &MockClient{
		address: address,
	}, nil
}

// GetCurrentTick retrieves the current market tick for a symbol (mock implementation)
func (c *MockClient) GetCurrentTick(ctx context.Context, symbol string) (*pricing.Tick, error) {
	// Mock implementation - return fixed price
	return &pricing.Tick{
		Symbol:    symbol,
		Price:     1.08523, // Mock EUR/USD price
		Timestamp: time.Now(),
	}, nil
}

// Subscribe creates a subscription to market ticks for a symbol (mock implementation)
func (c *MockClient) Subscribe(ctx context.Context, symbol string) (<-chan *pricing.Tick, error) {
	tickChan := make(chan *pricing.Tick, 10)

	// Mock implementation - simulate tick updates
	go func() {
		defer close(tickChan)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		basePrice := 1.08523
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Generate mock price with small random variation
				variation := (float64(time.Now().UnixNano()%1000) - 500) / 100000.0
				tick := &pricing.Tick{
					Symbol:    symbol,
					Price:     basePrice + variation,
					Timestamp: time.Now(),
				}

				select {
				case tickChan <- tick:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return tickChan, nil
}

// Close closes the mock feed client connection
func (c *MockClient) Close() error {
	// Mock implementation - nothing to close
	return nil
}
