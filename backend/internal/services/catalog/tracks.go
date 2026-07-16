package catalog

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	deezer "github.com/fisher/backend/internal/services/deezer"
)

// Options controls how much work a track-ingestion run performs.
type Options struct {
	MaxArtists   int           // how many not-yet-processed artists to handle in this run
	RequestDelay time.Duration // throttle between Deezer API calls
}

func DefaultOptions() Options {
	return Options{MaxArtists: 50, RequestDelay: 200 * time.Millisecond}
}

// Result summarizes what a track-ingestion run did, for logging.
type Result struct {
	ArtistsProcessed int
	ArtistsResolved  int
	ArtistsNoTracks  int
	TracksUpserted   int
	TrackGenreLinks  int
}

type pendingArtist struct {
	name     string
	genreIDs []string
}

// IngestTracksForArtists walks artists already discovered by the taxonomy
// pipeline (artist_genres), resolves each one on Deezer, and stores their
// top tracks in `tracks` + `track_genres`. Progress is tracked in
// artist_track_progress so repeated runs incrementally grow the catalog.
func IngestTracksForArtists(ctx context.Context, pool *pgxpool.Pool, opts Options) (*Result, error) {
	if opts.MaxArtists <= 0 {
		opts.MaxArtists = 50
	}

	artists, err := loadPendingArtists(ctx, pool, opts.MaxArtists)
	if err != nil {
		return nil, fmt.Errorf("load pending artists: %w", err)
	}
	log.Printf("loaded %d pending artists", len(artists))

	res := &Result{}
	for _, a := range artists {
		res.ArtistsProcessed++
		if ctx.Err() != nil {
			return res, ctx.Err()
		}

		artist, err := deezer.SearchArtist(a.name)
		time.Sleep(opts.RequestDelay)
		if err != nil {
			log.Printf("  deezer resolve failed for %q: %v", a.name, err)
			markProgress(ctx, pool, a.name, "")
			continue
		}
		if artist.ID == 0 {
			log.Printf("  not found on Deezer: %q", a.name)
			markProgress(ctx, pool, a.name, "")
			continue
		}
		res.ArtistsResolved++

		tracks, err := deezer.GetArtistTopTracks(artist.ID, 20)
		time.Sleep(opts.RequestDelay)
		if err != nil {
			log.Printf("  top-tracks failed for %q: %v", a.name, err)
			continue
		}
		if len(tracks) == 0 {
			log.Printf("  no tracks for %q (deezerID=%d)", a.name, artist.ID)
			res.ArtistsNoTracks++
			continue
		}

		log.Printf("  %d tracks for %q", len(tracks), a.name)
		idMap, err := batchUpsertTracks(ctx, pool, tracks, a.name)
		if err != nil {
			log.Printf("  batch upsert failed for %q: %v", a.name, err)
			continue
		}
		res.TracksUpserted += len(idMap)

		if len(a.genreIDs) > 0 && len(idMap) > 0 {
			links, err := batchInsertTrackGenres(ctx, pool, idMap, a.genreIDs)
			if err != nil {
				log.Printf("  batch genre link failed for %q: %v", a.name, err)
			}
			res.TrackGenreLinks += links
		}

		markProgress(ctx, pool, a.name, fmt.Sprintf("%d", artist.ID))
	}

	return res, nil
}

func loadPendingArtists(ctx context.Context, pool *pgxpool.Pool, max int) ([]pendingArtist, error) {
	rows, err := pool.Query(ctx,
		`SELECT ag.artist_name, array_agg(DISTINCT ag.genre_id)
		 FROM artist_genres ag
		 WHERE ag.artist_name NOT IN (SELECT artist_name FROM artist_track_progress)
		 GROUP BY ag.artist_name
		 ORDER BY MAX(ag.confidence) DESC
		 LIMIT $1`,
		max,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []pendingArtist
	for rows.Next() {
		var p pendingArtist
		if err := rows.Scan(&p.name, &p.genreIDs); err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

func batchUpsertTracks(ctx context.Context, pool *pgxpool.Pool, tracks []deezer.TopTrack, artistName string) (map[string]string, error) {
	if len(tracks) == 0 {
		return nil, nil
	}

	deezerIDs := make([]string, 0, len(tracks))
	for _, t := range tracks {
		if t.DeezerID != "" {
			deezerIDs = append(deezerIDs, t.DeezerID)
		}
	}
	if len(deezerIDs) == 0 {
		return nil, nil
	}

	existing := make(map[string]string) // deezer_id -> track_id
	rows, err := pool.Query(ctx,
		`SELECT id, deezer_id FROM tracks WHERE deezer_id = ANY($1)`, deezerIDs)
	if err != nil {
		return nil, fmt.Errorf("query existing tracks: %w", err)
	}
	for rows.Next() {
		var id, did string
		if err := rows.Scan(&id, &did); err == nil {
			existing[did] = id
		}
	}
	rows.Close()

	var toInsert []deezer.TopTrack
	var toUpdate []deezer.TopTrack
	for _, t := range tracks {
		if _, ok := existing[t.DeezerID]; ok {
			toUpdate = append(toUpdate, t)
		} else {
			toInsert = append(toInsert, t)
		}
	}

	result := make(map[string]string, len(tracks))

	if len(toUpdate) > 0 {
		for _, t := range toUpdate {
			var id string
			err := pool.QueryRow(ctx,
				`UPDATE tracks SET title=$1, album=$2, album_art_url=$3, duration_ms=$4, preview_url=$5
				 WHERE deezer_id=$6 RETURNING id`,
				t.Title, t.AlbumName, t.AlbumArtURL, t.DurationSec*1000, t.PreviewURL, t.DeezerID,
			).Scan(&id)
			if err != nil {
				log.Printf("  update track %q failed: %v", t.DeezerID, err)
			} else {
				result[t.DeezerID] = id
			}
		}
	}

	if len(toInsert) > 0 {
		const icols = 8
		iargs := make([]interface{}, 0, len(toInsert)*icols)
		ivalueParts := make([]string, 0, len(toInsert))
		for _, t := range toInsert {
			offset := len(iargs)
			iargs = append(iargs,
				t.Title, artistName, t.AlbumName, t.AlbumArtURL,
				t.DurationSec*1000, t.DeezerID, t.PreviewURL, 0,
			)
			ivalueParts = append(ivalueParts,
				fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
					offset+1, offset+2, offset+3, offset+4,
					offset+5, offset+6, offset+7, offset+8),
			)
		}
		iquery := fmt.Sprintf(
			`INSERT INTO tracks (title, artist, album, album_art_url, duration_ms, deezer_id, preview_url, release_year)
			 VALUES %s
			 RETURNING id, deezer_id`,
			strings.Join(ivalueParts, ","),
		)
		irows, err := pool.Query(ctx, iquery, iargs...)
		if err != nil {
			return nil, fmt.Errorf("batch insert tracks: %w", err)
		}
		for irows.Next() {
			var id, did string
			if err := irows.Scan(&id, &did); err == nil {
				result[did] = id
			}
		}
		irows.Close()
	}

	return result, nil
}

func batchInsertTrackGenres(ctx context.Context, pool *pgxpool.Pool, trackIDMap map[string]string, genreIDs []string) (int, error) {
	if len(trackIDMap) == 0 || len(genreIDs) == 0 {
		return 0, nil
	}

	const cols = 2
	total := len(trackIDMap) * len(genreIDs)
	args := make([]interface{}, 0, total*cols)
	valueParts := make([]string, 0, total)

	for _, trackID := range trackIDMap {
		for _, genreID := range genreIDs {
			offset := len(args)
			args = append(args, trackID, genreID)
			valueParts = append(valueParts,
				fmt.Sprintf("($%d,$%d)", offset+1, offset+2),
			)
		}
	}

	query := fmt.Sprintf(
		`INSERT INTO track_genres (track_id, genre_id) VALUES %s ON CONFLICT DO NOTHING`,
		strings.Join(valueParts, ","),
	)

	tag, err := pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func markProgress(ctx context.Context, pool *pgxpool.Pool, artistName, externalID string) {
	pool.Exec(ctx,
		`INSERT INTO artist_track_progress (artist_name, spotify_artist_id, fetched_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (artist_name) DO UPDATE SET spotify_artist_id = EXCLUDED.spotify_artist_id, fetched_at = NOW()`,
		artistName, externalID,
	)
}
