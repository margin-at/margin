package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"margin.at/internal/analytics"
	"margin.at/internal/api"
	"margin.at/internal/db"
	"margin.at/internal/embeddings"
	"margin.at/internal/firehose"
	"margin.at/internal/logger"
	internalMiddleware "margin.at/internal/middleware"
	"margin.at/internal/oauth"
	"margin.at/internal/recommendations"
	"margin.at/internal/sync"
)

func decodedTokenEncryptionKey() ([]byte, error) {
	raw := os.Getenv("TOKEN_ENCRYPTION_KEY")
	if raw == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY: invalid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}

func main() {
	godotenv.Load("../.env", ".env")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		logger.Fatal("DATABASE_URL environment variable is required")
	}
	database, err := db.New(dsn)
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if key, kerr := decodedTokenEncryptionKey(); kerr != nil {
		logger.Fatal("config: %v", kerr)
	} else if key != nil {
		if err := database.SetEncryptionKey(key); err != nil {
			logger.Fatal("store: set encryption key: %v", err)
		}
		logger.Info("store: at-rest encryption of OAuth secrets enabled")
	}

	if err := database.Migrate(); err != nil {
		logger.Fatal("Failed to run migrations: %v", err)
	}
	database.MigrateUnifiedNotes(context.Background())

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		runCleanup := func() {
			_, err := database.TryAdvisoryLock(context.Background(), db.LockSessionCleanup, func() error {
				if err := database.DeleteExpiredOAuthSessions(context.Background()); err != nil {
					return err
				}
				return database.DeleteExpiredPendingAuthsOAuth(context.Background())
			})
			if err != nil {
				logger.Error("Failed to run cleanup of expired sessions: %v", err)
			}
		}

		runCleanup()
		for range ticker.C {
			runCleanup()
		}
	}()

	embeddingClient := embeddings.NewClient()
	recService := recommendations.NewService(database, embeddingClient)
	logger.Info("Recommendation engine initialized (embeddings enabled: %v)", embeddingClient.IsEnabled())

	syncSvc := sync.NewService(database)

	analyticsCl := analytics.New()
	defer analyticsCl.Close()

	oauthHandler, err := oauth.NewHandler(database, syncSvc, analyticsCl)
	if err != nil {
		logger.Fatal("Failed to initialize OAuth: %v", err)
	}

	ingester := firehose.NewIngester(database, syncSvc)
	firehose.RelayURL = getEnv("BLOCK_RELAY_URL", "wss://jetstream2.us-east.bsky.network/subscribe")
	logger.Info("Firehose URL: %s", firehose.RelayURL)

	backfillCtx, backfillCancel := context.WithCancel(context.Background())
	defer backfillCancel()

	if recService.IsEnabled() {
		ingester.SetOnAnnotation(recService.OnAnnotation)
		ingester.SetOnDocument(recService.OnDocument)

		if getEnv("DISABLE_BACKFILL", "") == "" {
			go func() {
				time.Sleep(5 * time.Second)
				select {
				case <-backfillCtx.Done():
					return
				default:
				}
				database.TryAdvisoryLock(backfillCtx, db.LockBackfill, func() error { //nolint:errcheck
					logger.Info("Starting recommendation backfill...")
					if err := recService.BackfillDocumentEmbeddings(200); err != nil {
						logger.Error("Document embedding backfill error: %v", err)
					}
					if backfillCtx.Err() != nil {
						return nil
					}
					annCount, err := recService.BackfillAnnotationEmbeddings(200)
					if err != nil {
						logger.Error("Annotation embedding backfill error: %v", err)
					}
					if backfillCtx.Err() != nil {
						return nil
					}
					hlCount, err := recService.BackfillHighlightEmbeddings(200)
					if err != nil {
						logger.Error("Highlight embedding backfill error: %v", err)
					}
					if backfillCtx.Err() != nil {
						return nil
					}
					profileCount, err := recService.RebuildAllProfiles()
					if err != nil {
						logger.Error("Profile rebuild error: %v", err)
					}
					logger.Info("Recommendation backfill complete (annotations: %d, highlights: %d, profiles: %d)", annCount, hlCount, profileCount)
					return nil
				})
			}()
		} else {
			logger.Info("Recommendation backfill disabled (DISABLE_BACKFILL is set)")
		}
	}

	go func() {
		_, err := database.TryAdvisoryLock(context.Background(), db.LockFirehose, func() error {
			return ingester.Start(context.Background())
		})
		if err != nil {
			logger.Error("Firehose ingester error: %v", err)
		}
	}()

	r := chi.NewRouter()

	r.Use(internalMiddleware.PrivacyLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Throttle(100))

	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			if strings.HasPrefix(origin, "chrome-extension://") ||
				strings.HasPrefix(origin, "moz-extension://") ||
				strings.HasPrefix(origin, "safari-web-extension://") {
				return true
			}
			if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
				return origin == baseURL
			}
			return false
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Session-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	tokenRefresher := api.NewTokenRefresher(database, oauthHandler.GetSigningKey())
	noteWriteSvc := api.NewNoteWriteService(database, tokenRefresher)

	handler := api.NewHandler(database, noteWriteSvc, tokenRefresher, syncSvc, recService, analyticsCl)
	handler.RegisterRoutes(r)

	r.Route("/auth", func(r chi.Router) {
		r.Use(middleware.Throttle(10))
		r.Get("/login", oauthHandler.HandleLogin)
		r.Post("/start", oauthHandler.HandleStart)
		r.Post("/signup", oauthHandler.HandleSignup)
		r.Get("/callback", oauthHandler.HandleCallback)
		r.Post("/logout", oauthHandler.HandleLogout)
		r.Get("/session", oauthHandler.HandleSession)
	})
	r.Get("/oauth-client-metadata.json", oauthHandler.HandleClientMetadata)
	r.Get("/jwks.json", oauthHandler.HandleJWKS)

	port := getEnv("PORT", "8081")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		logger.Info("Margin API server running on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Infoln("Shutting down server...")
	backfillCancel()
	ingester.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown: %v", err)
	}

	logger.Infoln("Server exited")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
