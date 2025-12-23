package app

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
)

func TestApp(t *testing.T) {
	app, err := New(Config{
		GRPCAddress: ":0",
		HTTPAddress: ":0",
		Version:     "test",
		FeedAddress: "localhost:50052",
	})
	if err != nil {
		t.Fatalf("couldn't create app: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		err := app.Run(ctx)
		if err != nil {
			t.Errorf("app.Run failed: %v", err)
		}
		close(done)
	}()

	select {
	case <-app.Ready():
		// all good
	case <-ctx.Done():
		t.Fatalf("app didn't start in time: %v", ctx.Err())
	}

	addr := app.grpcListener.Addr()
	conn, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("couldn't create grpc client: %v", err)
	}
	defer conn.Close()

	cli := pb.NewPricingServiceClient(conn)

	// Test GetAsk
	req := &pb.GetAskRequest{
		OptionParameters: &pb.OptionParameters{
			Symbol:       "EUR/USD",
			ContractType: pb.ContractType_CONTRACT_TYPE_CALL,
			Currency:     "USD",
			Stake:        "100.00",
			Duration:     "5m",
		},
	}

	resp, err := cli.GetAsk(ctx, req)
	if err != nil {
		t.Fatalf("couldn't call GetAsk: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.AskPrice == "" {
		t.Fatal("expected non-empty ask_price")
	}
	if resp.Currency != "USD" {
		t.Fatalf("unexpected currency: %s", resp.Currency)
	}

	cancel()
	<-done
}
