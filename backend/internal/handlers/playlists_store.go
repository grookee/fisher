package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/models"
	applemusicsvc "github.com/fisher/backend/internal/services/applemusic"
	spotifysvc "github.com/fisher/backend/internal/services/spotify"
)

func searchExt(ctx context.Context, query string) []models.Track {
	tracks := []models.Track{}

	if spotifysvc.Client != nil {
		svc := spotifysvc.New()
		token := spotifyTokenForUser(ctx, userIDFromContext(ctx))
		results, err := svc.SearchTracksWithAccessToken(query, 10, token)
		if err == nil {
			for _, r := range results {
				track, saveErr := saveTrack(ctx, models.Track{
					Title:           r.Title,
					Artist:          r.Artist,
					Album:           r.Album,
					AlbumArtURL:     r.AlbumArt,
					DurationMs:      r.Duration,
					SpotifyURI:      r.SpotifyID,
					PreviewURL:      r.Preview,
					ReleaseYear:     r.ReleaseYear,
					Danceability:    r.Danceability,
					Energy:          r.Energy,
					Valence:         r.Valence,
					Acousticness:    r.Acousticness,
					Instrumentalness: r.Instrumentalness,
					Speechiness:     r.Speechiness,
					Tempo:           r.Tempo,
				})
				if saveErr == nil {
					tracks = append(tracks, track)
				}
			}
			if len(tracks) > 0 {
				return tracks
			}
		}
	}

	if os.Getenv("APPLE_MUSIC_DEVELOPER_TOKEN") == "" {
		return tracks
	}

	apple := applemusicsvc.New()
	apple.Init()
	results, err := apple.SearchTracks(query, 10)
	if err != nil {
		return tracks
	}

	for _, r := range results {
		track, saveErr := saveTrack(ctx, models.Track{
			Title:        r.Title,
			Artist:       r.Artist,
			Album:        r.Album,
			AlbumArtURL:  r.AlbumArt,
			DurationMs:   r.Duration,
			AppleMusicID: r.AppleID,
			PreviewURL:   r.Preview,
		})
		if saveErr == nil {
			tracks = append(tracks, track)
		}
	}

	return tracks
}

func saveTrack(ctx context.Context, t models.Track) (models.Track, error) {
	if t.SpotifyURI != "" {
		if err := database.Pool.QueryRow(ctx,
			`SELECT id FROM tracks WHERE spotify_uri = $1 LIMIT 1`,
			t.SpotifyURI,
		).Scan(&t.ID); err == nil {
			return t, nil
		}
	}

	if t.AppleMusicID != "" {
		if err := database.Pool.QueryRow(ctx,
			`SELECT id FROM tracks WHERE apple_music_id = $1 LIMIT 1`,
			t.AppleMusicID,
		).Scan(&t.ID); err == nil {
			return t, nil
		}
	}

	err := database.Pool.QueryRow(ctx,
		`INSERT INTO tracks (title, artist, album, album_art_url, duration_ms, spotify_uri, apple_music_id, preview_url, release_year, danceability, energy, valence, acousticness, instrumentalness, speechiness, tempo)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		 RETURNING id`,
		t.Title, t.Artist, t.Album, t.AlbumArtURL, t.DurationMs, t.SpotifyURI, t.AppleMusicID, t.PreviewURL,
		t.ReleaseYear, t.Danceability, t.Energy, t.Valence, t.Acousticness, t.Instrumentalness, t.Speechiness, t.Tempo,
	).Scan(&t.ID)
	if err != nil {
		return models.Track{}, err
	}

	return t, nil
}

func canEdit(r *http.Request, playlistID, userID string) bool {
	if isOwner(r.Context(), playlistID, userID) {
		return true
	}

	var permission string
	err := database.Pool.QueryRow(r.Context(),
		`SELECT permission FROM collaborations WHERE playlist_id = $1 AND user_id = $2`,
		playlistID, userID,
	).Scan(&permission)
	if err != nil {
		return false
	}
	return permission == "edit" || permission == "admin"
}

func canAdmin(r *http.Request, playlistID, userID string) bool {
	if isOwner(r.Context(), playlistID, userID) {
		return true
	}

	var permission string
	err := database.Pool.QueryRow(r.Context(),
		`SELECT permission FROM collaborations WHERE playlist_id = $1 AND user_id = $2`,
		playlistID, userID,
	).Scan(&permission)
	if err != nil {
		return false
	}
	return permission == "admin"
}

func isOwner(ctx context.Context, playlistID, userID string) bool {
	var ownerID string
	err := database.Pool.QueryRow(ctx,
		`SELECT owner_id FROM playlists WHERE id = $1`, playlistID,
	).Scan(&ownerID)
	if err != nil {
		return false
	}
	return ownerID == userID
}

func listTracks(ctx context.Context, playlistID string) ([]models.Track, error) {
	rows, err := database.Pool.Query(ctx,
		`SELECT t.id, t.title, t.artist, t.album, t.album_art_url, t.duration_ms, t.spotify_uri, t.apple_music_id, t.preview_url,
		        t.release_year, t.danceability, t.energy, t.valence, t.acousticness, t.instrumentalness, t.speechiness, t.tempo
		 FROM tracks t
		 JOIN playlist_tracks pt ON pt.track_id = t.id
		 WHERE pt.playlist_id = $1
		 ORDER BY pt.position`,
		playlistID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracks := []models.Track{}
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.AlbumArtURL, &t.DurationMs, &t.SpotifyURI, &t.AppleMusicID, &t.PreviewURL,
			&t.ReleaseYear, &t.Danceability, &t.Energy, &t.Valence, &t.Acousticness, &t.Instrumentalness, &t.Speechiness, &t.Tempo); err == nil {
			tracks = append(tracks, t)
		}
	}
	return tracks, nil
}
