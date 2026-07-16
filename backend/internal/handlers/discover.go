package handlers

import (
	"math/rand"
	"net/http"
	"strconv"

	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/debug"
	spotifysvc "github.com/fisher/backend/internal/services/spotify"
)

type DiscoverHandler struct{}

func NewDiscoverHandler() *DiscoverHandler {
	return &DiscoverHandler{}
}

var discoverSeeds = []string{
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
	"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	"new", "top", "best", "fresh", "underground", "chill", "vibes",
	"discover", "mood", "lofi", "energy",
}

func (h *DiscoverHandler) Discover(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("genre")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	svc := &spotifysvc.Service{}

	searchGenre := func(name string, lmt int) []spotifysvc.SearchResult {
		t, err := svc.SearchTracks(name, lmt)
		if err != nil {
			debug.Logf("Discover: search for %q failed: %v", name, err)
			return nil
		}
		return t
	}

	var tracks []spotifysvc.SearchResult
	seen := map[string]bool{}

	addUnique := func(list []spotifysvc.SearchResult) {
		for _, t := range list {
			if !seen[t.ID] {
				seen[t.ID] = true
				tracks = append(tracks, t)
			}
		}
	}

	if genre != "" {
		t := searchGenre(genre, limit)
		addUnique(t)
	} else {
		var genreNames []string
		rows, err := database.Pool.Query(r.Context(), `SELECT name FROM genres ORDER BY RANDOM() LIMIT 3`)
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

		debug.Logf("Discover: picking from genres %v", genreNames)

		perGenre := limit / len(genreNames)
		if perGenre < 5 {
			perGenre = 5
		}
		for _, gn := range genreNames {
			t := searchGenre(gn, perGenre)
			if len(t) > 0 {
				addUnique(t)
				if len(tracks) >= limit {
					break
				}
			}
		}

		if len(tracks) < limit {
			needed := limit - len(tracks)
			seed := discoverSeeds[rand.Intn(len(discoverSeeds))]
			debug.Logf("Discover: need %d more tracks, trying seed %q", needed, seed)
			t := searchGenre(seed, needed)
			addUnique(t)
		}
	}

	debug.Logf("Discover: returning %d tracks", len(tracks))
	if tracks == nil {
		tracks = []spotifysvc.SearchResult{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tracks": tracks,
		"genre":  genre,
	})
}
