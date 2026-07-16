package deezer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://api.deezer.com"

var client = &http.Client{Timeout: 10 * time.Second}

// ---------- public types for ingestion ----------

type Artist struct {
	ID   int64
	Name string
}

type TopTrack struct {
	ID          int64
	Title       string
	AlbumName   string
	AlbumArtURL string
	DurationSec int
	PreviewURL  string
	DeezerID    string
}

// ---------- API response shapes ----------

type searchArtistResponse struct {
	Data []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

type topTracksResponse struct {
	Data []struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Duration int    `json:"duration"`
		Preview  string `json:"preview"`
		Album    struct {
			Title   string `json:"title"`
			CoverXL string `json:"cover_xl"`
		} `json:"album"`
	} `json:"data"`
}

// ---------- public functions ----------

// SearchArtist returns the best-match Deezer artist for the given name, or a zero value if not found.
func SearchArtist(name string) (Artist, error) {
	u := fmt.Sprintf("%s/search/artist?q=%s&limit=5", baseURL, url.QueryEscape(name))
	resp, err := client.Get(u)
	if err != nil {
		return Artist{}, fmt.Errorf("deezer search %q: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Artist{}, fmt.Errorf("deezer search %q: status %d: %s", name, resp.StatusCode, string(body))
	}
	var sr searchArtistResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return Artist{}, fmt.Errorf("deezer search decode %q: %w", name, err)
	}
	if len(sr.Data) == 0 {
		return Artist{}, nil
	}

	// exact or best match
	normTarget := normalize(name)
	for _, a := range sr.Data {
		if normalize(a.Name) == normTarget {
			return Artist{ID: a.ID, Name: a.Name}, nil
		}
	}
	// fallback: first result
	return Artist{ID: sr.Data[0].ID, Name: sr.Data[0].Name}, nil
}

// GetArtistTopTracks returns up to `limit` top tracks for the given Deezer artist ID.
func GetArtistTopTracks(artistID int64, limit int) ([]TopTrack, error) {
	if limit <= 0 {
		limit = 10
	}
	u := fmt.Sprintf("%s/artist/%d/top?limit=%d&index=0", baseURL, artistID, limit)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("deezer top tracks artist %d: %w", artistID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deezer top tracks artist %d: status %d: %s", artistID, resp.StatusCode, string(body))
	}
	var tr topTracksResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("deezer top tracks decode %d: %w", artistID, err)
	}
	out := make([]TopTrack, 0, len(tr.Data))
	for _, t := range tr.Data {
		out = append(out, TopTrack{
			ID:          t.ID,
			Title:       t.Title,
			AlbumName:   t.Album.Title,
			AlbumArtURL: t.Album.CoverXL,
			DurationSec: t.Duration,
			PreviewURL:  t.Preview,
			DeezerID:    fmt.Sprintf("%d", t.ID),
		})
	}
	return out, nil
}

// ---------- preview-URL resolution (existing use, kept for backward compat) ----------

type TrackRequest struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

type TrackResult struct {
	ID         string `json:"id"`
	PreviewURL string `json:"preview_url"`
}

type searchTrackResponse struct {
	Data []struct {
		Preview string `json:"preview"`
		Title   string `json:"title"`
		Artist  struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"data"`
}

// ResolvePreviews looks up 30-second preview URLs on Deezer for the given tracks (concurrency-capped).
func ResolvePreviews(tracks []TrackRequest) []TrackResult {
	results := make([]TrackResult, len(tracks))
	for i, t := range tracks {
		results[i] = TrackResult{ID: t.ID}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for i, t := range tracks {
		if t.Title == "" || t.Artist == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, track TrackRequest) {
			defer wg.Done()
			defer func() { <-sem }()

			preview := searchDeezerPreview(track.Artist, track.Title)
			if preview != "" {
				mu.Lock()
				results[idx].PreviewURL = preview
				mu.Unlock()
			}
		}(i, t)
	}
	wg.Wait()
	return results
}

func searchDeezerPreview(artist, title string) string {
	query := fmt.Sprintf("%s %s", artist, title)
	u := baseURL + "/search?q=" + url.QueryEscape(query)

	resp, err := client.Get(u)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var sr searchTrackResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return ""
	}

	normArtist := normalize(artist)
	normTitle := normalize(title)

	for _, d := range sr.Data {
		if normalize(d.Artist.Name) == normArtist && normalize(d.Title) == normTitle {
			return d.Preview
		}
	}
	for _, d := range sr.Data {
		da := normalize(d.Artist.Name)
		dt := normalize(d.Title)
		if strings.Contains(da, normArtist) || strings.Contains(normArtist, da) {
			if strings.Contains(dt, normTitle) || strings.Contains(normTitle, dt) {
				return d.Preview
			}
		}
	}
	for _, d := range sr.Data {
		da := normalize(d.Artist.Name)
		if strings.Contains(da, normArtist) || strings.Contains(normArtist, da) {
			return d.Preview
		}
	}
	if len(sr.Data) > 0 {
		return sr.Data[0].Preview
	}
	return ""
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	for _, strip := range []string{"feat.", "ft.", "featuring", "remix", "radio edit", "single version", "album version"} {
		s = strings.ReplaceAll(s, strip, "")
	}
	s = strings.NewReplacer("(", "", ")", "", "[", "", "]", "", "-", " ", "&", " ").Replace(s)
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
