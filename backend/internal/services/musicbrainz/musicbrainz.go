package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const baseURL = "https://musicbrainz.org/ws/2"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// RequestDelay is the minimum time between MusicBrainz requests, per their
// free-tier rate limit (1 request/second for unauthenticated clients).
const RequestDelay = 1100 * time.Millisecond

// ArtistInfo is the subset of MusicBrainz artist data Fisher cares about.
type ArtistInfo struct {
	MBID    string
	Name    string
	Country string // ISO 3166-1 alpha-2 country code, e.g. "US", "HU", "JP" - empty if unknown
	Area    string // human-readable area/place name, e.g. "Budapest", "Seoul" - empty if unknown
}

type artistSearchResponse struct {
	Artists []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Score   int    `json:"score"`
		Country string `json:"country"`
		Area    struct {
			Name string `json:"name"`
		} `json:"area"`
	} `json:"artists"`
}

// CacheLookup checks the musicbrainz_cache table for a previously fetched result.
func CacheLookup(ctx context.Context, pool *pgxpool.Pool, artistName string) (*ArtistInfo, bool) {
	var mbid, country, area string
	err := pool.QueryRow(ctx,
		`SELECT mbid, country, area FROM musicbrainz_cache WHERE LOWER(artist_name) = LOWER($1)`,
		artistName,
	).Scan(&mbid, &country, &area)
	if err != nil {
		return nil, false
	}
	return &ArtistInfo{MBID: mbid, Name: artistName, Country: country, Area: area}, true
}

// CacheStore saves a lookup result to the musicbrainz_cache table.
func CacheStore(ctx context.Context, pool *pgxpool.Pool, artistName string, info *ArtistInfo) {
	if info == nil {
		return
	}
	_, err := pool.Exec(ctx,
		`INSERT INTO musicbrainz_cache (artist_name, mbid, country, area, fetched_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (artist_name) DO UPDATE SET
			mbid = EXCLUDED.mbid,
			country = EXCLUDED.country,
			area = EXCLUDED.area,
			fetched_at = NOW()`,
		artistName, info.MBID, info.Country, info.Area,
	)
	if err != nil {
		log.Printf("musicbrainz: cache store failed for %q: %v", artistName, err)
	}
}

// LookupArtist searches MusicBrainz for the best-matching artist by name and
// returns their country/area of origin. Callers are responsible for pacing
// calls at RequestDelay to respect MusicBrainz's free-tier rate limit.
func LookupArtist(name string) (*ArtistInfo, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("artist name is required")
	}

	query := fmt.Sprintf(`artist:"%s"`, strings.ReplaceAll(name, `"`, ``))
	params := url.Values{}
	params.Set("query", query)
	params.Set("fmt", "json")
	params.Set("limit", "5")

	req, err := http.NewRequest(http.MethodGet, baseURL+"/artist/?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "FisherMusicBrainzEnrichment/1.0 ( https://github.com/fisher ; fisher@example.com )")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz returned status %d", resp.StatusCode)
	}

	var result artistSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode musicbrainz response: %w", err)
	}

	if len(result.Artists) == 0 {
		return nil, nil
	}

	// Prefer the highest-scoring result whose name matches case-insensitively;
	// fall back to the top result otherwise.
	best := result.Artists[0]
	for _, a := range result.Artists {
		if strings.EqualFold(a.Name, name) && a.Score >= best.Score {
			best = a
		}
	}

	return &ArtistInfo{
		MBID:    best.ID,
		Name:    best.Name,
		Country: best.Country,
		Area:    best.Area.Name,
	}, nil
}
