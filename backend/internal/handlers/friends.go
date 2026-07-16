package handlers

import (
	"net/http"

	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/models"
)

type FriendHandler struct{}

func NewFriendHandler() *FriendHandler {
	return &FriendHandler{}
}

func (h *FriendHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	rows, err := database.Pool.Query(r.Context(),
		`SELECT u.id, u.username, u.email, u.avatar_url, f.status
		 FROM friends f
		 JOIN users u ON u.id = f.friend_id
		 WHERE f.user_id = $1 AND f.status = 'accepted'
		 UNION
		 SELECT u.id, u.username, u.email, u.avatar_url, f.status
		 FROM friends f
		 JOIN users u ON u.id = f.user_id
		 WHERE f.friend_id = $1 AND f.status = 'accepted'`,
		claims.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch friends")
		return
	}
	defer rows.Close()

	type FriendWithUser struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar_url"`
		Status   string `json:"status"`
	}

	friends := []FriendWithUser{}
	for rows.Next() {
		var f FriendWithUser
		if err := rows.Scan(&f.ID, &f.Username, &f.Email, &f.Avatar, &f.Status); err == nil {
			friends = append(friends, f)
		}
	}

	respondJSON(w, http.StatusOK, friends)
}

func (h *FriendHandler) Requests(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	rows, err := database.Pool.Query(r.Context(),
		`SELECT u.id, u.username, u.email, u.avatar_url, f.status, f.created_at
		 FROM friends f
		 JOIN users u ON u.id = f.user_id
		 WHERE f.friend_id = $1 AND f.status = 'pending'
		 ORDER BY f.created_at DESC`,
		claims.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch requests")
		return
	}
	defer rows.Close()

	type FriendRequest struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
		Status    string `json:"status"`
	}

	requests := []FriendRequest{}
	for rows.Next() {
		var f FriendRequest
		if err := rows.Scan(&f.ID, &f.Username, &f.Email, &f.AvatarURL, &f.Status); err == nil {
			requests = append(requests, f)
		}
	}

	respondJSON(w, http.StatusOK, requests)
}

func (h *FriendHandler) SendRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	var req models.FriendRequest
	if !decode(w, r, &req) {
		return
	}
	if req.FriendID == claims.UserID {
		respondError(w, http.StatusBadRequest, "cannot add yourself")
		return
	}

	_, err := database.Pool.Exec(r.Context(),
		`INSERT INTO friends (user_id, friend_id, status) VALUES ($1, $2, 'pending')
		 ON CONFLICT (user_id, friend_id) DO NOTHING`,
		claims.UserID, req.FriendID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to send request")
		return
	}

	okMsg(w, "friend request sent")
}

func (h *FriendHandler) AcceptRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	var req models.FriendRequest
	if !decode(w, r, &req) {
		return
	}

	_, err := database.Pool.Exec(r.Context(),
		`UPDATE friends SET status = 'accepted' WHERE user_id = $1 AND friend_id = $2 AND status = 'pending'`,
		req.FriendID, claims.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to accept request")
		return
	}

	okMsg(w, "friend request accepted")
}

func (h *FriendHandler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	friendID := r.URL.Query().Get("friend_id")
	if friendID == "" {
		respondError(w, http.StatusBadRequest, "friend_id is required")
		return
	}

	database.Pool.Exec(r.Context(),
		`DELETE FROM friends WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`,
		claims.UserID, friendID,
	)
	w.WriteHeader(http.StatusNoContent)
}

func (h *FriendHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}

	rows, err := database.Pool.Query(r.Context(),
		`SELECT id, username, email, avatar_url FROM users
		 WHERE username ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%'
		 LIMIT 20`,
		query,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "search failed")
		return
	}
	defer rows.Close()

	type UserBrief struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	users := []UserBrief{}
	for rows.Next() {
		var u UserBrief
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.AvatarURL); err == nil {
			users = append(users, u)
		}
	}

	respondJSON(w, http.StatusOK, users)
}
