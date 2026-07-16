package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fisher/backend/internal/database"
	"github.com/go-chi/chi/v5"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type OAuthHandler struct{}

func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{}
}

func newState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *OAuthHandler) SpotifyAuthorize(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")
	if clientID == "" || redirectURI == "" {
		respondError(w, http.StatusInternalServerError, "Spotify OAuth not configured")
		return
	}

	state := newState()
	database.Pool.Exec(r.Context(),
		`INSERT INTO oauth_states (state, user_id, service) VALUES ($1, $2, 'spotify')`,
		state, claims.UserID,
	)

	auth := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPrivate,
			spotifyauth.ScopeUserReadEmail,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
			spotifyauth.ScopeUserTopRead,
		),
	)

	respondJSON(w, http.StatusOK, map[string]string{"url": auth.AuthURL(state)})
}

func (h *OAuthHandler) SpotifyCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if errText := r.URL.Query().Get("error"); errText != "" {
		http.Error(w, fmt.Sprintf("spotify auth error: %s", errText), http.StatusBadRequest)
		return
	}
	if code == "" || state == "" {
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	var userID string
	err := database.Pool.QueryRow(r.Context(),
		`DELETE FROM oauth_states WHERE state = $1 AND service = 'spotify' AND created_at > NOW() - INTERVAL '10 minutes' RETURNING user_id`,
		state,
	).Scan(&userID)
	if err != nil {
		http.Error(w, "invalid or expired state", http.StatusBadRequest)
		return
	}

	auth := spotifyauth.New(
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_CLIENT_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_CLIENT_SECRET")),
		spotifyauth.WithRedirectURL(os.Getenv("SPOTIFY_REDIRECT_URI")),
	)

	token, err := auth.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, fmt.Sprintf("token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}

	client := auth.Client(r.Context(), token)
	userResp, err := client.Get("https://api.spotify.com/v1/me")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get user info: %v", err), http.StatusInternalServerError)
		return
	}
	defer userResp.Body.Close()

	var spotifyUser struct {
		ID string `json:"id"`
	}
	json.NewDecoder(userResp.Body).Decode(&spotifyUser)

	expiry := time.Now()
	if !token.Expiry.IsZero() {
		expiry = token.Expiry
	}

	_, err = database.Pool.Exec(r.Context(),
		`INSERT INTO user_accounts (user_id, service, service_user_id, access_token, refresh_token, token_expiry)
		 VALUES ($1, 'spotify', $2, $3, $4, $5)
		 ON CONFLICT (user_id, service) DO UPDATE SET
		   service_user_id = EXCLUDED.service_user_id,
		   access_token = EXCLUDED.access_token,
		   refresh_token = EXCLUDED.refresh_token,
		   token_expiry = EXCLUDED.token_expiry,
		   updated_at = NOW()`,
		userID, spotifyUser.ID, token.AccessToken, token.RefreshToken, expiry,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to store account: %v", err), http.StatusInternalServerError)
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	http.Redirect(w, r, frontendURL+"/auth?linked=spotify", http.StatusFound)
}

func (h *OAuthHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	rows, err := database.Pool.Query(r.Context(),
		`SELECT service, service_user_id, token_expiry, updated_at FROM user_accounts WHERE user_id = $1`,
		claims.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch accounts")
		return
	}
	defer rows.Close()

	type Account struct {
		Service       string     `json:"service"`
		ServiceUserID string     `json:"service_user_id"`
		TokenExpiry   *time.Time `json:"token_expiry"`
		UpdatedAt     time.Time  `json:"updated_at"`
	}

	accounts := []Account{}
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.Service, &a.ServiceUserID, &a.TokenExpiry, &a.UpdatedAt); err == nil {
			accounts = append(accounts, a)
		}
	}

	respondJSON(w, http.StatusOK, accounts)
}

func (h *OAuthHandler) UnlinkAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	service := chi.URLParam(r, "service")
	if service != "spotify" && service != "apple" {
		respondError(w, http.StatusBadRequest, "invalid service")
		return
	}

	_, err := database.Pool.Exec(r.Context(),
		`DELETE FROM user_accounts WHERE user_id = $1 AND service = $2`,
		claims.UserID, service,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to unlink account")
		return
	}

	okMsg(w, "account unlinked")
}
