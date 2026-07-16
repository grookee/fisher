package applemusic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Service struct {
	developerToken string
	httpClient     *http.Client
}

type SearchResult struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	AlbumArt  string `json:"album_art_url"`
	Duration  int    `json:"duration_ms"`
	AppleID   string `json:"apple_music_id"`
	Preview   string `json:"preview_url"`
}

type AppleMusicResponse struct {
	Results struct {
		Songs struct {
			Data []SongData `json:"data"`
		} `json:"songs"`
	} `json:"results"`
}

type SongData struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name            string `json:"name"`
		ArtistName      string `json:"artistName"`
		AlbumName       string `json:"albumName"`
		DurationInMillis int    `json:"durationInMillis"`
		PreviewURLs     []struct {
			URL string `json:"url"`
		} `json:"previews"`
		Artwork struct {
			URL  string `json:"url"`
			Width int   `json:"width"`
			Height int `json:"height"`
		} `json:"artwork"`
	} `json:"attributes"`
}

func New() *Service {
	return &Service{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Service) Init() {
	token := os.Getenv("APPLE_MUSIC_DEVELOPER_TOKEN")
	if token == "" {
		fmt.Println("Apple Music: APPLE_MUSIC_DEVELOPER_TOKEN not set. To use Apple Music:")
		fmt.Println("  1. Enroll in Apple Developer Program ($99/yr): https://developer.apple.com/programs/")
		fmt.Println("  2. Create a MusicKit identifier and generate a developer token")
		fmt.Println("  3. Set APPLE_MUSIC_DEVELOPER_TOKEN in your environment")
		return
	}
	s.developerToken = token
	fmt.Println("Apple Music: configured (search available, OAuth flow TBD)")
}

func (s *Service) SearchTracks(query string, limit int) ([]SearchResult, error) {
	if s.developerToken == "" {
		return nil, fmt.Errorf("apple music not configured")
	}

	baseURL := "https://api.music.apple.com/v1/catalog/us/search"
	params := url.Values{}
	params.Set("term", query)
	params.Set("types", "songs")
	params.Set("limit", fmt.Sprintf("%d", limit))

	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.developerToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result AppleMusicResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var tracks []SearchResult
	for _, song := range result.Results.Songs.Data {
		track := SearchResult{
			ID:       song.ID,
			Title:    song.Attributes.Name,
			Artist:   song.Attributes.ArtistName,
			Album:    song.Attributes.AlbumName,
			Duration: song.Attributes.DurationInMillis,
			AppleID:  song.ID,
		}

		if song.Attributes.Artwork.URL != "" {
			track.AlbumArt = strings.Replace(
				song.Attributes.Artwork.URL,
				"{w}x{h}",
				fmt.Sprintf("%dx%d", song.Attributes.Artwork.Width, song.Attributes.Artwork.Height),
				1,
			)
		}

		if len(song.Attributes.PreviewURLs) > 0 {
			track.Preview = song.Attributes.PreviewURLs[0].URL
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}
