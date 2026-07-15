package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fisher/backend/internal/auth"
)

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*rateEntry
	duration time.Duration
	max      int
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(max int, duration time.Duration) *RateLimiter {
	return &RateLimiter{
		clients:  make(map[string]*rateEntry),
		duration: duration,
		max:      max,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		claims := auth.GetUser(r.Context())
		key := ip
		if claims != nil {
			key = claims.UserID
		}

		rl.mu.Lock()
		entry, exists := rl.clients[key]
		now := time.Now()
		if !exists || now.After(entry.resetAt) {
			entry = &rateEntry{count: 0, resetAt: now.Add(rl.duration)}
			rl.clients[key] = entry
		}
		entry.count++
		count := entry.count
		rl.mu.Unlock()

		if count > rl.max {
			w.Header().Set("Retry-After", rl.duration.String())
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(parts[1])
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := auth.WithUser(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header != "" {
			parts := strings.SplitN(header, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				claims, err := auth.ValidateToken(parts[1])
				if err == nil {
					ctx := auth.WithUser(r.Context(), claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(context.Background()))
	})
}
