package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	nethttp "net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	azhttp "github.com/local/picobot/internal/azimutplus/http"
	"github.com/local/picobot/internal/azimutplus/service"
	"github.com/local/picobot/internal/azimutplus/store"
	"github.com/local/picobot/internal/azimutplus/whatsapp"
	"github.com/spf13/cobra"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func newAzimutPlusAPICmd() *cobra.Command {
	var (
		addr                   string
		databaseURL            string
		businessID             int64
		timezone               string
		seed                   bool
		apiKey                 string
		webhookSecret          string
		adminToken             string
		allowedOrigins         string
		rateLimitPerMinute     int
		rateLimitWindowSeconds int
	)

	cmd := &cobra.Command{
		Use:   "azimutplus-api",
		Short: "Start AzimutPlus HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if databaseURL == "" {
				databaseURL = os.Getenv("AZIMUTPLUS_DATABASE_URL")
			}
			if apiKey == "" {
				apiKey = os.Getenv("AZIMUTPLUS_API_KEY")
			}
			if apiKey == "" {
				apiKey = os.Getenv("AZIMUTPLUS_API_KEYS")
			}
			if webhookSecret == "" {
				webhookSecret = os.Getenv("AZIMUTPLUS_WEBHOOK_SECRET")
			}
			if webhookSecret == "" {
				webhookSecret = os.Getenv("AZIMUTPLUS_WEBHOOK_SECRETS")
			}
			if adminToken == "" {
				adminToken = os.Getenv("AZIMUTPLUS_ADMIN_TOKEN")
			}
			if allowedOrigins == "" {
				allowedOrigins = os.Getenv("AZIMUTPLUS_ALLOWED_ORIGINS")
			}

			if rateLimitPerMinute == 0 {
				if v := strings.TrimSpace(os.Getenv("AZIMUTPLUS_RATE_LIMIT_PER_MINUTE")); v != "" {
					if parsed, err := strconv.Atoi(v); err == nil {
						rateLimitPerMinute = parsed
					}
				}
			}
			if rateLimitWindowSeconds == 0 {
				if v := strings.TrimSpace(os.Getenv("AZIMUTPLUS_RATE_LIMIT_WINDOW_SECONDS")); v != "" {
					if parsed, err := strconv.Atoi(v); err == nil {
						rateLimitWindowSeconds = parsed
					}
				}
			}

			if databaseURL == "" {
				return fmt.Errorf("missing database URL: use --database-url or AZIMUTPLUS_DATABASE_URL")
			}

			db, err := sql.Open("pgx", databaseURL)
			if err != nil {
				return fmt.Errorf("open postgres: %w", err)
			}
			defer db.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := db.PingContext(ctx); err != nil {
				return fmt.Errorf("ping postgres: %w", err)
			}

			repo := store.New(db)
			if err := repo.InitSchema(ctx); err != nil {
				return err
			}
			if err := repo.EnsureBusiness(ctx, businessID, "AzimutPlus Demo Business"); err != nil {
				return err
			}
			if seed {
				if err := repo.SeedCustomers(ctx, businessID); err != nil {
					return err
				}
			}

			loc, err := time.LoadLocation(timezone)
			if err != nil {
				return fmt.Errorf("invalid timezone %q: %w", timezone, err)
			}

			waSender, err := whatsapp.NewSenderFromEnv()
			if err != nil {
				return fmt.Errorf("configure whatsapp sender: %w", err)
			}

			svc := service.New(repo, waSender, businessID, loc)
			server := azhttp.NewServer(svc, azhttp.Options{
				APIKeysCSV:             apiKey,
				WebhookSecretsCSV:      webhookSecret,
				AdminToken:             adminToken,
				AllowedOriginsCSV:      allowedOrigins,
				RateLimitPerMinute:     rateLimitPerMinute,
				RateLimitWindowSeconds: rateLimitWindowSeconds,
			})
			httpServer := &nethttp.Server{
				Addr:         addr,
				Handler:      server.Router(),
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  30 * time.Second,
			}

			go func() {
				log.Printf("AzimutPlus API listening on %s", addr)
				if err := httpServer.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
					log.Printf("api server error: %v", err)
				}
			}()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			return httpServer.Shutdown(shutdownCtx)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	cmd.Flags().StringVar(&databaseURL, "database-url", "", "PostgreSQL DSN")
	cmd.Flags().Int64Var(&businessID, "business-id", 1, "Fixed business id for MVP")
	cmd.Flags().StringVar(&timezone, "timezone", "Europe/Paris", "Business timezone")
	cmd.Flags().BoolVar(&seed, "seed", true, "Seed demo customers on startup")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key(s) required in X-API-Key header, comma-separated rotation supported (AZIMUTPLUS_API_KEY(S))")
	cmd.Flags().StringVar(&webhookSecret, "webhook-secret", "", "HMAC secret(s) for X-WhatsApp-Signature, comma-separated rotation supported (AZIMUTPLUS_WEBHOOK_SECRET(S))")
	cmd.Flags().StringVar(&adminToken, "admin-token", "", "Admin token for /api/v1/admin/security/rotate (or AZIMUTPLUS_ADMIN_TOKEN)")
	cmd.Flags().StringVar(&allowedOrigins, "allowed-origins", "", "Allowed CORS origins CSV (or AZIMUTPLUS_ALLOWED_ORIGINS), empty allows all")
	cmd.Flags().IntVar(&rateLimitPerMinute, "rate-limit-per-minute", 0, "Per-IP limit for API requests per minute (or AZIMUTPLUS_RATE_LIMIT_PER_MINUTE)")
	cmd.Flags().IntVar(&rateLimitWindowSeconds, "rate-limit-window-seconds", 0, "Rate limit fixed window in seconds (or AZIMUTPLUS_RATE_LIMIT_WINDOW_SECONDS)")

	return cmd
}
