package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fisher/backend/internal/auth"
	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/debug"
	"github.com/fisher/backend/internal/services/ranking"
	spotifysvc "github.com/fisher/backend/internal/services/spotify"
)

type ExploreHandler struct{}

func NewExploreHandler() *ExploreHandler {
	return &ExploreHandler{}
}

var decades = []struct {
	Label string
	Start int
	End   int
}{
	{"1950s", 1950, 1959},
	{"1960s", 1960, 1969},
	{"1970s", 1970, 1979},
	{"1980s", 1980, 1989},
	{"1990s", 1990, 1999},
	{"2000s", 2000, 2009},
	{"2010s", 2010, 2019},
	{"2020s", 2020, 2029},
}

var decadeLabelToRange = func() map[string][2]int {
	m := make(map[string][2]int)
	for _, d := range decades {
		m[d.Label] = [2]int{d.Start, d.End}
	}
	return m
}()



func (h *ExploreHandler) Feed(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 30
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	svc := &spotifysvc.Service{}
	spotifyToken := spotifyTokenForUser(r.Context(), userIDFromContext(r.Context()))

	var genreNames []string
	rows, err := database.Pool.Query(r.Context(), `SELECT name FROM genres ORDER BY RANDOM() LIMIT 6`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			rows.Scan(&name)
			genreNames = append(genreNames, name)
		}
	}
	if len(genreNames) == 0 {
		genreNames = []string{"rock", "pop", "electronic"}
	}

	debug.Logf("ExploreFeed: genres %v", genreNames)

	var mu sync.Mutex
	var wg sync.WaitGroup
	var tracks []spotifysvc.SearchResult
	seen := map[string]bool{}

	for _, gn := range genreNames {
		wg.Add(1)
		go func(genreName string) {
			defer wg.Done()
			perGenre := limit / len(genreNames)
			if perGenre < 3 {
				perGenre = 3
			}
			t, err := svc.SearchTracksWithAccessToken(genreName, perGenre+rand.Intn(5), spotifyToken)
			if err == nil {
				mu.Lock()
				for _, tr := range t {
					tr.Genre = genreName
					if !seen[tr.ID] {
						seen[tr.ID] = true
						tracks = append(tracks, tr)
					}
				}
				mu.Unlock()
			}
		}(gn)
	}

	wg.Wait()

	shuffle := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffle.Shuffle(len(tracks), func(i, j int) {
		tracks[i], tracks[j] = tracks[j], tracks[i]
	})

	if len(tracks) > limit {
		tracks = tracks[:limit]
	}

	if tracks == nil {
		tracks = []spotifysvc.SearchResult{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tracks": tracks,
		"genres": genreNames,
	})
}

func (h *ExploreHandler) MissedGenres(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())

	var topGenreIDs []string
	rows, err := database.Pool.Query(r.Context(),
		`SELECT genre_id, weight FROM taste_genres WHERE user_id = $1 ORDER BY weight DESC LIMIT 5`,
		claims.UserID,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var gid string
			var weight float64
			rows.Scan(&gid, &weight)
			if weight > 0.3 {
				topGenreIDs = append(topGenreIDs, gid)
			}
		}
	}

	if len(topGenreIDs) == 0 {
		debug.Logf("MissedGenres: no taste_genres for user %s, using random fallback", claims.UserID)
		rows, err := database.Pool.Query(r.Context(),
			`SELECT id FROM genres ORDER BY RANDOM() LIMIT 5`,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var gid string
				rows.Scan(&gid)
				topGenreIDs = append(topGenreIDs, gid)
			}
		}
	}

	if len(topGenreIDs) == 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"genres": []interface{}{},
			"note":   "no genres found in the database",
		})
		return
	}

	type GenreResult struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Color       string  `json:"color"`
		X           float64 `json:"x"`
		Y           float64 `json:"y"`
		ParentID    *string `json:"parent_id"`
		Reason      string  `json:"reason"`
	}

	parents := make([]string, 0, len(topGenreIDs))
	for _, gid := range topGenreIDs {
		var parentID *string
		err := database.Pool.QueryRow(r.Context(),
			`SELECT parent_id FROM genres WHERE id = $1`, gid,
		).Scan(&parentID)
		if err != nil {
			continue
		}
		if parentID != nil {
			parents = append(parents, *parentID)
		} else {
			parents = append(parents, gid)
		}
	}

	parents = uniqueStrings(parents)
	if len(parents) == 0 {
		parents = topGenreIDs
	}

	debug.Logf("MissedGenres: using parent genres: %v", parents)

	var matchedGenres []GenreResult

	// Prefer the genre adjacency graph (built by the taxonomy ingestion
	// pipeline from Wikidata + Last.fm) when it's available: it surfaces
	// genuinely *related* genres - subgenre/influence relationships and real
	// co-listening data - rather than just "other children of the same
	// hand-picked parent category".
	graphPlaceholders := make([]string, len(topGenreIDs))
	graphArgs := make([]interface{}, len(topGenreIDs)+1)
	graphArgs[0] = claims.UserID
	for i, gid := range topGenreIDs {
		graphPlaceholders[i] = "$" + strconv.Itoa(i+2)
		graphArgs[i+1] = gid
	}

	graphQuery := `SELECT g.id, g.name, g.description, g.color, g.x, g.y, g.parent_id, AVG(gr.weight) AS avg_weight
		FROM genre_relations gr
		JOIN genres g ON g.id = gr.related_genre_id
		WHERE gr.genre_id IN (` + strings.Join(graphPlaceholders, ",") + `)
		  AND g.id NOT IN (SELECT genre_id FROM taste_genres WHERE user_id = $1)
		GROUP BY g.id, g.name, g.description, g.color, g.x, g.y, g.parent_id
		ORDER BY avg_weight DESC
		LIMIT 10`

	if graphRows, err := database.Pool.Query(r.Context(), graphQuery, graphArgs...); err == nil {
		defer graphRows.Close()
		for graphRows.Next() {
			var g GenreResult
			var avgWeight float64
			if err := graphRows.Scan(&g.ID, &g.Name, &g.Description, &g.Color, &g.X, &g.Y, &g.ParentID, &avgWeight); err != nil {
				continue
			}
			g.Reason = "adjacent to your taste"
			matchedGenres = append(matchedGenres, g)
		}
	} else {
		debug.Logf("MissedGenres: graph query failed (taxonomy may not be ingested yet): %v", err)
	}

	if len(matchedGenres) == 0 {
		placeholders := make([]string, len(parents))
		args := make([]interface{}, len(parents)+1)
		args[0] = claims.UserID
		for i, pid := range parents {
			placeholders[i] = "$" + strconv.Itoa(i+2)
			args[i+1] = pid
		}

		query := `SELECT g.id, g.name, g.description, g.color, g.x, g.y, g.parent_id
			FROM genres g
			WHERE g.parent_id IN (` + strings.Join(placeholders, ",") + `)
			AND g.name NOT IN (
				SELECT g2.name FROM taste_genres tg
				JOIN genres g2 ON g2.id = tg.genre_id
				WHERE tg.user_id = $1
			)
			ORDER BY RANDOM()
			LIMIT 10`

		rows, err := database.Pool.Query(r.Context(), query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var g GenreResult
				if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Color, &g.X, &g.Y, &g.ParentID); err != nil {
					continue
				}
				g.Reason = "related to your taste"
				matchedGenres = append(matchedGenres, g)
			}
		} else {
			debug.Logf("MissedGenres: parent-sibling query failed: %v", err)
		}
	}

	if len(matchedGenres) == 0 {
		debug.Logf("MissedGenres: query returned no results, trying random genres")
		fallbackRows, err := database.Pool.Query(r.Context(),
			`SELECT id, name, description, color, x, y, parent_id FROM genres ORDER BY RANDOM() LIMIT 6`,
		)
		if err == nil {
			defer fallbackRows.Close()
			for fallbackRows.Next() {
				var g GenreResult
				if err := fallbackRows.Scan(&g.ID, &g.Name, &g.Description, &g.Color, &g.X, &g.Y, &g.ParentID); err != nil {
					continue
				}
				g.Reason = "check this out"
				matchedGenres = append(matchedGenres, g)
			}
		}
	}

	if matchedGenres == nil {
		matchedGenres = []GenreResult{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"genres": matchedGenres,
	})
}

func (h *ExploreHandler) TimeMachine(w http.ResponseWriter, r *http.Request) {
	label := r.URL.Query().Get("decade")
	if label == "" {
		label = "1980s"
	}

	yearRange, ok := decadeLabelToRange[label]
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid decade. use: 1950s, 1960s, 1970s, 1980s, 1990s, 2000s, 2010s, 2020s")
		return
	}
	startYear, endYear := yearRange[0], yearRange[1]

	mood := r.URL.Query().Get("mood")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	svc := &spotifysvc.Service{}
	spotifyToken := spotifyTokenForUser(r.Context(), userIDFromContext(r.Context()))
	seen := map[string]bool{}
	collected := make([]spotifysvc.SearchResult, 0, limit)

	// --- local catalog: tracks already ingested with accurate release_year ---
	localRows, err := database.Pool.Query(r.Context(),
		`SELECT t.id, t.title, t.artist, t.album, t.album_art_url, t.duration_ms,
		        t.preview_url, t.release_year
		 FROM tracks t
		 WHERE t.release_year BETWEEN $1 AND $2
		   AND t.preview_url IS NOT NULL AND t.preview_url <> ''
		 ORDER BY RANDOM()
		 LIMIT $3`,
		startYear, endYear, limit,
	)
	if err == nil {
		defer localRows.Close()
		for localRows.Next() {
			var id, title, artist, album, albumArt, preview string
			var duration, releaseYear int
			if err := localRows.Scan(&id, &title, &artist, &album, &albumArt, &duration, &preview, &releaseYear); err != nil {
				continue
			}
			seen[id] = true
			collected = append(collected, spotifysvc.SearchResult{
				ID:          id,
				Title:       title,
				Artist:      artist,
				Album:       album,
				AlbumArt:    albumArt,
				Duration:    duration,
				SpotifyID:   "",
				Preview:     preview,
				ReleaseYear: releaseYear,
			})
		}
	}
	debug.Logf("TimeMachine: local catalog returned %d tracks for %s", len(collected), label)

	// --- supplement with Spotify year: filter (no free text) ---
	if len(collected) < limit {
		remaining := limit - len(collected)
		yearToken := fmt.Sprintf("year:%d-%d", startYear, endYear)

		// Multiple queries with offset variation to bypass the 10-result cap
		offsets := []int{0, 10, 20, 30}
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, off := range offsets {
			if len(collected) >= limit {
				break
			}
			wg.Add(1)
			go func(offset int) {
				defer wg.Done()
				q := yearToken
				if mood != "" {
					q = yearToken + " " + mood
				}
				debug.Logf("TimeMachine: spotify search %q (offset=%d)", q, offset)
				results, err := svc.SearchTracksWithAccessTokenOffset(q, remaining+5, offset, spotifyToken)
				if err != nil {
					debug.Logf("TimeMachine: spotify search failed: %v", err)
					return
				}
				mu.Lock()
				for _, t := range results {
					if seen[t.ID] {
						continue
					}
					if t.ReleaseYear < startYear || t.ReleaseYear > endYear {
						continue
					}
					seen[t.ID] = true
					collected = append(collected, t)
				}
				mu.Unlock()
			}(off)
		}
		wg.Wait()
	}

	if len(collected) > limit {
		collected = collected[:limit]
	}

	if collected == nil {
		collected = []spotifysvc.SearchResult{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tracks": collected,
		"decade": label,
		"mood":   mood,
	})
}

func userIDFromContext(ctx context.Context) string {
	claims := auth.GetUser(ctx)
	if claims == nil {
		return ""
	}
	return claims.UserID
}

func spotifyTokenForUser(ctx context.Context, userID string) string {
	if userID == "" {
		return ""
	}

	var accessToken, refreshToken string
	var tokenExpiry *time.Time
	err := database.Pool.QueryRow(ctx,
		`SELECT access_token, refresh_token, token_expiry
		 FROM user_accounts
		 WHERE user_id = $1 AND service = 'spotify' LIMIT 1`,
		userID,
	).Scan(&accessToken, &refreshToken, &tokenExpiry)
	if err != nil {
		return ""
	}

	if accessToken != "" && (tokenExpiry == nil || time.Now().Before(tokenExpiry.Add(-time.Minute))) {
		return accessToken
	}
	if refreshToken == "" {
		return accessToken
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return accessToken
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		debug.Logf("spotifyTokenForUser: refresh request failed for user=%s: %v", userID, err)
		return accessToken
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		debug.Logf("spotifyTokenForUser: refresh status=%d for user=%s", resp.StatusCode, userID)
		return accessToken
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		debug.Logf("spotifyTokenForUser: refresh decode failed for user=%s: %v", userID, err)
		return accessToken
	}
	if payload.AccessToken == "" {
		return accessToken
	}

	newRefreshToken := refreshToken
	if payload.RefreshToken != "" {
		newRefreshToken = payload.RefreshToken
	}
	newExpiry := time.Now().Add(1 * time.Hour)
	if payload.ExpiresIn > 0 {
		newExpiry = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}

	_, _ = database.Pool.Exec(ctx,
		`UPDATE user_accounts
		 SET access_token = $2,
		     refresh_token = $3,
		     token_expiry = $4,
		     updated_at = NOW()
		 WHERE user_id = $1 AND service = 'spotify'`,
		userID, payload.AccessToken, newRefreshToken, newExpiry,
	)

	return payload.AccessToken
}

func uniqueStrings(s []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// RelatedGenres returns the weighted neighbors of a genre in the adjacency
// graph built by the taxonomy ingestion pipeline (Wikidata subgenre/influence
// relations + Last.fm co-occurrence). Falls back to an empty list if the
// graph hasn't been ingested yet.
func (h *ExploreHandler) RelatedGenres(w http.ResponseWriter, r *http.Request) {
	genreID := r.URL.Query().Get("id")
	if genreID == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 15
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	rows, err := database.Pool.Query(r.Context(),
		`SELECT g.id, g.name, g.description, g.color, g.x, g.y, g.parent_id, gr.relation_type, gr.weight
		 FROM genre_relations gr
		 JOIN genres g ON g.id = gr.related_genre_id
		 WHERE gr.genre_id = $1
		 ORDER BY gr.weight DESC
		 LIMIT $2`,
		genreID, limit,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch related genres")
		return
	}
	defer rows.Close()

	type RelatedGenre struct {
		ID           string  `json:"id"`
		Name         string  `json:"name"`
		Description  string  `json:"description"`
		Color        string  `json:"color"`
		X            float64 `json:"x"`
		Y            float64 `json:"y"`
		ParentID     *string `json:"parent_id"`
		RelationType string  `json:"relation_type"`
		Weight       float64 `json:"weight"`
	}

	related := []RelatedGenre{}
	for rows.Next() {
		var g RelatedGenre
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Color, &g.X, &g.Y, &g.ParentID, &g.RelationType, &g.Weight); err != nil {
			continue
		}
		related = append(related, g)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"genre_id": genreID,
		"related":  related,
	})
}

// Recommendations returns an "obscure but related" genre recommendation
// list computed by the ranking package: candidates from the genre adjacency
// graph, scored by taste similarity, empirical scene affinity, regional
// novelty, obscurity, and how novel the genre still is within the user's
// friend group. Falls back to an empty list if the user has no taste
// profile yet or the taxonomy graph hasn't been ingested.
func (h *ExploreHandler) Recommendations(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())

	limit := 12
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 30 {
		limit = l
	}

	var ignoredIDs []string
	var ignoredSetting string
	database.Pool.QueryRow(r.Context(),
		`SELECT value FROM user_settings WHERE user_id = $1 AND key = 'ignored_genres'`,
		claims.UserID,
	).Scan(&ignoredSetting)
	if ignoredSetting != "" {
		json.Unmarshal([]byte(ignoredSetting), &ignoredIDs)
	}

	var obscureOnly bool
	var obscureSetting string
	database.Pool.QueryRow(r.Context(),
		`SELECT value FROM user_settings WHERE user_id = $1 AND key = 'obscure_only'`,
		claims.UserID,
	).Scan(&obscureSetting)
	if obscureSetting == "true" || obscureSetting == "1" {
		obscureOnly = true
	}

	scores, deepCuts, err := ranking.RecommendGenres(r.Context(), database.Pool, claims.UserID, limit, ignoredIDs, obscureOnly)
	if err != nil {
		debug.Logf("Recommendations: ranking failed: %v", err)
	}
	if scores == nil {
		scores = []ranking.GenreScore{}
	}
	if deepCuts == nil {
		deepCuts = []ranking.GenreScore{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recommendations": scores,
		"deep_cuts":       deepCuts,
	})
}

func (h *ExploreHandler) Genre(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	debug.Logf("Genre: searching %q", name)

	// --- primary: local catalog via track_genres ---
	var tracks []spotifysvc.SearchResult
	var genreID string
	err := database.Pool.QueryRow(r.Context(),
		`SELECT id FROM genres WHERE LOWER(name) = LOWER($1) LIMIT 1`, name,
	).Scan(&genreID)
	if err == nil {
		rows, err := database.Pool.Query(r.Context(),
			`SELECT t.id, t.title, t.artist, t.album, t.album_art_url, t.duration_ms,
			        t.preview_url, t.release_year
			 FROM tracks t
			 JOIN track_genres tg ON tg.track_id = t.id
			 WHERE tg.genre_id = $1
			   AND t.preview_url IS NOT NULL AND t.preview_url <> ''
			 ORDER BY RANDOM()
			 LIMIT $2`,
			genreID, limit,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, title, artist, album, albumArt, preview string
				var duration, releaseYear int
				if err := rows.Scan(&id, &title, &artist, &album, &albumArt, &duration, &preview, &releaseYear); err != nil {
					continue
				}
				tracks = append(tracks, spotifysvc.SearchResult{
					ID:          id,
					Title:       title,
					Artist:      artist,
					Album:       album,
					AlbumArt:    albumArt,
					Duration:    duration,
					SpotifyID:   "",
					Preview:     preview,
					ReleaseYear: releaseYear,
				})
			}
		}
	}
	debug.Logf("Genre: local catalog returned %d tracks for %q", len(tracks), name)

	// --- supplement: Spotify genre: filter, then fallback to text search ---
	if len(tracks) < limit {
		svc := &spotifysvc.Service{}
		spotifyToken := spotifyTokenForUser(r.Context(), userIDFromContext(r.Context()))
		remaining := limit - len(tracks)
		seen := map[string]bool{}
		for _, t := range tracks {
			seen[t.ID] = true
		}

		// try genre: field filter first
		genreQuery := fmt.Sprintf("genre:\"%s\"", name)
		spotifyTracks, err := svc.SearchTracksWithAccessToken(genreQuery, remaining+5, spotifyToken)
		if err == nil {
			for _, t := range spotifyTracks {
				if seen[t.ID] {
					continue
				}
				seen[t.ID] = true
				tracks = append(tracks, t)
			}
		}

		// if still sparse, fall back to plain text search
		if len(tracks) < limit {
			textTracks, err := svc.SearchTracksWithAccessToken(name, remaining+5, spotifyToken)
			if err == nil {
				for _, t := range textTracks {
					if seen[t.ID] {
						continue
					}
					seen[t.ID] = true
					tracks = append(tracks, t)
				}
			}
		}
	}

	if len(tracks) > limit {
		tracks = tracks[:limit]
	}

	if tracks == nil {
		tracks = []spotifysvc.SearchResult{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tracks": tracks,
		"genre":  name,
	})
}
