package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Genre struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Color       string  `json:"color"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	ParentID    *string `json:"parent_id,omitempty"`
}

type Track struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Artist          string   `json:"artist"`
	Album           string   `json:"album"`
	AlbumArtURL     string   `json:"album_art_url"`
	DurationMs      int      `json:"duration_ms"`
	SpotifyURI      string   `json:"spotify_uri,omitempty"`
	AppleMusicID    string   `json:"apple_music_id,omitempty"`
	PreviewURL      string   `json:"preview_url,omitempty"`
	ReleaseYear     int      `json:"release_year,omitempty"`
	GenreIDs        []string `json:"genre_ids,omitempty"`
	Danceability    float64  `json:"danceability,omitempty"`
	Energy          float64  `json:"energy,omitempty"`
	Valence         float64  `json:"valence,omitempty"`
	Acousticness    float64  `json:"acousticness,omitempty"`
	Instrumentalness float64 `json:"instrumentalness,omitempty"`
	Speechiness     float64  `json:"speechiness,omitempty"`
	Tempo           float64  `json:"tempo,omitempty"`
}

type Playlist struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	OwnerID     string    `json:"owner_id"`
	IsPublic    bool      `json:"is_public"`
	TrackIDs    []string  `json:"track_ids,omitempty"`
	Tracks      []Track   `json:"tracks,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Collaboration struct {
	PlaylistID string `json:"playlist_id"`
	UserID     string `json:"user_id"`
	Permission string `json:"permission"` // "view", "edit", "admin"
}

type Friend struct {
	UserID   string `json:"user_id"`
	FriendID string `json:"friend_id"`
	Status   string `json:"status"` // "pending", "accepted", "blocked"
}

type AudioFeatures struct {
	Danceability     float64 `json:"danceability"`
	Energy           float64 `json:"energy"`
	Valence          float64 `json:"valence"`
	Acousticness     float64 `json:"acousticness"`
	Instrumentalness float64 `json:"instrumentalness"`
	Speechiness      float64 `json:"speechiness"`
	Liveness         float64 `json:"liveness"`
	Tempo            float64 `json:"tempo"`
}

type TasteProfile struct {
	UserID       string         `json:"user_id"`
	Genres       []GenreWeight  `json:"genres"`
	TopTracks    []Track        `json:"top_tracks"`
	TopArtists   []string       `json:"top_artists"`
	AudioFeatures *AudioFeatures `json:"audio_features,omitempty"`
	SharedWith   []string       `json:"shared_with,omitempty"`
}

type GenreWeight struct {
	GenreID string  `json:"genre_id"`
	Weight  float64 `json:"weight"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreatePlaylistRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	IsPublic    bool     `json:"is_public"`
	TrackIDs    []string `json:"track_ids,omitempty"`
}

type UpdatePlaylistRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	IsPublic    *bool    `json:"is_public,omitempty"`
	AddTracks   []string `json:"add_tracks,omitempty"`
	RemoveTracks []string `json:"remove_tracks,omitempty"`
}

type AddCollaboratorRequest struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
}

type FriendRequest struct {
	FriendID string `json:"friend_id"`
}

type ShareTasteRequest struct {
	UserIDs []string `json:"user_ids"`
}
