package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type contextKey string

const httpClientKey contextKey = "httpClient"

var (
	apiKey        string
	defaultClient = &http.Client{Timeout: 60 * time.Second}

	apiCache   = make(map[string]apiCacheEntry)
	apiCacheMu sync.RWMutex
)

type apiCacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

const apiCacheTTL = 1 * time.Hour

func cacheGet(key string, dest interface{}) bool {
	apiCacheMu.RLock()
	defer apiCacheMu.RUnlock()
	entry, ok := apiCache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return false
	}
	// copy via JSON round-trip
	data, _ := json.Marshal(entry.value)
	return json.Unmarshal(data, dest) == nil
}

func cacheSet(key string, value interface{}) {
	apiCacheMu.Lock()
	defer apiCacheMu.Unlock()
	apiCache[key] = apiCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(apiCacheTTL),
	}
}

func getClient() *http.Client {
	return defaultClient
}

func ContextWithClient(ctx context.Context, client *http.Client) context.Context {
	return context.WithValue(ctx, httpClientKey, client)
}

type ArtistInfo struct {
	Name  string   `json:"name"`
	Tags  []string `json:"tags"`
	Image string   `json:"image"`
}

type topArtistsResponse struct {
	TopArtists struct {
		Artist []struct {
			Name  string `json:"name"`
			Image []struct {
				Size string `json:"size"`
				Text string `json:"#text"`
			} `json:"image"`
		} `json:"artist"`
	} `json:"topartists"`
}

type artistInfoResponse struct {
	Artist struct {
		Name string `json:"name"`
		Tags struct {
			Tag []struct {
				Name string `json:"name"`
			} `json:"tag"`
		} `json:"tags"`
		Bio struct {
			Content string `json:"content"`
		} `json:"bio"`
	} `json:"artist"`
}

func Init() {
	apiKey = os.Getenv("LASTFM_API_KEY")
	if apiKey == "" {
		fmt.Println("warning: last.fm API key not configured")
	}
}

func IsConfigured() bool {
	return apiKey != ""
}

func FetchTopArtists(username string, limit int) ([]ArtistInfo, error) {
	if !IsConfigured() {
		return nil, fmt.Errorf("last.fm not configured")
	}

	cacheKey := fmt.Sprintf("topartists:%s:%d", username, limit)
	var cached []ArtistInfo
	if cacheGet(cacheKey, &cached) {
		return cached, nil
	}

	params := url.Values{}
	params.Set("method", "user.gettopartists")
	params.Set("user", username)
	params.Set("api_key", apiKey)
	params.Set("format", "json")
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("period", "12month")

	resp, err := getClient().Get("https://ws.audioscrobbler.com/2.0/?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top artists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("last.fm API returned status %d", resp.StatusCode)
	}

	var result topArtistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	artists := make([]ArtistInfo, 0, len(result.TopArtists.Artist))
	for _, a := range result.TopArtists.Artist {
		info := ArtistInfo{Name: a.Name}
		for _, img := range a.Image {
			if img.Size == "large" {
				info.Image = img.Text
				break
			}
		}

		tags, _ := FetchArtistTags(a.Name)
		info.Tags = tags
		artists = append(artists, info)
	}

	cacheSet(cacheKey, artists)
	return artists, nil
}

func FetchArtistTags(artistName string) ([]string, error) {
	cacheKey := fmt.Sprintf("arttags:%s", artistName)
	var cached []string
	if cacheGet(cacheKey, &cached) {
		return cached, nil
	}

	params := url.Values{}
	params.Set("method", "artist.getinfo")
	params.Set("artist", artistName)
	params.Set("api_key", apiKey)
	params.Set("format", "json")

	resp, err := getClient().Get("https://ws.audioscrobbler.com/2.0/?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result artistInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	tags := make([]string, 0, len(result.Artist.Tags.Tag))
	for _, t := range result.Artist.Tags.Tag {
		name := strings.TrimSpace(t.Name)
		if name != "" {
			tags = append(tags, strings.ToLower(name))
		}
	}

	cacheSet(cacheKey, tags)
	return tags, nil
}

var lastFmToOurGenre = map[string]string{
	"rock":          "Rock",
	"alternative":   "Indie",
	"indie":         "Indie",
	"pop":           "Pop",
	"electronic":    "Electronic",
	"hip-hop":       "Hip Hop",
	"hip hop":       "Hip Hop",
	"rap":           "Hip Hop",
	"jazz":          "Jazz",
	"classical":     "Classical",
	"r&b":           "R&B",
	"rnb":           "R&B",
	"soul":          "Soul",
	"funk":          "Funk",
	"reggae":        "Reggae",
	"blues":         "Blues",
	"country":       "Country",
	"folk":          "Folk",
	"metal":         "Metal",
	"punk":          "Punk",
	"ambient":       "Ambient",
	"latin":         "Latin",
	"world":         "World",
	"gospel":        "Gospel",
	"disco":         "Funk",
	"house":         "Electronic",
	"techno":        "Electronic",
	"dubstep":       "Electronic",
	"drum and bass": "Electronic",
	"trance":        "Electronic",
	"edm":           "Electronic",
}

func MapTagsToOurGenre(tags []string) string {
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if mapped, ok := lastFmToOurGenre[tag]; ok {
			return mapped
		}
	}
	return ""
}

type ArtistTopTag struct {
	Name  string
	Count int
}

type artistTopTagsResponse struct {
	TopTags struct {
		Tag []struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		} `json:"tag"`
	} `json:"toptags"`
}

// FetchArtistTopTags returns an artist's Last.fm folksonomy tags ordered by
// relevance, with Last.fm's 0-100 relative weight for each tag. Used to
// build the genre co-occurrence graph.
func FetchArtistTopTags(artistName string) ([]ArtistTopTag, error) {
	if !IsConfigured() {
		return nil, fmt.Errorf("last.fm not configured")
	}

	params := url.Values{}
	params.Set("method", "artist.gettoptags")
	params.Set("artist", artistName)
	params.Set("api_key", apiKey)
	params.Set("format", "json")

	resp, err := getClient().Get("https://ws.audioscrobbler.com/2.0/?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("last.fm API returned status %d", resp.StatusCode)
	}

	var result artistTopTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tags := make([]ArtistTopTag, 0, len(result.TopTags.Tag))
	for _, t := range result.TopTags.Tag {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		tags = append(tags, ArtistTopTag{Name: name, Count: t.Count})
	}
	return tags, nil
}

type tagTopArtistsResponse struct {
	TopArtists struct {
		Artist []struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"topartists"`
}

// FetchTopArtistsForTag returns the top artists Last.fm associates with a
// given tag/genre name. Treating each of Fisher's genre names as a Last.fm
// tag lets us sample real-world genre-to-artist associations for free.
func FetchTopArtistsForTag(tag string, limit int) ([]string, error) {
	if !IsConfigured() {
		return nil, fmt.Errorf("last.fm not configured")
	}
	if limit <= 0 {
		limit = 15
	}

	params := url.Values{}
	params.Set("method", "tag.gettopartists")
	params.Set("tag", tag)
	params.Set("api_key", apiKey)
	params.Set("format", "json")
	params.Set("limit", fmt.Sprintf("%d", limit))

	resp, err := getClient().Get("https://ws.audioscrobbler.com/2.0/?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top artists for tag: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("last.fm API returned status %d", resp.StatusCode)
	}

	var result tagTopArtistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	names := make([]string, 0, len(result.TopArtists.Artist))
	for _, a := range result.TopArtists.Artist {
		name := strings.TrimSpace(a.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

type geoTopArtistsResponse struct {
	TopArtists struct {
		Artist []struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"topartists"`
}

// FetchTopArtistsByCountry returns a country's Last.fm top-artist chart
// (method=geo.gettopartists). This is the main lever for growing artist
// coverage beyond genre-tag sampling alone, and for surfacing artists from
// smaller/underrepresented music cultures that a purely genre-driven crawl
// would rarely reach.
func FetchTopArtistsByCountry(country string, limit int) ([]string, error) {
	if !IsConfigured() {
		return nil, fmt.Errorf("last.fm not configured")
	}
	if limit <= 0 {
		limit = 30
	}

	params := url.Values{}
	params.Set("method", "geo.gettopartists")
	params.Set("country", country)
	params.Set("api_key", apiKey)
	params.Set("format", "json")
	params.Set("limit", fmt.Sprintf("%d", limit))

	resp, err := getClient().Get("https://ws.audioscrobbler.com/2.0/?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch geo top artists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("last.fm API returned status %d", resp.StatusCode)
	}

	var result geoTopArtistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	names := make([]string, 0, len(result.TopArtists.Artist))
	for _, a := range result.TopArtists.Artist {
		name := strings.TrimSpace(a.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}
