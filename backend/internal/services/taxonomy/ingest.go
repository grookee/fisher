package taxonomy

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Options controls how much work an ingestion run performs. Every knob is
// tunable so this pipeline can run inside free-tier constraints (a small VM,
// a GitHub Actions cron job, etc.) without ever touching a paid API.
type Options struct {
	SkipWikidata bool
	SkipLastfm   bool
	Lastfm       LastfmGraphOptions
}

func DefaultOptions() Options {
	return Options{Lastfm: DefaultLastfmGraphOptions()}
}

// Run executes one full genre-taxonomy ingestion pass:
//  1. Pull the deep genre taxonomy + influence graph + countries from Wikidata (free, no key).
//  2. Pull empirical genre co-occurrence from Last.fm tag/artist data (free, API key already configured).
//  3. Merge everything into Postgres, enriching Fisher's existing hand-seeded genres and
//     growing the taxonomy with anything new that was discovered.
//
// Meant to run on a schedule (cron / systemd timer / CI job) via
// `cmd/ingest-genres`, not on the request path.
func Run(ctx context.Context, pool *pgxpool.Pool, opts Options) error {
	var nodes []GenreNode
	var relations []Relation

	if !opts.SkipWikidata {
		log.Println("taxonomy: fetching genre taxonomy from Wikidata...")
		wdNodes, wdRelations, err := FetchGenreTaxonomy(ctx)
		if err != nil {
			return fmt.Errorf("wikidata fetch: %w", err)
		}
		log.Printf("taxonomy: wikidata returned %d genres, %d relations", len(wdNodes), len(wdRelations))
		nodes = append(nodes, wdNodes...)
		relations = append(relations, wdRelations...)
	}

	firstPass, err := Persist(ctx, pool, nodes, relations, nil)
	if err != nil {
		return fmt.Errorf("persist wikidata taxonomy: %w", err)
	}
	log.Printf("taxonomy: persisted wikidata pass: %+v", *firstPass)

	if !opts.SkipLastfm {
		genreNames, err := AllGenreNames(ctx, pool)
		if err != nil {
			return fmt.Errorf("load genre names: %w", err)
		}
		log.Printf("taxonomy: building last.fm co-occurrence graph over %d genres...", len(genreNames))
		lfRelations, hits, err := BuildLastfmCooccurrence(ctx, genreNames, opts.Lastfm)
		if err != nil {
			return fmt.Errorf("lastfm cooccurrence: %w", err)
		}
		log.Printf("taxonomy: last.fm returned %d relations, %d artist-genre hits", len(lfRelations), len(hits))

		secondPass, err := Persist(ctx, pool, nil, lfRelations, hits)
		if err != nil {
			return fmt.Errorf("persist lastfm graph: %w", err)
		}
		log.Printf("taxonomy: persisted last.fm pass: %+v", *secondPass)
	}

	return nil
}

// AllGenreNames returns every canonical genre name currently stored, used to
// seed the Last.fm co-occurrence crawl.
func AllGenreNames(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT name FROM genres ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err == nil {
			names = append(names, n)
		}
	}
	return names, nil
}
