package cmd

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/spf13/cobra"

	"github.com/nuonco/nuon-ext-webhook-server/internal/server"
)

type serveOptions struct {
	addr            string
	webhookPath     string
	healthPath      string
	expectedSecret  string
	shutdownTimeout time.Duration
	readTimeout     time.Duration
	writeTimeout    time.Duration
	maxBodyBytes    int64
	prettyLogs      bool
}

func newServeCommand() *cobra.Command {
	opts := serveOptions{
		addr:            ":8787",
		webhookPath:     "/webhooks/operation/lifecycle",
		healthPath:      "/healthz",
		shutdownTimeout: 10 * time.Second,
		readTimeout:     15 * time.Second,
		writeTimeout:    15 * time.Second,
		maxBodyBytes:    1024 * 1024,
	}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run a webhook receiver that logs Nuon operation lifecycle CloudEvents payloads",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.addr, "addr", opts.addr, "Address to bind the HTTP server to")
	cmd.Flags().StringVar(&opts.webhookPath, "path", opts.webhookPath, "Webhook endpoint path")
	cmd.Flags().StringVar(&opts.healthPath, "health-path", opts.healthPath, "Health check endpoint path")
	cmd.Flags().StringVar(&opts.expectedSecret, "secret", "", "Shared secret for HMAC-SHA256 signature validation (X-Nuon-Signature)")
	cmd.Flags().DurationVar(&opts.shutdownTimeout, "shutdown-timeout", opts.shutdownTimeout, "Graceful shutdown timeout")
	cmd.Flags().DurationVar(&opts.readTimeout, "read-timeout", opts.readTimeout, "HTTP read timeout")
	cmd.Flags().DurationVar(&opts.writeTimeout, "write-timeout", opts.writeTimeout, "HTTP write timeout")
	cmd.Flags().Int64Var(&opts.maxBodyBytes, "max-body-bytes", opts.maxBodyBytes, "Maximum accepted webhook payload size in bytes")
	cmd.Flags().BoolVar(&opts.prettyLogs, "pretty-logs", false, "Use human-readable development logs")

	return cmd
}

func runServe(parentCtx context.Context, opts serveOptions) error {
	if opts.expectedSecret == "" {
		opts.expectedSecret = strings.TrimSpace(os.Getenv("NUON_WEBHOOK_SECRET"))
	}

	logger, err := buildLogger(opts.prettyLogs)
	if err != nil {
		return err
	}
	defer logger.Sync()

	webhookPath := normalizePath(opts.webhookPath)
	healthPath := normalizePath(opts.healthPath)

	srv := server.New(server.Config{
		Logger:         logger,
		WebhookPath:    webhookPath,
		HealthPath:     healthPath,
		ExpectedSecret: opts.expectedSecret,
		MaxBodyBytes:   opts.maxBodyBytes,
	})

	httpServer := &http.Server{
		Addr:              opts.addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       opts.readTimeout,
		WriteTimeout:      opts.writeTimeout,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	logger.Info("webhook server started",
		zap.String("addr", opts.addr),
		zap.String("webhook_path", webhookPath),
		zap.String("health_path", healthPath),
		zap.Bool("secret_required", opts.expectedSecret != ""),
		zap.Int64("max_body_bytes", opts.maxBodyBytes))

	signalCtx, stopSignals := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	select {
	case <-signalCtx.Done():
		logger.Info("received shutdown signal")
	case err := <-errCh:
		if err != nil {
			logger.Error("webhook server exited with error", zap.Error(err))
			return err
		}

		logger.Info("webhook server stopped")
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), opts.shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
		return err
	}

	if err := <-errCh; err != nil {
		logger.Error("webhook server exited with error", zap.Error(err))
		return err
	}

	logger.Info("webhook server stopped")

	return nil
}

func buildLogger(pretty bool) (*zap.Logger, error) {
	if pretty {
		cfg := zap.NewDevelopmentConfig()
		cfg.DisableCaller = true
		return cfg.Build()
	}

	cfg := zap.NewProductionConfig()
	return cfg.Build()
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}

	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}

	return "/" + trimmed
}
