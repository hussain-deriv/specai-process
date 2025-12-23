package feed

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	feedapi "github.com/regentmarkets/service-feed/api"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// Client represents a connection to the service-feed
type Client struct {
	conn   *grpc.ClientConn
	client feedapi.TickServiceClient
}

// NewClient creates a new feed client connected to service-feed
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service-feed at %s: %w", addr, err)
	}
	return &Client{
		conn:   conn,
		client: feedapi.NewTickServiceClient(conn),
	}, nil
}

// Subscribe creates a subscription to market ticks for a symbol
func (c *Client) Subscribe(ctx context.Context, symbol string) (<-chan *pricing.Tick, error) {
	stream, err := c.client.StreamTicks(ctx, &feedapi.StreamTicksRequest{
		Symbol: symbol,
		Time:   timestamppb.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start tick stream: %w", err)
	}

	ticks := make(chan *pricing.Tick, 100)
	go func() {
		defer close(ticks)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := stream.Recv()
				if err != nil {
					return
				}
				for _, tick := range resp.Ticks {
					select {
					case ticks <- &pricing.Tick{
						Symbol:    tick.Symbol,
						Price:     parseQuote(tick.Quote),
						Timestamp: tick.Time.AsTime(),
					}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ticks, nil
}

// GetCurrentTick retrieves the current market tick for a symbol
func (c *Client) GetCurrentTick(ctx context.Context, symbol string) (*pricing.Tick, error) {
	stream, err := c.client.StreamTicks(ctx, &feedapi.StreamTicksRequest{
		Symbol: symbol,
		Time:   timestamppb.Now(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	if len(resp.Ticks) == 0 {
		return nil, fmt.Errorf("no ticks received for symbol %s", symbol)
	}
	tick := resp.Ticks[0]
	return &pricing.Tick{
		Symbol:    tick.Symbol,
		Price:     parseQuote(tick.Quote),
		Timestamp: tick.Time.AsTime(),
	}, nil
}

// Close closes the feed client connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// parseQuote converts the string quote to float64
// In the service-feed, Quote is a string representation of the price
func parseQuote(quote string) float64 {
	var price float64
	fmt.Sscanf(quote, "%f", &price)
	return price
}
