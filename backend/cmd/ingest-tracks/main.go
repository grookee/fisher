package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/fisher/backend/internal/config"
	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/services/catalog"
)

func main() {
	maxArtists := flag.Int("max-artists", 50, "max number of not-yet-processed artists to handle in this run")
	delayMs := flag.Int("delay-ms", 200, "delay between Deezer API calls, in milliseconds")
	flag.Parse()

	config.LoadEnv()

	if err := database.Connect(); err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	if err := database.Migrate(ctx); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	opts := catalog.DefaultOptions()
	opts.MaxArtists = *maxArtists
	opts.RequestDelay = time.Duration(*delayMs) * time.Millisecond

	start := time.Now()
	log.Printf("calling IngestTracksForArtists (max=%d delay=%s)", opts.MaxArtists, opts.RequestDelay)
	res, err := catalog.IngestTracksForArtists(ctx, database.Pool, opts)
	if err != nil {
		log.Fatalf("track ingestion failed: %v", err)
	}
	log.Printf("track ingestion complete in %s: %+v", time.Since(start), *res)
}
