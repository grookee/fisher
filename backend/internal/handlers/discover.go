package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"sync"

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

	var tracks []spotifysvc.SearchResult
	seen := map[string]bool{}
	var mu sync.Mutex

	addUnique := func(list []spotifysvc.SearchResult) {
		mu.Lock()
		defer mu.Unlock()
		for _, t := range list {
			if !seen[t.ID] {
				seen[t.ID] = true
				tracks = append(tracks, t)
			}
		}
	}

	if genre != "" {
		t, err := svc.SearchTracks(genre, limit)
		if err != nil {
			debug.Logf("Discover: search for %q failed: %v", genre, err)
		}
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

		var wg sync.WaitGroup
		for _, gn := range genreNames {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				t, err := svc.SearchTracks(name, perGenre)
				if err != nil {
					debug.Logf("Discover: search for %q failed: %v", name, err)
					return
				}
				addUnique(t)
			}(gn)
		}
		wg.Wait()

		if len(tracks) < limit {
			needed := limit - len(tracks)
			seed := discoverSeeds[rand.Intn(len(discoverSeeds))]
			debug.Logf("Discover: need %d more tracks, trying seed %q", needed, seed)
			t, err := svc.SearchTracks(seed, needed)
			if err != nil {
				debug.Logf("Discover: seed search for %q failed: %v", seed, err)
			}
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
