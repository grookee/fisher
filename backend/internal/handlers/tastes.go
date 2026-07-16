package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/fisher/backend/internal/auth"
	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/debug"
	"github.com/fisher/backend/internal/models"
	"github.com/fisher/backend/internal/services/lastfm"
	spotifysvc "github.com/fisher/backend/internal/services/spotify"
)

type TasteHandler struct{}

func NewTasteHandler() *TasteHandler {
	return &TasteHandler{}
}

func (h *TasteHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	claims := auth.GetUser(r.Context())

	if userID == "" {
		userID = claims.UserID
	}

	if userID != claims.UserID {
		var count int
		database.Pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM taste_shares WHERE user_id = $1 AND shared_with = $2`,
			userID, claims.UserID,
		).Scan(&count)
		if count == 0 {
			respondError(w, http.StatusForbidden, "taste profile not shared with you")
			return
		}
	}

	profile := models.TasteProfile{
		UserID:    userID,
		Genres:    []models.GenreWeight{},
		TopTracks: []models.Track{},
	}

	genreRows, err := database.Pool.Query(r.Context(),
		`SELECT tg.genre_id, tg.weight
		 FROM taste_genres tg
		 WHERE tg.user_id = $1
		 ORDER BY tg.weight DESC`,
		userID,
	)
	if err == nil {
		defer genreRows.Close()
		for genreRows.Next() {
			var gw models.GenreWeight
			if err := genreRows.Scan(&gw.GenreID, &gw.Weight); err != nil {
				continue
			}
			profile.Genres = append(profile.Genres, gw)
		}
	}

	var topArtists []string
	database.Pool.QueryRow(r.Context(),
		`SELECT top_artists FROM taste_profiles WHERE user_id = $1`, userID,
	).Scan(&topArtists)
	profile.TopArtists = topArtists

	var topTracksJSON []byte
	database.Pool.QueryRow(r.Context(),
		`SELECT top_tracks FROM taste_profiles WHERE user_id = $1`, userID,
	).Scan(&topTracksJSON)
	if len(topTracksJSON) > 0 && string(topTracksJSON) != "null" {
		json.Unmarshal(topTracksJSON, &profile.TopTracks)
	}

	shareRows, err := database.Pool.Query(r.Context(),
		`SELECT shared_with FROM taste_shares WHERE user_id = $1`, userID,
	)
	if err == nil {
		defer shareRows.Close()
		for shareRows.Next() {
			var sw string
			shareRows.Scan(&sw)
			profile.SharedWith = append(profile.SharedWith, sw)
		}
	}

	respondJSON(w, http.StatusOK, profile)
}

func (h *TasteHandler) UpdateGenres(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())

	var req struct {
		Genres []models.GenreWeight `json:"genres"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tx, err := database.Pool.Begin(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback(r.Context())

	tx.Exec(r.Context(), `DELETE FROM taste_genres WHERE user_id = $1`, claims.UserID)

	for _, g := range req.Genres {
		tx.Exec(r.Context(),
			`INSERT INTO taste_genres (user_id, genre_id, weight) VALUES ($1, $2, $3)`,
			claims.UserID, g.GenreID, g.Weight,
		)
	}

	if err := tx.Commit(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save genres")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "genres updated"})
}

func (h *TasteHandler) Share(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())

	var req models.ShareTasteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	for _, uid := range req.UserIDs {
		database.Pool.Exec(r.Context(),
			`INSERT INTO taste_shares (user_id, shared_with) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			claims.UserID, uid,
		)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "taste shared"})
}

func (h *TasteHandler) Unshare(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	database.Pool.Exec(r.Context(),
		`DELETE FROM taste_shares WHERE user_id = $1 AND shared_with = $2`,
		claims.UserID, req.UserID,
	)

	respondJSON(w, http.StatusOK, map[string]string{"message": "taste unshared"})
}

func (h *TasteHandler) FriendsTastes(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())

	rows, err := database.Pool.Query(r.Context(),
		`SELECT DISTINCT u.id FROM taste_shares ts
		 JOIN users u ON u.id = ts.user_id
		 WHERE ts.shared_with = $1
		 LIMIT 50`,
		claims.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch shared tastes")
		return
	}
	defer rows.Close()

	type FriendTaste struct {
		UserID string              `json:"user_id"`
		Genres []models.GenreWeight `json:"genres"`
	}

	results := []FriendTaste{}
	for rows.Next() {
		var uid string
		rows.Scan(&uid)

		ft := FriendTaste{UserID: uid}
		gRows, _ := database.Pool.Query(r.Context(),
			`SELECT genre_id, weight FROM taste_genres WHERE user_id = $1 ORDER BY weight DESC LIMIT 5`,
			uid,
		)
		if gRows != nil {
			for gRows.Next() {
				var gw models.GenreWeight
				gRows.Scan(&gw.GenreID, &gw.Weight)
				ft.Genres = append(ft.Genres, gw)
			}
			gRows.Close()
		}
		results = append(results, ft)
	}

	if results == nil {
		results = []FriendTaste{}
	}

	respondJSON(w, http.StatusOK, results)
}

func (h *TasteHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())
	debug.Logf("Analyze called for user=%s", claims.UserID)

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "medium_term"
	}
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "spotify"
	}
	debug.Logf("Analyze params: time_range=%s source=%s", timeRange, source)

	genreRows, err := database.Pool.Query(r.Context(), `SELECT id, name FROM genres`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch genres")
		return
	}
	defer genreRows.Close()
	nameToID := make(map[string]string)
	for genreRows.Next() {
		var id, name string
		genreRows.Scan(&id, &name)
		nameToID[name] = id
	}

	var ourGenres map[string]float64
	var topArtists []string
	var topTracks []models.Track
	var audioFeatures *models.AudioFeatures

	if source == "lastfm" || source == "combined" {
		debug.Log("Analyze: checking last.fm source")
		var lastfmUser string
		err := database.Pool.QueryRow(r.Context(),
			`SELECT value FROM user_settings WHERE user_id = $1 AND key = 'lastfm_username'`,
			claims.UserID,
		).Scan(&lastfmUser)
		if err != nil || lastfmUser == "" {
			debug.Logf("Analyze: last.fm username not found for user=%s: %v", claims.UserID, err)
			if source == "lastfm" {
				respondError(w, http.StatusBadRequest, "last.fm username not set. Add your last.fm username in settings first.")
				return
			}
		} else if !lastfm.IsConfigured() {
			debug.Log("Analyze: LASTFM_API_KEY not configured")
			if source == "lastfm" {
				respondError(w, http.StatusBadRequest, "last.fm API key not configured on the server. Contact the administrator.")
				return
			}
		} else {
			lfArtists, lfErr := lastfm.FetchTopArtists(lastfmUser, 20)
			if lfErr != nil {
				debug.Logf("Analyze: last.fm fetch failed for user=%s: %v", claims.UserID, lfErr)
				if source == "lastfm" {
					respondError(w, http.StatusInternalServerError, fmt.Sprintf("last.fm analysis failed: %v", lfErr))
					return
				}
			} else {
				lfGenreCount := make(map[string]int)
				for _, a := range lfArtists {
					for _, tag := range a.Tags {
						mapped := lastfm.MapTagsToOurGenre([]string{tag})
						if mapped != "" {
							lfGenreCount[mapped]++
						}
					}
				}
				lfGenres := make(map[string]float64)
				for name, count := range lfGenreCount {
					if id, ok := nameToID[name]; ok {
						lfGenres[id] += float64(count)
					}
				}
				var maxW float64
				for _, w := range lfGenres {
					if w > maxW {
						maxW = w
					}
				}
				if maxW > 0 {
					ourGenres = make(map[string]float64)
					for id, w := range lfGenres {
						ourGenres[id] = w / maxW
					}
				}
				for _, a := range lfArtists {
					topArtists = append(topArtists, a.Name)
				}
				debug.Logf("Analyze: last.fm returned %d artists, %d genres", len(lfArtists), len(lfGenres))
			}
		}
	}

	if source == "spotify" || source == "combined" {
		debug.Log("Analyze: checking spotify source")
		var accessToken string
		err := database.Pool.QueryRow(r.Context(),
			`SELECT access_token FROM user_accounts WHERE user_id = $1 AND service = 'spotify'`,
			claims.UserID,
		).Scan(&accessToken)
		if err != nil {
			debug.Logf("Analyze: no spotify token for user=%s: %v", claims.UserID, err)
		}
		if err == nil {
			debug.Logf("Analyze: calling Spotify AnalyzeTasteExtended for user=%s time_range=%s", claims.UserID, timeRange)
			result, spErr := spotifysvc.AnalyzeTasteExtended(accessToken, timeRange)
			if spErr != nil {
				debug.Logf("Analyze: Spotify API error for user=%s: %v", claims.UserID, spErr)
				if strings.Contains(spErr.Error(), "status 403") || strings.Contains(spErr.Error(), "status 401") {
					database.Pool.Exec(r.Context(),
						`DELETE FROM user_accounts WHERE user_id = $1 AND service = 'spotify'`,
						claims.UserID,
					)
					respondJSON(w, http.StatusUnauthorized, map[string]string{
						"error":   "spotify_reauth_required",
						"message": "Spotify authorization expired or missing permissions. Reconnect to continue.",
					})
					return
				}
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("spotify analysis failed: %v", spErr))
				return
			}

			debug.Logf("Analyze: got %d top artists, %d top tracks, %d genre tags from Spotify",
				len(result.TopArtists), len(result.TopTracks), len(result.GenreCount))
			spGenres := spotifysvc.MapGenresToOurSystem(result.GenreCount, nameToID)
			debug.Logf("Analyze: mapped to %d our genres", len(spGenres))
			if len(spGenres) == 0 {
				debug.Logf("Analyze: genre map keys: %v", result.GenreCount)
			}
			if source == "combined" && ourGenres != nil {
				for id, w := range spGenres {
					if existing, ok := ourGenres[id]; ok {
						ourGenres[id] = (existing + w) / 2
					} else {
						ourGenres[id] = w * 0.5
					}
				}
			} else {
				ourGenres = spGenres
			}

			spArtists := make([]string, len(result.TopArtists))
			for i, a := range result.TopArtists {
				spArtists[i] = a.Name
			}
			if source == "combined" {
				seen := map[string]bool{}
				for _, a := range topArtists {
					seen[a] = true
				}
				for _, a := range spArtists {
					if !seen[a] {
						topArtists = append(topArtists, a)
						seen[a] = true
					}
				}
			} else {
				topArtists = spArtists
			}

		for _, t := range result.TopTracks {
			topTracks = append(topTracks, models.Track{
				Title:           t.Title,
				Artist:          t.Artist,
				Album:           t.Album,
				AlbumArtURL:     t.AlbumArt,
				PreviewURL:      t.Preview,
				SpotifyURI:      t.URI,
				DurationMs:      t.Duration,
				ReleaseYear:     t.ReleaseYear,
				Danceability:    t.Danceability,
				Energy:          t.Energy,
				Valence:         t.Valence,
				Acousticness:    t.Acousticness,
				Instrumentalness: t.Instrumentalness,
				Speechiness:     t.Speechiness,
				Tempo:           t.Tempo,
			})
		}

			if result.AudioFeatures != nil {
				debug.Logf("Analyze: got audio features (danceability=%.2f energy=%.2f valence=%.2f)",
					result.AudioFeatures.Danceability, result.AudioFeatures.Energy, result.AudioFeatures.Valence)
				audioFeatures = &models.AudioFeatures{
					Danceability:     result.AudioFeatures.Danceability,
					Energy:           result.AudioFeatures.Energy,
					Valence:          result.AudioFeatures.Valence,
					Acousticness:     result.AudioFeatures.Acousticness,
					Instrumentalness: result.AudioFeatures.Instrumentalness,
					Speechiness:      result.AudioFeatures.Speechiness,
					Liveness:         result.AudioFeatures.Liveness,
					Tempo:            result.AudioFeatures.Tempo,
				}
			} else {
				debug.Log("Analyze: no audio features returned from Spotify")
			}
		} else if source == "spotify" {
			respondError(w, http.StatusBadRequest, "no linked Spotify account. connect Spotify in the sidebar first.")
			return
		}
	}

	if topArtists == nil {
		topArtists = []string{}
	}

	if ourGenres == nil {
		ourGenres = make(map[string]float64)
		debug.Log("Analyze: ourGenres was nil, initialized to empty map")
	}

	tx, err := database.Pool.Begin(r.Context())
	if err != nil {
		debug.Logf("Analyze: failed to begin transaction: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback(r.Context())

	tx.Exec(r.Context(), `DELETE FROM taste_genres WHERE user_id = $1`, claims.UserID)
	genreCount := 0
	for genreID, weight := range ourGenres {
		tx.Exec(r.Context(),
			`INSERT INTO taste_genres (user_id, genre_id, weight) VALUES ($1, $2, $3)`,
			claims.UserID, genreID, weight,
		)
		genreCount++
	}
	debug.Logf("Analyze: saved %d genre weights for user=%s", genreCount, claims.UserID)

	tx.Exec(r.Context(),
		`INSERT INTO taste_profiles (user_id, top_artists, top_tracks, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET top_artists = $2, top_tracks = $3, updated_at = NOW()`,
		claims.UserID, topArtists, topTracks,
	)

	if err := tx.Commit(r.Context()); err != nil {
		debug.Logf("Analyze: commit failed: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to save taste analysis")
		return
	}

	debug.Log("Analyze: transaction committed successfully")

	profile := models.TasteProfile{
		UserID:        claims.UserID,
		Genres:        []models.GenreWeight{},
		TopArtists:    topArtists,
		TopTracks:     topTracks,
		AudioFeatures: audioFeatures,
	}
	for id, w := range ourGenres {
		profile.Genres = append(profile.Genres, models.GenreWeight{GenreID: id, Weight: w})
	}

	respondJSON(w, http.StatusOK, profile)
}

func (h *TasteHandler) Genres(w http.ResponseWriter, r *http.Request) {
	rows, err := database.Pool.Query(r.Context(),
		`SELECT id, name, description, color, x, y, parent_id FROM genres ORDER BY name`,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch genres")
		return
	}
	defer rows.Close()

	genres := []models.Genre{}
	for rows.Next() {
		var g models.Genre
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Color, &g.X, &g.Y, &g.ParentID); err != nil {
			continue
		}
		genres = append(genres, g)
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	respondJSON(w, http.StatusOK, genres)
}
