package taxonomy

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PersistResult summarizes what an ingestion run did, for logging.
type PersistResult struct {
	GenresInserted       int
	GenresUpdated        int
	AliasesInserted      int
	RelationsUpserted    int
	ArtistGenresUpserted int
}

// Persist merges discovered genre nodes and relation edges into Postgres.
// Existing hand-curated genres (matched case-insensitively by name) are
// enriched in place rather than duplicated; genres that don't exist yet are
// inserted, so the taxonomy grows automatically over time.
func Persist(ctx context.Context, pool *pgxpool.Pool, nodes []GenreNode, relations []Relation, artistHits []ArtistGenreHit) (*PersistResult, error) {
	res := &PersistResult{}

	start := time.Now()
	log.Printf("persist: starting with %d nodes, %d relations, %d artist-genre hits...", len(nodes), len(relations), len(artistHits))

	nameToID := make(map[string]string)
	rows, err := pool.Query(ctx, `SELECT id, LOWER(name) FROM genres`)
	if err != nil {
		return nil, fmt.Errorf("load existing genres: %w", err)
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err == nil {
			nameToID[name] = id
		}
	}
	rows.Close()

	aliasRows, err := pool.Query(ctx, `SELECT genre_id, LOWER(alias) FROM genre_aliases`)
	if err == nil {
		for aliasRows.Next() {
			var id, alias string
			if err := aliasRows.Scan(&id, &alias); err == nil {
				if _, exists := nameToID[alias]; !exists {
					nameToID[alias] = id
				}
			}
		}
		aliasRows.Close()
	}
	log.Printf("persist: loaded %d existing genre names/aliases from database", len(nameToID))

	resolve := func(name string) (string, bool) {
		id, ok := nameToID[strings.ToLower(strings.TrimSpace(name))]
		return id, ok
	}

	for _, n := range nodes {
		key := strings.ToLower(n.Name)
		id, exists := nameToID[key]
		if !exists {
			color := colorFromName(n.Name)
			err := pool.QueryRow(ctx,
				`INSERT INTO genres (name, description, color, x, y, source, wikidata_qid, countries)
				 VALUES ($1, '', $2, 0, 0, 'wikidata', $3, $4)
				 ON CONFLICT (name) DO UPDATE SET wikidata_qid = EXCLUDED.wikidata_qid
				 RETURNING id`,
				n.Name, color, nullableString(n.QID), orEmptyStrings(n.Countries),
			).Scan(&id)
			if err != nil {
				continue
			}
			nameToID[key] = id
			res.GenresInserted++
		} else {
			_, err := pool.Exec(ctx,
				`UPDATE genres SET wikidata_qid = COALESCE(wikidata_qid, $2),
				        countries = CASE WHEN array_length($3::text[], 1) > 0 THEN $3 ELSE countries END,
				        updated_at = NOW()
				 WHERE id = $1`,
				id, nullableString(n.QID), orEmptyStrings(n.Countries),
			)
			if err == nil {
				res.GenresUpdated++
			}
		}

		for _, alias := range n.Aliases {
			if alias == "" || strings.EqualFold(alias, n.Name) {
				continue
			}
			tag, err := pool.Exec(ctx,
				`INSERT INTO genre_aliases (genre_id, alias, source) VALUES ($1, $2, 'wikidata')
				 ON CONFLICT (genre_id, alias) DO NOTHING`,
				id, alias,
			)
			if err == nil && tag.RowsAffected() > 0 {
				res.AliasesInserted++
				nameToID[strings.ToLower(alias)] = id
			}
		}
	}

	for _, rel := range relations {
		fromID, ok1 := resolve(rel.From)
		toID, ok2 := resolve(rel.To)
		if !ok1 || !ok2 || fromID == toID {
			continue
		}
		upsertRelation(ctx, pool, fromID, toID, rel.Type, rel.Weight, rel.Source)
		upsertRelation(ctx, pool, toID, fromID, rel.Type, rel.Weight, rel.Source)
		res.RelationsUpserted++
	}

	for _, hit := range artistHits {
		genreID, ok := resolve(hit.GenreName)
		if !ok {
			continue
		}
		_, err := pool.Exec(ctx,
			`INSERT INTO artist_genres (artist_name, genre_id, confidence, source)
			 VALUES ($1, $2, $3, 'lastfm')
			 ON CONFLICT (artist_name, genre_id, source) DO UPDATE SET confidence = GREATEST(artist_genres.confidence, EXCLUDED.confidence)`,
			hit.ArtistName, genreID, hit.Confidence,
		)
		if err == nil {
			res.ArtistGenresUpserted++
		}
	}

	elapsed := time.Since(start)
	log.Printf("persist: finished in %s — %d genres inserted, %d updated, %d aliases, %d relations, %d artist-genre links",
		elapsed.Round(time.Millisecond), res.GenresInserted, res.GenresUpdated,
		res.AliasesInserted, res.RelationsUpserted, res.ArtistGenresUpserted)

	return res, nil
}

func upsertRelation(ctx context.Context, pool *pgxpool.Pool, fromID, toID, relType string, weight float64, source string) {
	pool.Exec(ctx,
		`INSERT INTO genre_relations (genre_id, related_genre_id, relation_type, weight, source)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (genre_id, related_genre_id, relation_type)
		 DO UPDATE SET weight = (genre_relations.weight + EXCLUDED.weight) / 2, updated_at = NOW()`,
		fromID, toID, relType, weight, source,
	)
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func orEmptyStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// colorFromName deterministically derives a stable hex color for newly
// discovered genres so the existing "genre map" visualization keeps working
// without manual curation.
func colorFromName(name string) string {
	h := 0
	for _, c := range name {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	palette := []string{
		"#e74c3c", "#f39c12", "#9b59b6", "#3498db", "#1abc9c", "#2ecc71",
		"#e91e63", "#795548", "#c0392b", "#00bcd4", "#8bc34a", "#ff5722",
		"#ff4081", "#4caf50", "#3f51b5", "#ff9800", "#607d8b", "#ff6f00",
		"#e040fb", "#009688",
	}
	r := rand.New(rand.NewSource(int64(h)))
	return palette[r.Intn(len(palette))]
}
