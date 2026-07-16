package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/models"
)

type PlaylistHandler struct{}

func NewPlaylistHandler() *PlaylistHandler {
	return &PlaylistHandler{}
}

func (h *PlaylistHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	rows, err := database.Pool.Query(r.Context(),
		`SELECT p.id, p.title, p.description, p.owner_id, p.is_public, p.created_at, p.updated_at
		 FROM playlists p
		 WHERE p.owner_id = $1 OR p.is_public = true OR EXISTS (
		   SELECT 1 FROM collaborations c WHERE c.playlist_id = p.id AND c.user_id = $1
		 )
		 ORDER BY p.updated_at DESC`,
		claims.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch playlists")
		return
	}
	defer rows.Close()

	playlists := []models.Playlist{}
	for rows.Next() {
		var p models.Playlist
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.OwnerID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt); err == nil {
			playlists = append(playlists, p)
		}
	}

	respondJSON(w, http.StatusOK, playlists)
}

func (h *PlaylistHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	playlistID := chi.URLParam(r, "id")
	var p models.Playlist
	err := database.Pool.QueryRow(r.Context(),
		`SELECT p.id, p.title, p.description, p.owner_id, p.is_public, p.created_at, p.updated_at
		 FROM playlists p WHERE p.id = $1`, playlistID,
	).Scan(&p.ID, &p.Title, &p.Description, &p.OwnerID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		respondError(w, http.StatusNotFound, "playlist not found")
		return
	}

	if !p.IsPublic && p.OwnerID != claims.UserID {
		var count int
		database.Pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM collaborations WHERE playlist_id = $1 AND user_id = $2`,
			playlistID, claims.UserID,
		).Scan(&count)
		if count == 0 {
			respondError(w, http.StatusForbidden, "not authorized")
			return
		}
	}

	if tracks, err := listTracks(r.Context(), playlistID); err == nil {
		p.Tracks = tracks
	}

	respondJSON(w, http.StatusOK, p)
}

func (h *PlaylistHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	var req models.CreatePlaylistRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	var p models.Playlist
	err := database.Pool.QueryRow(r.Context(),
		`INSERT INTO playlists (title, description, owner_id, is_public)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, title, description, owner_id, is_public, created_at, updated_at`,
		req.Title, req.Description, claims.UserID, req.IsPublic,
	).Scan(&p.ID, &p.Title, &p.Description, &p.OwnerID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create playlist")
		return
	}

	for i, trackID := range req.TrackIDs {
		database.Pool.Exec(r.Context(),
			`INSERT INTO playlist_tracks (playlist_id, track_id, position) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			p.ID, trackID, i,
		)
	}

	respondJSON(w, http.StatusCreated, p)
}

func (h *PlaylistHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	playlistID := chi.URLParam(r, "id")
	var req models.UpdatePlaylistRequest
	if !decode(w, r, &req) {
		return
	}
	if !canEdit(r, playlistID, claims.UserID) {
		respondError(w, http.StatusForbidden, "not authorized")
		return
	}

	var p models.Playlist
	err := database.Pool.QueryRow(r.Context(),
		`UPDATE playlists SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			is_public = COALESCE($4, is_public),
			updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, title, description, owner_id, is_public, created_at, updated_at`,
		playlistID, req.Title, req.Description, req.IsPublic,
	).Scan(&p.ID, &p.Title, &p.Description, &p.OwnerID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update playlist")
		return
	}

	for _, trackID := range req.AddTracks {
		database.Pool.Exec(r.Context(),
			`INSERT INTO playlist_tracks (playlist_id, track_id, position)
			 VALUES ($1, $2, (SELECT COALESCE(MAX(position), -1) + 1 FROM playlist_tracks WHERE playlist_id = $1))
			 ON CONFLICT DO NOTHING`,
			playlistID, trackID,
		)
	}

	for _, trackID := range req.RemoveTracks {
		database.Pool.Exec(r.Context(),
			`DELETE FROM playlist_tracks WHERE playlist_id = $1 AND track_id = $2`,
			playlistID, trackID,
		)
	}

	respondJSON(w, http.StatusOK, p)
}

func (h *PlaylistHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	playlistID := chi.URLParam(r, "id")
	var ownerID string
	err := database.Pool.QueryRow(r.Context(), `SELECT owner_id FROM playlists WHERE id = $1`, playlistID).Scan(&ownerID)
	if err != nil {
		respondError(w, http.StatusNotFound, "playlist not found")
		return
	}
	if ownerID != claims.UserID {
		respondError(w, http.StatusForbidden, "only owner can delete")
		return
	}

	database.Pool.Exec(r.Context(), `DELETE FROM playlists WHERE id = $1`, playlistID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *PlaylistHandler) AddCollaborator(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	playlistID := chi.URLParam(r, "id")
	var req models.AddCollaboratorRequest
	if !decode(w, r, &req) {
		return
	}
	if !canAdmin(r, playlistID, claims.UserID) {
		respondError(w, http.StatusForbidden, "not authorized")
		return
	}

	_, err := database.Pool.Exec(r.Context(),
		`INSERT INTO collaborations (playlist_id, user_id, permission) VALUES ($1, $2, $3)
		 ON CONFLICT (playlist_id, user_id) DO UPDATE SET permission = $3`,
		playlistID, req.UserID, req.Permission,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to add collaborator")
		return
	}

	okMsg(w, "collaborator added")
}

func (h *PlaylistHandler) RemoveCollaborator(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	playlistID := chi.URLParam(r, "id")
	collabID := chi.URLParam(r, "userId")
	if !canAdmin(r, playlistID, claims.UserID) {
		respondError(w, http.StatusForbidden, "not authorized")
		return
	}

	database.Pool.Exec(r.Context(),
		`DELETE FROM collaborations WHERE playlist_id = $1 AND user_id = $2`,
		playlistID, collabID,
	)
	w.WriteHeader(http.StatusNoContent)
}

func (h *PlaylistHandler) SearchTracks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondJSON(w, http.StatusOK, []models.Track{})
		return
	}

	rows, err := database.Pool.Query(r.Context(),
		`SELECT id, title, artist, album, album_art_url, duration_ms, spotify_uri, apple_music_id, preview_url,
		        release_year, danceability, energy, valence, acousticness, instrumentalness, speechiness, tempo
		 FROM tracks WHERE title ILIKE '%' || $1 || '%' OR artist ILIKE '%' || $1 || '%'
		 LIMIT 20`,
		query,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "search failed")
		return
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

	if len(tracks) == 0 {
		tracks = searchExt(r.Context(), query)
	}

	respondJSON(w, http.StatusOK, tracks)
}
