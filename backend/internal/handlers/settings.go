package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fisher/backend/internal/auth"
	"github.com/fisher/backend/internal/database"
)

type SettingsHandler struct{}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

func (h *SettingsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())

	rows, err := database.Pool.Query(r.Context(),
		`SELECT key, value FROM user_settings WHERE user_id = $1`,
		claims.UserID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch settings")
		return
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		settings[k] = v
	}

	respondJSON(w, http.StatusOK, settings)
}

func (h *SettingsHandler) Set(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUser(r.Context())

	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		respondError(w, http.StatusBadRequest, "key is required")
		return
	}

	_, err := database.Pool.Exec(r.Context(),
		`INSERT INTO user_settings (user_id, key, value)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, key) DO UPDATE SET value = EXCLUDED.value`,
		claims.UserID, req.Key, req.Value,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save setting")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "setting saved"})
}
