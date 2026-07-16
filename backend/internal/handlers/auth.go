package handlers

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/fisher/backend/internal/auth"
	"github.com/fisher/backend/internal/database"
	"github.com/fisher/backend/internal/debug"
	"github.com/fisher/backend/internal/models"
	"github.com/jackc/pgx/v5"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if !decode(w, r, &req) {
		return
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "email, username, and password required")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	if !emailRegex.MatchString(req.Email) {
		respondError(w, http.StatusBadRequest, "please enter a valid email address")
		return
	}
	if len(req.Username) < 2 {
		respondError(w, http.StatusBadRequest, "username must be at least 2 characters")
		return
	}
	if len(req.Password) < 6 {
		respondError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	var existing int
	database.Pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM users WHERE email = $1 OR username = $2`,
		req.Email, req.Username,
	).Scan(&existing)
	if existing > 0 {
		database.Pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM users WHERE email = $1`, req.Email,
		).Scan(&existing)
		if existing > 0 {
			respondError(w, http.StatusConflict, "an account with this email already exists")
			return
		}
		respondError(w, http.StatusConflict, "this username is already taken")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		debug.Logf("failed to hash password for user %s: %v", req.Username, err)
		respondError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	var user models.User
	err = database.Pool.QueryRow(r.Context(),
		`INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, email, username, avatar_url, created_at, updated_at`,
		req.Email, req.Username, hash,
	).Scan(&user.ID, &user.Email, &user.Username, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		debug.Logf("failed to create user %s: %v", req.Username, err)
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, user.Username)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respondJSON(w, http.StatusCreated, models.AuthResponse{Token: token, User: user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if !decode(w, r, &req) {
		return
	}

	var user models.User
	err := database.Pool.QueryRow(r.Context(),
		`SELECT id, email, username, password_hash, avatar_url, created_at, updated_at FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, user.Username)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: user})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := mustUser(w, r)
	if !ok {
		return
	}

	var user models.User
	err := database.Pool.QueryRow(r.Context(),
		`SELECT id, email, username, avatar_url, created_at, updated_at FROM users WHERE id = $1`,
		claims.UserID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}
