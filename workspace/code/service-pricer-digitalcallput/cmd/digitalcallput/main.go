// The digitalcallput program runs gRPC server that implements digitalcallput service.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/regentmarkets/service-pricer-digitalcallput/internal/app"
)

var (
	// version initialized during build with `git describe --always`
	version string
)

var rootCmd = &cobra.Command{
	Use:   "digitalcallput",
	Short: "run deriv digitalcallput service",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg := app.Config{
			GRPCAddress: viper.GetString("grpc-address"),
			HTTPAddress: viper.GetString("http-address"),
			FeedAddress: viper.GetString("feed-address"),
			Version:     version,
		}
		a, err := app.New(cfg)
		if err != nil {
			return fmt.Errorf("failed to create the application: %v", err)
		}
		ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		defer cancel()
		return a.Run(ctx)
	},
}

func init() {
	flags := rootCmd.PersistentFlags()

	flags.String("log-level", "INFO", "log level, can be DEBUG, INFO, WARN, ERROR")
	viper.MustBindEnv("log-level", "LOG_LEVEL")

	flags.Bool("log-text-format", false, "if true log messages are printed in text format instead of JSON")
	viper.MustBindEnv("log-text-format", "LOG_TEXT_FORMAT")

	flags.String("grpc-address", ":8090", "address for gRPC server to bind")
	viper.MustBindEnv("grpc-address", "GRPC_ADDRESS")

	flags.String("http-address", ":8080", "address for HTTP grpc-gateway server to bind")
	viper.MustBindEnv("http-address", "HTTP_ADDRESS")

	flags.String("feed-address", "localhost:50052", "address of service-feed")
	viper.MustBindEnv("feed-address", "FEED_ADDRESS")

	if err := viper.BindPFlags(flags); err != nil {
		panic("BindPFlags failed")
	}

}

func setDefaultLogger(level string, useText bool) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
		defer slog.Error("Invalid logging level requested, using INFO instead", slog.Any("error", err))
	}
	ho := slog.HandlerOptions{
		Level: l,
	}
	var h slog.Handler
	if useText {
		h = slog.NewTextHandler(os.Stderr, &ho)
	} else {
		h = slog.NewJSONHandler(os.Stderr, &ho)
	}
	logger := slog.New(h).With(
		slog.String("version", version),
		slog.String("application", "digitalcallput"),
	)
	slog.SetDefault(logger)
}

func main() {
	setDefaultLogger(viper.GetString("log-level"), viper.GetBool("log-text-format"))
	slog.Info("starting up")
	if err := rootCmd.Execute(); err != nil {
		slog.Error("exited with an error", slog.Any("error", err))
		os.Exit(1)
	}
}
