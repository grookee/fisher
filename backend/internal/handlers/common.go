package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fisher/backend/internal/auth"
	"github.com/fisher/backend/internal/database"
)

func mustUser(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims := auth.GetUser(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return nil, false
	}
	return claims, true
}

func decode(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func parseLimit(r *http.Request, key string, fallback, max int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	v, err := strconv.Atoi(value)
	if err != nil || v <= 0 {
		return fallback
	}
	if v > max {
		return max
	}
	return v
}

func randGenreNames(ctx context.Context, n int, fallback []string) []string {
	rows, err := database.Pool.Query(ctx, `SELECT name FROM genres ORDER BY RANDOM() LIMIT $1`, n)
	if err != nil {
		return fallback
	}
	defer rows.Close()

	names := make([]string, 0, n)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return fallback
	}
	return names
}

func uniq(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		result = append(result, v)
	}
	return result
}

func okMsg(w http.ResponseWriter, msg string) {
	respondJSON(w, http.StatusOK, map[string]string{"message": msg})
}
