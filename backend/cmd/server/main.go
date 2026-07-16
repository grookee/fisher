package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/fisher/backend/internal/config"
	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/debug"
	"github.com/fisher/backend/internal/handlers"
	"github.com/fisher/backend/internal/middleware"
	applemusic "github.com/fisher/backend/internal/services/applemusic"
	"github.com/fisher/backend/internal/services/collaboration"
	"github.com/fisher/backend/internal/services/lastfm"
	spotifysvc "github.com/fisher/backend/internal/services/spotify"
)

func main() {
	config.LoadEnv()
	debug.Init()

	if err := database.Connect(); err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	if err := database.Migrate(ctx); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	spotifysvc.Init()
	lastfm.Init()
	apple := applemusic.New()
	apple.Init()

	_ = collaboration.NewCollaborationService()

	authHandler := handlers.NewAuthHandler()
	oauthHandler := handlers.NewOAuthHandler()
	playlistHandler := handlers.NewPlaylistHandler()
	friendHandler := handlers.NewFriendHandler()
	tasteHandler := handlers.NewTasteHandler()
	settingsHandler := handlers.NewSettingsHandler()
	discoverHandler := handlers.NewDiscoverHandler()
	exploreHandler := handlers.NewExploreHandler()
	previewHandler := handlers.NewPreviewHandler()

	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate)
			r.Get("/me", authHandler.Me)
			r.Get("/accounts", oauthHandler.ListAccounts)
			r.Delete("/accounts/{service}", oauthHandler.UnlinkAccount)
		})
	})

	r.Route("/api/auth/spotify", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate)
			r.Get("/authorize", oauthHandler.SpotifyAuthorize)
		})
		r.Get("/callback", oauthHandler.SpotifyCallback)
	})

	r.Route("/api/playlists", func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Get("/", playlistHandler.List)
		r.Post("/", playlistHandler.Create)
		r.Get("/search", playlistHandler.SearchTracks)
		r.Get("/{id}", playlistHandler.Get)
		r.Put("/{id}", playlistHandler.Update)
		r.Delete("/{id}", playlistHandler.Delete)
		r.Post("/{id}/collaborators", playlistHandler.AddCollaborator)
		r.Delete("/{id}/collaborators/{userId}", playlistHandler.RemoveCollaborator)
	})

	r.Route("/api/friends", func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Get("/", friendHandler.List)
		r.Get("/requests", friendHandler.Requests)
		r.Post("/request", friendHandler.SendRequest)
		r.Post("/accept", friendHandler.AcceptRequest)
		r.Delete("/", friendHandler.RemoveFriend)
		r.Get("/search", friendHandler.SearchUsers)
	})

	r.Route("/api/tastes", func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Get("/genres", tasteHandler.Genres)
		r.Get("/profile", tasteHandler.GetProfile)
		r.Get("/profile/{userId}", tasteHandler.GetProfile)
		r.Post("/genres", tasteHandler.UpdateGenres)
		r.Post("/analyze", tasteHandler.Analyze)
		r.Post("/share", tasteHandler.Share)
		r.Post("/unshare", tasteHandler.Unshare)
		r.Get("/friends", tasteHandler.FriendsTastes)
	})

	r.Route("/api/discover", func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Use(middleware.NewRateLimiter(10, time.Minute).Middleware)
		r.Get("/", discoverHandler.Discover)
	})

	r.Route("/api/explore", func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Get("/feed", exploreHandler.Feed)
		r.Get("/missed", exploreHandler.MissedGenres)
		r.Get("/timemachine", exploreHandler.TimeMachine)
		r.Get("/genre", exploreHandler.Genre)
		r.Get("/related", exploreHandler.RelatedGenres)
		r.Get("/recommendations", exploreHandler.Recommendations)
	})

	r.Route("/api/previews", func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Post("/resolve", previewHandler.Resolve)
	})

	r.Route("/api/settings", func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Get("/", settingsHandler.GetAll)
		r.Put("/", settingsHandler.Set)
	})

	r.Get("/api/ws", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"WebSocket endpoint (use SSE for now)"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}

	log.Println("server stopped")
}
