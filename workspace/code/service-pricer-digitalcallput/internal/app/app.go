// Package app initializes all the app dependencies and starts the app
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/feed"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/grpcsvc"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricing"
)

// Config contains application configuration
type Config struct {
	// address to which gRPC server binds
	GRPCAddress string
	// address to which HTTP grpc-gateway binds
	HTTPAddress string
	// version of the app
	Version string
	// FeedAddress is the address of the service-feed
	FeedAddress string
}

// App is the application instance
type App struct {
	grpcListener net.Listener
	ready        chan struct{}
	cfg          Config
	wg           sync.WaitGroup
}

// New creates a new application
func New(cfg Config) (*App, error) {
	app := &App{
		cfg:   cfg,
		ready: make(chan struct{}),
	}
	return app, nil
}

// Run starts the application
func (app *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// initialize all the components and run the application

	if err := app.startGRPCServer(ctx, cancel); err != nil {
		return fmt.Errorf("couldn't start gRPC server: %v", err)
	}
	if err := app.startGRPCGateway(ctx, cancel); err != nil {
		return fmt.Errorf("couldn't start gRPC gateway: %v", err)
	}

	close(app.ready)
	app.wg.Wait()

	return nil
}

// Ready returns a channel that is closed when the app initialization is done
func (app *App) Ready() <-chan struct{} {
	return app.ready
}

func (app *App) startGRPCServer(ctx context.Context, cancel func()) error {
	l, err := net.Listen("tcp", app.cfg.GRPCAddress)
	if err != nil {
		return fmt.Errorf("couldn't bind to %s: %v", app.cfg.GRPCAddress, err)
	}

	// Initialize feed client
	feedAddress := app.cfg.FeedAddress
	if feedAddress == "" {
		feedAddress = "localhost:50052" // Default feed address
	}
	feedClient, err := feed.NewClient(feedAddress)
	if err != nil {
		return fmt.Errorf("couldn't create feed client: %v", err)
	}

	// Initialize feed subscriber
	subscriber := feed.NewSubscriber(feedClient)

	// Initialize pricing configuration (from architecture spec)
	pricingConfig := &pricing.PricingConfig{
		Volatility:   0.10, // 10% fixed volatility
		Commission:   0.02, // 2% commission (hidden)
		InterestRate: 0.0,  // 0% interest rate
		QuantoDrift:  0.0,  // 0 quanto drift
	}

	// Initialize trading limits (from architecture spec)
	tradingLimits := &pricing.TradingLimits{
		MinStake:  1.00,     // $1 minimum stake
		MaxPayout: 50000.00, // $50,000 maximum payout
	}

	// Initialize calculator (pass subscriber as FeedSubscriber interface)
	calculator := pricing.NewCalculator(pricingConfig, tradingLimits, subscriber)

	// Create gRPC service
	service := grpcsvc.New(calculator)

	s := grpc.NewServer()
	digitalcallput.RegisterPricingServiceServer(s, service)
	reflection.Register(s)

	app.grpcListener = l
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer cancel()
		defer subscriber.Close()
		defer feedClient.Close()

		err := s.Serve(l)
		if err != nil {
			slog.Error("gRPC server exited", slog.Any("error", err))
			cancel()
		} else {
			slog.Info("gRPC server exited")
		}
	}()

	go func() {
		<-ctx.Done()
		s.GracefulStop()
	}()

	return nil
}

func (app *App) startGRPCGateway(ctx context.Context, cancel func()) error {
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
		}),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := digitalcallput.RegisterPricingServiceHandlerFromEndpoint(ctx, mux, app.grpcListener.Addr().String(), opts); err != nil {
		return fmt.Errorf("couldn't register grpc-gateway: %v", err)
	}
	gw := &http.Server{
		Addr:              app.cfg.HTTPAddress,
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           mux,
	}
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer cancel()
		err := gw.ListenAndServe()
		if err == http.ErrServerClosed {
			slog.Info("gateway server exited")
		} else {
			slog.Error("gateway server exited with an error", slog.Any("error", err))
		}
	}()
	go func() {
		<-ctx.Done()
		_ = gw.Shutdown(context.Background())
	}()

	return nil
}
