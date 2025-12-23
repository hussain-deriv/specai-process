package feed

import (
	"context"
	"sync"
	"time"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// Subscriber manages subscriptions to market data feeds
type Subscriber struct {
	client        *Client
	subscriptions map[string]*subscription
	mu            sync.RWMutex
}

// subscription represents an active subscription to a symbol
type subscription struct {
	symbol   string
	tickChan <-chan *pricing.Tick
	cancel   context.CancelFunc
	lastTick *pricing.Tick
	mu       sync.RWMutex
}

// NewSubscriber creates a new feed subscriber
func NewSubscriber(client *Client) *Subscriber {
	return &Subscriber{
		client:        client,
		subscriptions: make(map[string]*subscription),
	}
}

// GetCurrentTick returns the most recent tick for a symbol
// If no subscription exists, it fetches a fresh tick from the feed
func (s *Subscriber) GetCurrentTick(ctx context.Context, symbol string) (*pricing.Tick, error) {
	s.mu.RLock()
	sub, exists := s.subscriptions[symbol]
	s.mu.RUnlock()

	if exists {
		sub.mu.RLock()
		defer sub.mu.RUnlock()
		if sub.lastTick != nil {
			return sub.lastTick, nil
		}
	}

	// No cached tick, fetch from feed
	return s.client.GetCurrentTick(ctx, symbol)
}

// Subscribe creates or returns an existing subscription for a symbol
func (s *Subscriber) Subscribe(ctx context.Context, symbol string) (<-chan *pricing.Tick, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if subscription already exists
	if sub, exists := s.subscriptions[symbol]; exists {
		// Return a new channel that mirrors the existing subscription
		return s.mirrorSubscription(sub), nil
	}

	// Create new subscription
	subCtx, cancel := context.WithCancel(context.Background())
	tickChan, err := s.client.Subscribe(subCtx, symbol)
	if err != nil {
		cancel()
		return nil, err
	}

	sub := &subscription{
		symbol:   symbol,
		tickChan: tickChan,
		cancel:   cancel,
	}

	s.subscriptions[symbol] = sub

	// Start goroutine to track the last tick
	go s.trackLastTick(sub)

	return s.mirrorSubscription(sub), nil
}

// mirrorSubscription creates a new channel that mirrors an existing subscription
func (s *Subscriber) mirrorSubscription(sub *subscription) <-chan *pricing.Tick {
	outChan := make(chan *pricing.Tick, 10)

	go func() {
		defer close(outChan)
		for tick := range sub.tickChan {
			select {
			case outChan <- tick:
			default:
				// Channel full, skip this tick
			}
		}
	}()

	return outChan
}

// trackLastTick keeps track of the most recent tick for a subscription
func (s *Subscriber) trackLastTick(sub *subscription) {
	for tick := range sub.tickChan {
		sub.mu.Lock()
		sub.lastTick = tick
		sub.mu.Unlock()
	}
}

// Unsubscribe removes a subscription for a symbol
func (s *Subscriber) Unsubscribe(symbol string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sub, exists := s.subscriptions[symbol]; exists {
		sub.cancel()
		delete(s.subscriptions, symbol)
	}
}

// Close closes all subscriptions
func (s *Subscriber) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sub := range s.subscriptions {
		sub.cancel()
	}
	s.subscriptions = make(map[string]*subscription)
}

// StreamWithFallback streams ticks with a time-based fallback
// If no ticks arrive within the fallback duration, it fetches a fresh tick
func (s *Subscriber) StreamWithFallback(ctx context.Context, symbol string, fallbackDuration time.Duration) (<-chan *pricing.Tick, error) {
	tickChan, err := s.Subscribe(ctx, symbol)
	if err != nil {
		return nil, err
	}

	outChan := make(chan *pricing.Tick, 10)

	go func() {
		defer close(outChan)

		ticker := time.NewTicker(fallbackDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case tick, ok := <-tickChan:
				if !ok {
					return
				}

				// Reset the fallback ticker
				ticker.Reset(fallbackDuration)

				select {
				case outChan <- tick:
				case <-ctx.Done():
					return
				}

			case <-ticker.C:
				// Fallback: fetch current tick
				tick, err := s.GetCurrentTick(ctx, symbol)
				if err != nil {
					continue // Skip this fallback tick on error
				}

				select {
				case outChan <- tick:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outChan, nil
}
