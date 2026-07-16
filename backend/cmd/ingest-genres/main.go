package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/fisher/backend/internal/config"
	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/services/lastfm"
	"github.com/fisher/backend/internal/services/taxonomy"
)

func main() {
	skipWikidata := flag.Bool("skip-wikidata", false, "skip the Wikidata taxonomy pass")
	skipLastfm := flag.Bool("skip-lastfm", false, "skip the Last.fm co-occurrence pass")
	artistsPerGenre := flag.Int("artists-per-genre", 15, "how many top artists to sample per genre when building the Last.fm co-occurrence graph")
	delayMs := flag.Int("delay-ms", 250, "delay between Last.fm API calls, in milliseconds (keep this polite - it's a free/shared API)")
	enrichArtists := flag.Bool("enrich-artist-countries", false, "run the (slow, rate-limited) MusicBrainz artist country enrichment pass after ingestion")
	maxArtistsToEnrich := flag.Int("max-artists-to-enrich", 200, "max number of artists to look up on MusicBrainz per run (rate-limited to ~1/sec)")
	recomputeLayout := flag.Bool("recompute-layout", false, "recompute genre map x/y coordinates from the adjacency graph after ingestion")
	expandGeo := flag.Bool("expand-geo", false, "pull Last.fm per-country top-artist charts to grow artist/genre/country coverage beyond tag sampling")
	artistsPerCountry := flag.Int("artists-per-country", 30, "how many top artists to sample per country when -expand-geo is set")
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

	lastfm.Init()

	opts := taxonomy.DefaultOptions()
	opts.SkipWikidata = *skipWikidata
	opts.SkipLastfm = *skipLastfm
	opts.Lastfm.ArtistsPerGenre = *artistsPerGenre
	opts.Lastfm.RequestDelay = time.Duration(*delayMs) * time.Millisecond

	start := time.Now()
	if err := taxonomy.Run(ctx, database.Pool, opts); err != nil {
		log.Fatalf("genre taxonomy ingestion failed: %v", err)
	}
	log.Printf("genre taxonomy ingestion complete in %s", time.Since(start))

	if *expandGeo {
		geoStart := time.Now()
		geoOpts := taxonomy.DefaultGeoExpandOptions()
		geoOpts.ArtistsPerCountry = *artistsPerCountry
		geoOpts.RequestDelay = time.Duration(*delayMs) * time.Millisecond
		upserted, err := taxonomy.ExpandArtistsByGeo(ctx, database.Pool, geoOpts)
		if err != nil {
			log.Printf("geo artist expansion failed: %v", err)
		} else {
			log.Printf("geo artist expansion complete in %s: %d artist-genre links upserted", time.Since(geoStart), upserted)
		}
	}

	if *enrichArtists {
		enrichStart := time.Now()
		enriched, err := taxonomy.EnrichArtistCountries(ctx, database.Pool, *maxArtistsToEnrich)
		if err != nil {
			log.Printf("musicbrainz artist country enrichment failed: %v", err)
		} else {
			log.Printf("musicbrainz artist country enrichment complete in %s: %d artists enriched", time.Since(enrichStart), enriched)
		}
	}

	if *recomputeLayout {
		updated, err := taxonomy.RecomputeLayout(ctx, database.Pool, taxonomy.DefaultLayoutOptions())
		if err != nil {
			log.Printf("genre map layout recompute failed (non-fatal): %v", err)
		} else {
			log.Printf("genre map layout recompute repositioned %d genres", updated)
		}
	}
}
