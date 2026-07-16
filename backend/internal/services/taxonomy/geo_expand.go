package taxonomy

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/fisher/backend/internal/services/lastfm"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GeoExpandOptions controls the free artist-discovery-by-country pass.
type GeoExpandOptions struct {
	Countries         []string
	ArtistsPerCountry int
	RequestDelay      time.Duration
}

func DefaultGeoExpandOptions() GeoExpandOptions {
	return GeoExpandOptions{
		Countries:         DefaultGeoCountries,
		ArtistsPerCountry: 30,
		RequestDelay:      250 * time.Millisecond,
	}
}

// DefaultGeoCountries is a deliberately broad mix of large and small music
// markets/cultures, so artist coverage isn't dominated by the usual
// Anglophone charts. This list is the direct, free, automatic answer to
// "surface small artists from small cultures" - expand it freely.
var DefaultGeoCountries = []string{
	"United States", "United Kingdom", "Japan", "South Korea", "Brazil",
	"Hungary", "Iceland", "Mongolia", "Estonia", "Ghana", "Vietnam", "Peru",
	"Finland", "Portugal", "Nigeria", "Indonesia", "Poland", "Greece",
	"Turkey", "Israel", "Kenya", "Colombia", "Thailand", "Ukraine",
	"Czech Republic", "Morocco", "New Zealand", "Chile", "Serbia", "Croatia",
}

// ExpandArtistsByGeo pulls each country's Last.fm top-artist chart (free,
// using Fisher's existing LASTFM_API_KEY) and folds the results into
// artist_genres (via each artist's own top tags, matched against Fisher's
// genre taxonomy) and genres.countries (using the chart's country directly,
// without needing a separate MusicBrainz lookup for these artists). This is
// the main lever for growing artist coverage beyond what genre-tag sampling
// alone finds.
func ExpandArtistsByGeo(ctx context.Context, pool *pgxpool.Pool, opts GeoExpandOptions) (int, error) {
	if !lastfm.IsConfigured() {
		return 0, nil
	}
	if len(opts.Countries) == 0 {
		opts.Countries = DefaultGeoCountries
	}
	if opts.ArtistsPerCountry <= 0 {
		opts.ArtistsPerCountry = 30
	}

	log.Printf("geo expansion: loading genre names from database...")
	genreNames, err := AllGenreNames(ctx, pool)
	if err != nil {
		return 0, err
	}
	knownGenres := make(map[string]string, len(genreNames))
	for _, g := range genreNames {
		knownGenres[strings.ToLower(g)] = g
	}

	totalCountries := len(opts.Countries)
	log.Printf("geo expansion: starting %d countries, %d artists/country (rate-limited to ~4 req/sec)",
		totalCountries, opts.ArtistsPerCountry)

	var hits []ArtistGenreHit
	countryByArtist := make(map[string]string)

	start := time.Now()
	for i, country := range opts.Countries {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		log.Printf("geo expansion: [%d/%d] fetching top artists for %s...", i+1, totalCountries, country)
		artists, err := lastfm.FetchTopArtistsByCountry(country, opts.ArtistsPerCountry)
		time.Sleep(opts.RequestDelay)
		if err != nil || len(artists) == 0 {
			log.Printf("geo expansion: [%d/%d] %s: no artists found (err=%v)", i+1, totalCountries, country, err)
			continue
		}

		log.Printf("geo expansion: [%d/%d] %s: got %d artists, fetching tags...", i+1, totalCountries, country, len(artists))
		for _, artistName := range artists {
			countryByArtist[artistName] = country

			tags, err := lastfm.FetchArtistTopTags(artistName)
			time.Sleep(opts.RequestDelay)
			if err != nil {
				continue
			}
			for _, t := range tags {
				canonical, ok := knownGenres[strings.ToLower(t.Name)]
				if !ok {
					continue
				}
				hits = append(hits, ArtistGenreHit{
					ArtistName: artistName,
					GenreName:  canonical,
					Confidence: float64(t.Count) / 100.0,
				})
			}
		}
	}

	log.Printf("geo expansion: persisting %d artist-genre hits...", len(hits))
	result, err := Persist(ctx, pool, nil, nil, hits)
	if err != nil {
		return 0, err
	}

	// Fold each artist's chart country directly into every genre they're
	// tagged with - cheaper and more direct than the MusicBrainz pass, since
	// the country is already known from the chart itself.
	log.Printf("geo expansion: updating genres.countries for %d artists...", len(countryByArtist))
	for artistName, country := range countryByArtist {
		pool.Exec(ctx,
			`UPDATE genres SET countries = array_append(countries, $2), updated_at = NOW()
			 WHERE id IN (SELECT genre_id FROM artist_genres WHERE artist_name = $1)
			   AND NOT ($2 = ANY(countries))`,
			artistName, country,
		)
	}

	elapsed := time.Since(start)
	log.Printf("geo expansion: finished %d countries, %d artist-genre links upserted [%s elapsed]",
		totalCountries, result.ArtistGenresUpserted, elapsed.Round(time.Second))

	return result.ArtistGenresUpserted, nil
}
