package taxonomy

import (
	"context"
	"log"
	"time"

	"github.com/fisher/backend/internal/services/musicbrainz"
	"github.com/jackc/pgx/v5/pgxpool"
)

const musicbrainzProgressInterval = 10

// EnrichArtistCountries looks up MusicBrainz country/area data for artists
// already present in artist_genres (populated by the Last.fm co-occurrence
// pass) that don't have an mbid yet, and stores their country of origin.
// This is a separate, later pass because MusicBrainz's free-tier rate limit
// (1 req/sec) makes it too slow to run inline with the main ingestion.
//
// maxArtists caps how many artists are looked up in a single run so this can
// be safely re-run repeatedly (e.g. nightly) to gradually enrich the whole
// artist_genres table without one run taking hours.
func EnrichArtistCountries(ctx context.Context, pool *pgxpool.Pool, maxArtists int) (int, error) {
	if maxArtists <= 0 {
		maxArtists = 200
	}

	log.Printf("musicbrainz enrichment: querying artist_genres for up to %d artists without mbid...", maxArtists)
	rows, err := pool.Query(ctx,
		`SELECT DISTINCT artist_name FROM artist_genres WHERE mbid = '' OR mbid IS NULL LIMIT $1`,
		maxArtists,
	)
	if err != nil {
		return 0, err
	}
	var artistNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			artistNames = append(artistNames, name)
		}
	}
	rows.Close()

	totalArtists := len(artistNames)
	log.Printf("musicbrainz enrichment: found %d artists to enrich (rate-limited to ~1 req/sec)", totalArtists)

	start := time.Now()
	enriched := 0
	cacheHits := 0
	apiCalls := 0

	for i, name := range artistNames {
		if ctx.Err() != nil {
			return enriched, ctx.Err()
		}

		if (i+1)%musicbrainzProgressInterval == 0 || i+1 == totalArtists {
			elapsed := time.Since(start)
			rate := float64(i+1) / elapsed.Seconds()
			log.Printf("musicbrainz enrichment: %d/%d artists processed (%.1f%%) [%.1f artists/sec, %d cache hits, %d API calls, %s elapsed]",
				i+1, totalArtists, float64(i+1)/float64(totalArtists)*100,
				rate, cacheHits, apiCalls, elapsed.Round(time.Second))
		}

		// Check cache first
		if cached, ok := musicbrainz.CacheLookup(ctx, pool, name); ok {
			cacheHits++
			// Still update artist_genres with cached MBID
			if cached.MBID != "" {
				pool.Exec(ctx,
					`UPDATE artist_genres SET mbid = $2 WHERE artist_name = $1 AND (mbid = '' OR mbid IS NULL)`,
					name, cached.MBID,
				)
			}
		// Use cached country/area
		country := musicbrainz.NormalizeCountry(cached.Country)
		if country == "" {
			country = cached.Area
		}
			if country != "" {
				pool.Exec(ctx,
					`UPDATE genres SET countries = array_append(countries, $2), updated_at = NOW()
					 WHERE id IN (SELECT genre_id FROM artist_genres WHERE artist_name = $1)
					   AND NOT ($2 = ANY(countries))`,
					name, country,
				)
				enriched++
			}
			continue
		}

		// Not in cache, call API
		apiCalls++
		info, err := musicbrainz.LookupArtist(name)
		time.Sleep(musicbrainz.RequestDelay)
		if err != nil {
			log.Printf("musicbrainz enrichment: lookup failed for %q: %v", name, err)
			continue
		}

		// Store in cache (even if nil result to avoid re-fetching)
		musicbrainz.CacheStore(ctx, pool, name, info)

		if info == nil {
			continue
		}

		country := musicbrainz.NormalizeCountry(info.Country)
		if country == "" {
			country = info.Area
		}

		_, err = pool.Exec(ctx,
			`UPDATE artist_genres SET mbid = $2 WHERE artist_name = $1`,
			name, info.MBID,
		)
		if err != nil {
			continue
		}

		if country != "" {
			// Fold the artist's country into every genre they're tagged with,
			// growing genres.countries so genre-level regional data emerges
			// automatically from real artist data over time.
			_, err = pool.Exec(ctx,
				`UPDATE genres SET countries = array_append(countries, $2), updated_at = NOW()
				 WHERE id IN (SELECT genre_id FROM artist_genres WHERE artist_name = $1)
				   AND NOT ($2 = ANY(countries))`,
				name, country,
			)
			if err == nil {
				enriched++
			}
		}
	}

	elapsed := time.Since(start)
	log.Printf("musicbrainz enrichment: finished %d/%d artists, %d enriched [%d cache hits, %d API calls, %s elapsed]",
		totalArtists, totalArtists, enriched, cacheHits, apiCalls, elapsed.Round(time.Second))

	return enriched, nil
}
