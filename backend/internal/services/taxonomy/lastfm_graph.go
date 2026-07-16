package taxonomy

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/fisher/backend/internal/services/lastfm"
)

// LastfmGraphOptions controls the size/cost of the empirical co-occurrence
// crawl. Larger values produce a richer graph but issue more (free, but
// rate-limited) requests to the Last.fm API.
type LastfmGraphOptions struct {
	ArtistsPerGenre int           // how many top artists to sample per genre/tag
	RequestDelay    time.Duration // throttle between Last.fm calls, be polite to a free shared API
}

func DefaultLastfmGraphOptions() LastfmGraphOptions {
	return LastfmGraphOptions{ArtistsPerGenre: 15, RequestDelay: 250 * time.Millisecond}
}

// ArtistGenreHit associates an artist with a genre discovered via Last.fm tags.
type ArtistGenreHit struct {
	ArtistName string
	GenreName  string
	Confidence float64
}

type genreScore struct {
	name  string
	score float64
}

// BuildLastfmCooccurrence treats each known genre name as a Last.fm tag,
// fetches its top artists, then reads those artists' own top tags. Whenever
// two known genres co-occur heavily on the same artists, a weighted edge is
// recorded between them. This produces a genre-adjacency graph derived from
// real listener/tagger behaviour - built entirely through Fisher's own free
// Last.fm integration, not by scraping any third-party genre-map site.
func BuildLastfmCooccurrence(ctx context.Context, genreNames []string, opts LastfmGraphOptions) ([]Relation, []ArtistGenreHit, error) {
	if !lastfm.IsConfigured() {
		return nil, nil, nil
	}

	knownGenres := make(map[string]string, len(genreNames))
	for _, g := range genreNames {
		knownGenres[strings.ToLower(g)] = g
	}

	type edgeKey struct{ a, b string }
	edgeWeights := make(map[edgeKey]float64)
	var hits []ArtistGenreHit

	addEdge := func(a, b string, w float64) {
		if a == b {
			return
		}
		if a > b {
			a, b = b, a
		}
		edgeWeights[edgeKey{a, b}] += w
	}

	totalGenres := len(genreNames)
	lastLog := time.Now()

	for i, genreName := range genreNames {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}

		if time.Since(lastLog) > 30*time.Second {
			log.Printf("lastfm co-occurrence: %d/%d genres processed (%.1f%%)", i, totalGenres, float64(i)/float64(totalGenres)*100)
			lastLog = time.Now()
		}

		artists, err := lastfm.FetchTopArtistsForTag(genreName, opts.ArtistsPerGenre)
		time.Sleep(opts.RequestDelay)
		if err != nil || len(artists) == 0 {
			continue
		}

		for _, artistName := range artists {
			tags, err := lastfm.FetchArtistTopTags(artistName)
			time.Sleep(opts.RequestDelay)
			if err != nil {
				continue
			}

			var matched []genreScore
			for _, t := range tags {
				canonical, ok := knownGenres[strings.ToLower(t.Name)]
				if !ok {
					continue
				}
				score := float64(t.Count) / 100.0
				matched = append(matched, genreScore{canonical, score})
				hits = append(hits, ArtistGenreHit{ArtistName: artistName, GenreName: canonical, Confidence: score})
			}

			for i := 0; i < len(matched); i++ {
				for j := i + 1; j < len(matched); j++ {
					addEdge(matched[i].name, matched[j].name, matched[i].score*matched[j].score)
				}
			}
		}
	}

	log.Printf("lastfm co-occurrence: finished %d genres, %d artist-genre hits, %d edges", totalGenres, len(hits), len(edgeWeights))

	relations := make([]Relation, 0, len(edgeWeights))
	for k, w := range edgeWeights {
		relations = append(relations, Relation{From: k.a, To: k.b, Type: "cooccurs_with", Weight: clampWeight(w), Source: "lastfm"})
	}
	return relations, hits, nil
}

func clampWeight(w float64) float64 {
	if w > 1 {
		return 1
	}
	if w < 0 {
		return 0
	}
	return w
}
