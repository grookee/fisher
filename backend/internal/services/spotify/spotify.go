package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/fisher/backend/internal/debug"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var (
	Client *spotify.Client
	Auth   *spotifyauth.Authenticator
	Config *clientcredentials.Config

	searchCache   = make(map[string]searchCacheEntry)
	searchCacheMu sync.RWMutex
)

type searchCacheEntry struct {
	results   []SearchResult
	expiresAt time.Time
}

const searchCacheTTL = 5 * time.Minute

func searchCacheKey(query string, limit, offset int) string {
	return fmt.Sprintf("%s|%d|%d", query, limit, offset)
}

func getCachedResults(query string, limit, offset int) ([]SearchResult, bool) {
	key := searchCacheKey(query, limit, offset)
	searchCacheMu.RLock()
	defer searchCacheMu.RUnlock()
	entry, ok := searchCache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.results, true
}

func setCachedResults(query string, limit, offset int, results []SearchResult) {
	key := searchCacheKey(query, limit, offset)
	searchCacheMu.Lock()
	defer searchCacheMu.Unlock()
	searchCache[key] = searchCacheEntry{
		results:   results,
		expiresAt: time.Now().Add(searchCacheTTL),
	}
}

type Service struct{}

func New() *Service {
	return &Service{}
}

type retryTransport struct {
	base      http.RoundTripper
	maxWait   time.Duration
 consecutive int
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for {
		resp, err := t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			wait := 10 * time.Second
			if retryAfter != "" {
				if secs, parseErr := strconv.Atoi(strings.TrimSpace(retryAfter)); parseErr == nil && secs > 0 {
					wait = time.Duration(secs) * time.Second
				}
			}
			t.consecutive++
			if wait > t.maxWait {
				log.Printf("[spotify] 429 rate-limited, Retry-After=%s exceeds max %s — aborting (url=%s, consecutive=%d)",
					wait.Round(time.Second), t.maxWait, req.URL.Path, t.consecutive)
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("spotify: rate limited (Retry-After=%s exceeds max %s)", wait, t.maxWait)
			}
			log.Printf("[spotify] 429 rate-limited, waiting %s (url=%s, consecutive=%d)", wait.Round(time.Second), req.URL.Path, t.consecutive)
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			time.Sleep(wait)
			continue
		}
		t.consecutive = 0
		return resp, nil
	}
}

func Init() {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		fmt.Println("warning: Spotify credentials not configured")
		return
	}

	Config = &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}

	Auth = spotifyauth.New(spotifyauth.WithClientID(clientID), spotifyauth.WithClientSecret(clientSecret))

	token, err := Config.Token(context.Background())
	if err != nil {
		fmt.Printf("warning: failed to get spotify token: %v\n", err)
		return
	}

	httpClient := spotifyauth.New().Client(context.Background(), token)
	httpClient.Transport = &retryTransport{base: httpClient.Transport, maxWait: 60 * time.Second}
	Client = spotify.New(httpClient)
}

func (s *Service) SearchTracks(query string, limit int) ([]SearchResult, error) {
	return s.searchTracks(query, limit, "")
}

func (s *Service) SearchTracksWithAccessToken(query string, limit int, accessToken string) ([]SearchResult, error) {
	return s.searchTracks(query, limit, accessToken)
}

func (s *Service) SearchTracksWithAccessTokenOffset(query string, limit, offset int, accessToken string) ([]SearchResult, error) {
	limit = normalizeSearchLimit(limit)
	if offset < 0 {
		offset = 0
	}
	if accessToken != "" {
		t, err := s.searchTracksViaUserTokenOffset(accessToken, query, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("user token search failed: %w", err)
		}
		return t, nil
	}
	return s.searchTracksViaClientCredentialsOffset(query, limit, offset)
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 10 {
		return 10
	}
	return limit
}

func (s *Service) searchTracks(query string, limit int, accessToken string) ([]SearchResult, error) {
	limit = normalizeSearchLimit(limit)
	tracks := []SearchResult{}

	if accessToken != "" {
		t, err := s.searchTracksViaUserToken(accessToken, query, limit)
		if err != nil {
			debug.Logf("SearchTracks: user-token search failed: %v", err)
			return nil, fmt.Errorf("user token search failed: %w", err)
		}
		tracks = t
	} else {
		t, err := s.searchTracksViaClientCredentials(query, limit)
		if err != nil {
			return nil, err
		}
		tracks = t
	}

	if len(tracks) > 0 {
		enrichAudioFeaturesFromSoundStat(tracks)
		if !hasAnyAudioFeatures(tracks) {
			enrichAudioFeaturesFromReccobeats(tracks)
		}
	}

	return tracks, nil
}

func (s *Service) searchTracksViaClientCredentials(query string, limit int) ([]SearchResult, error) {
	limit = normalizeSearchLimit(limit)

	if cached, ok := getCachedResults(query, limit, 0); ok {
		return cached, nil
	}

	if Client == nil {
		return nil, fmt.Errorf("spotify client not initialized")
	}

	offset := rand.Intn(10)
	results, err := Client.Search(context.Background(), query, spotify.SearchTypeTrack, spotify.Limit(limit), spotify.Offset(offset))
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	tracks := make([]SearchResult, 0)
	if results.Tracks != nil {
		for _, t := range results.Tracks.Tracks {
			tracks = append(tracks, SearchResult{
				ID:          t.ID.String(),
				Title:       t.Name,
				Artist:      joinArtistNames(t.Artists),
				Album:       t.Album.Name,
				AlbumArt:    albumArtURL(t.Album.Images),
				Duration:    int(t.Duration),
				SpotifyID:   t.ID.String(),
				Preview:     t.PreviewURL,
				ReleaseYear: extractYear(t.Album.ReleaseDate),
			})
		}
	}

	setCachedResults(query, limit, 0, tracks)
	return tracks, nil
}

func (s *Service) searchTracksViaClientCredentialsOffset(query string, limit, offset int) ([]SearchResult, error) {
	limit = normalizeSearchLimit(limit)

	if cached, ok := getCachedResults(query, limit, offset); ok {
		return cached, nil
	}

	if Client == nil {
		return nil, fmt.Errorf("spotify client not initialized")
	}

	results, err := Client.Search(context.Background(), query, spotify.SearchTypeTrack, spotify.Limit(limit), spotify.Offset(offset))
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	tracks := make([]SearchResult, 0)
	if results.Tracks != nil {
		for _, t := range results.Tracks.Tracks {
			tracks = append(tracks, SearchResult{
				ID:          t.ID.String(),
				Title:       t.Name,
				Artist:      joinArtistNames(t.Artists),
				Album:       t.Album.Name,
				AlbumArt:    albumArtURL(t.Album.Images),
				Duration:    int(t.Duration),
				SpotifyID:   t.ID.String(),
				Preview:     t.PreviewURL,
				ReleaseYear: extractYear(t.Album.ReleaseDate),
			})
		}
	}

	setCachedResults(query, limit, offset, tracks)
	return tracks, nil
}

func (s *Service) searchTracksViaUserToken(accessToken, query string, limit int) ([]SearchResult, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("empty access token")
	}
	limit = normalizeSearchLimit(limit)

	offset := rand.Intn(10)
	u := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=%d&offset=%d", url.QueryEscape(query), limit, offset)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("user-token search request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-token search status %d", resp.StatusCode)
	}

	var payload struct {
		Tracks struct {
			Items []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				PreviewURL string `json:"preview_url"`
				DurationMs int    `json:"duration_ms"`
				Artists    []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name        string `json:"name"`
					ReleaseDate string `json:"release_date"`
					Images      []struct {
						URL string `json:"url"`
					} `json:"images"`
				} `json:"album"`
			} `json:"items"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode user-token search: %w", err)
	}

	tracks := make([]SearchResult, 0, len(payload.Tracks.Items))
	for _, t := range payload.Tracks.Items {
		artistNames := make([]string, 0, len(t.Artists))
		for _, a := range t.Artists {
			artistNames = append(artistNames, a.Name)
		}
		albumArt := ""
		if len(t.Album.Images) > 0 {
			albumArt = t.Album.Images[0].URL
		}
		tracks = append(tracks, SearchResult{
			ID:          t.ID,
			Title:       t.Name,
			Artist:      strings.Join(artistNames, ", "),
			Album:       t.Album.Name,
			AlbumArt:    albumArt,
			Duration:    t.DurationMs,
			SpotifyID:   t.ID,
			Preview:     t.PreviewURL,
			ReleaseYear: extractYear(t.Album.ReleaseDate),
		})
	}
	return tracks, nil
}

func (s *Service) searchTracksViaUserTokenOffset(accessToken, query string, limit, offset int) ([]SearchResult, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("empty access token")
	}
	limit = normalizeSearchLimit(limit)

	u := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=%d&offset=%d", url.QueryEscape(query), limit, offset)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("user-token search request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-token search status %d", resp.StatusCode)
	}

	var payload struct {
		Tracks struct {
			Items []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				PreviewURL string `json:"preview_url"`
				DurationMs int    `json:"duration_ms"`
				Artists    []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name        string `json:"name"`
					ReleaseDate string `json:"release_date"`
					Images      []struct {
						URL string `json:"url"`
					} `json:"images"`
				} `json:"album"`
			} `json:"items"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode user-token search: %w", err)
	}

	tracks := make([]SearchResult, 0, len(payload.Tracks.Items))
	for _, t := range payload.Tracks.Items {
		artistNames := make([]string, 0, len(t.Artists))
		for _, a := range t.Artists {
			artistNames = append(artistNames, a.Name)
		}
		albumArt := ""
		if len(t.Album.Images) > 0 {
			albumArt = t.Album.Images[0].URL
		}
		tracks = append(tracks, SearchResult{
			ID:          t.ID,
			Title:       t.Name,
			Artist:      strings.Join(artistNames, ", "),
			Album:       t.Album.Name,
			AlbumArt:    albumArt,
			Duration:    t.DurationMs,
			SpotifyID:   t.ID,
			Preview:     t.PreviewURL,
			ReleaseYear: extractYear(t.Album.ReleaseDate),
		})
	}
	return tracks, nil
}

func extractYear(releaseDate string) int {
	if len(releaseDate) >= 4 {
		y := 0
		fmt.Sscanf(releaseDate[:4], "%d", &y)
		if y > 1900 && y < 2100 {
			return y
		}
	}
	return 0
}

func (s *Service) GetTrack(id string) (*SearchResult, error) {
	if cached, ok := getCachedResults("track:"+id, 1, 0); ok && len(cached) > 0 {
		return &cached[0], nil
	}

	if Client == nil {
		return nil, fmt.Errorf("spotify client not initialized")
	}

	track, err := Client.GetTrack(context.Background(), spotify.ID(id))
	if err != nil {
		return nil, fmt.Errorf("get track failed: %w", err)
	}

	result := &SearchResult{
		ID:        track.ID.String(),
		Title:     track.Name,
		Artist:    joinArtistNames(track.Artists),
		Album:     track.Album.Name,
		AlbumArt:  albumArtURL(track.Album.Images),
		Duration:  int(track.Duration),
		SpotifyID: track.ID.String(),
		Preview:   track.PreviewURL,
	}

	setCachedResults("track:"+id, 1, 0, []SearchResult{*result})
	return result, nil
}

type SearchResult struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Artist           string  `json:"artist"`
	Album            string  `json:"album"`
	AlbumArt         string  `json:"album_art_url"`
	Duration         int     `json:"duration_ms"`
	SpotifyID        string  `json:"spotify_uri"`
	Preview          string  `json:"preview_url"`
	ReleaseYear      int     `json:"release_year"`
	Genre            string  `json:"genre,omitempty"`
	Danceability     float64 `json:"danceability"`
	Energy           float64 `json:"energy"`
	Valence          float64 `json:"valence"`
	Acousticness     float64 `json:"acousticness"`
	Instrumentalness float64 `json:"instrumentalness"`
	Speechiness      float64 `json:"speechiness"`
	Tempo            float64 `json:"tempo"`
}

type soundStatValue struct {
	Value float64 `json:"value"`
}

type soundStatTrackResponse struct {
	Features struct {
		Danceability     float64 `json:"danceability"`
		Energy           float64 `json:"energy"`
		Valence          float64 `json:"valence"`
		Acousticness     float64 `json:"acousticness"`
		Instrumentalness float64 `json:"instrumentalness"`
		Speechiness      float64 `json:"speechiness"`
		Tempo            float64 `json:"tempo"`
	} `json:"features"`
	AudioAnalysis struct {
		Danceability     soundStatValue `json:"danceability"`
		Energy           soundStatValue `json:"energy"`
		Valence          soundStatValue `json:"valence"`
		Acousticness     soundStatValue `json:"acousticness"`
		Instrumentalness soundStatValue `json:"instrumentalness"`
		Speechiness      soundStatValue `json:"speechiness"`
		Tempo            soundStatValue `json:"tempo"`
	} `json:"audio_analysis"`
}

func enrichAudioFeaturesFromSoundStat(tracks []SearchResult) {
	apiKey := os.Getenv("SOUNDSTAT_KEY")
	if apiKey == "" {
		return
	}

	client := &http.Client{Timeout: 8 * time.Second}
	var mu sync.Mutex
	var wg sync.WaitGroup
	updated := 0

	for i, t := range tracks {
		if t.SpotifyID == "" {
			continue
		}

		wg.Add(1)
		go func(idx int, track SearchResult) {
			defer wg.Done()

			endpoint := "https://soundstat.info/api/v1/track/" + url.PathEscape(track.SpotifyID)
			req, _ := http.NewRequest("GET", endpoint, nil)
			req.Header.Set("x-api-key", apiKey)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			if resp.StatusCode == http.StatusAccepted {
				resp.Body.Close()
				statusReq, _ := http.NewRequest("GET", endpoint+"/status", nil)
				statusReq.Header.Set("x-api-key", apiKey)
				if statusResp, err := client.Do(statusReq); err == nil {
					statusResp.Body.Close()
				}
				req2, _ := http.NewRequest("GET", endpoint, nil)
				req2.Header.Set("x-api-key", apiKey)
				resp, err = client.Do(req2)
				if err != nil {
					return
				}
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return
			}

			var data soundStatTrackResponse
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				resp.Body.Close()
				return
			}
			resp.Body.Close()

			dance := data.Features.Danceability
			energy := data.Features.Energy
			valence := data.Features.Valence
			acoustic := data.Features.Acousticness
			instrumental := data.Features.Instrumentalness
			speech := data.Features.Speechiness
			tempo := data.Features.Tempo

			if dance == 0 && energy == 0 && valence == 0 && acoustic == 0 && instrumental == 0 && speech == 0 && tempo == 0 {
				dance = data.AudioAnalysis.Danceability.Value
				energy = data.AudioAnalysis.Energy.Value
				valence = data.AudioAnalysis.Valence.Value
				acoustic = data.AudioAnalysis.Acousticness.Value
				instrumental = data.AudioAnalysis.Instrumentalness.Value
				speech = data.AudioAnalysis.Speechiness.Value
				tempo = data.AudioAnalysis.Tempo.Value
			}

			mu.Lock()
			tracks[idx].Danceability = dance
			tracks[idx].Energy = energy
			tracks[idx].Valence = valence
			tracks[idx].Acousticness = acoustic
			tracks[idx].Instrumentalness = instrumental
			tracks[idx].Speechiness = speech
			tracks[idx].Tempo = tempo
			if dance > 0 || energy > 0 || valence > 0 || acoustic > 0 || instrumental > 0 || speech > 0 || tempo > 0 {
				updated++
			}
			mu.Unlock()
		}(i, t)
	}

	wg.Wait()
	if updated > 0 {
		debug.Logf("SearchTracks: SoundStat audio features applied to %d tracks", updated)
	}
}

type reccobeatsAudioFeaturesResponse struct {
	Content []struct {
		Href             string  `json:"href"`
		Acousticness     float64 `json:"acousticness"`
		Danceability     float64 `json:"danceability"`
		Energy           float64 `json:"energy"`
		Instrumentalness float64 `json:"instrumentalness"`
		Speechiness      float64 `json:"speechiness"`
		Tempo            float64 `json:"tempo"`
		Valence          float64 `json:"valence"`
	} `json:"content"`
}

func enrichAudioFeaturesFromReccobeats(tracks []SearchResult) {
	ids := make([]string, 0, len(tracks))
	idxBySpotifyID := make(map[string]int)
	for i, t := range tracks {
		if t.SpotifyID == "" {
			continue
		}
		ids = append(ids, t.SpotifyID)
		idxBySpotifyID[t.SpotifyID] = i
	}
	if len(ids) == 0 {
		return
	}

	reqURL := "https://api.reccobeats.com/v1/audio-features?ids=" + strings.Join(ids, ",")
	req, _ := http.NewRequest("GET", reqURL, nil)
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		debug.Logf("SearchTracks: reccobeats request failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		debug.Logf("SearchTracks: reccobeats status=%d", resp.StatusCode)
		return
	}

	var payload reccobeatsAudioFeaturesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		debug.Logf("SearchTracks: reccobeats decode failed: %v", err)
		return
	}

	updated := 0
	for _, item := range payload.Content {
		spotifyID := ""
		if parsed, err := url.Parse(item.Href); err == nil {
			segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			if len(segments) >= 2 && segments[0] == "track" {
				spotifyID = segments[1]
			}
		}
		if spotifyID == "" {
			continue
		}
		idx, ok := idxBySpotifyID[spotifyID]
		if !ok {
			continue
		}
		tracks[idx].Danceability = item.Danceability
		tracks[idx].Energy = item.Energy
		tracks[idx].Valence = item.Valence
		tracks[idx].Acousticness = item.Acousticness
		tracks[idx].Instrumentalness = item.Instrumentalness
		tracks[idx].Speechiness = item.Speechiness
		tracks[idx].Tempo = item.Tempo
		updated++
	}
	if updated > 0 {
		debug.Logf("SearchTracks: Reccobeats audio features applied to %d tracks", updated)
	}
}

func hasAnyAudioFeatures(tracks []SearchResult) bool {
	for _, t := range tracks {
		if t.Danceability > 0 || t.Energy > 0 || t.Valence > 0 || t.Acousticness > 0 || t.Instrumentalness > 0 || t.Speechiness > 0 || t.Tempo > 0 {
			return true
		}
	}
	return false
}

func joinArtistNames(artists []spotify.SimpleArtist) string {
	names := make([]string, len(artists))
	for i, a := range artists {
		names[i] = a.Name
	}
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}

func albumArtURL(images []spotify.Image) string {
	if len(images) > 0 {
		return images[0].URL
	}
	return ""
}
