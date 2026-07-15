package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv loads .env from the current directory, then repo-root .env when
// running from backend\ (parent path), so root-level config can override stale
// backend-local values.
func LoadEnv() {
	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
	}

	existing := make([]string, 0, len(candidates))
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	if len(existing) == 0 {
		return
	}

	merged := map[string]string{}
	for _, path := range existing {
		values, err := godotenv.Read(path)
		if err != nil {
			continue
		}
		for k, v := range values {
			merged[k] = v
		}
	}
	for k, v := range merged {
		if _, alreadySet := os.LookupEnv(k); alreadySet {
			continue
		}
		_ = os.Setenv(k, v)
	}
}
