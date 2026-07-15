package database

import (
	"context"
	"fmt"
	"log"
)

func Migrate(ctx context.Context) error {
	for _, sql := range schemaMigrations {
		if _, err := Pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("migration: %w\nSQL: %s", err, sql)
		}
	}

	seedRootGenres(ctx)
	seedSubGenres(ctx)

	log.Println("migrations complete")
	return nil
}

func seedRootGenres(ctx context.Context) {
	for _, g := range rootGenres {
		Pool.Exec(ctx,
			`INSERT INTO genres (name, description, color, x, y) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (name) DO NOTHING`,
			g.Name, g.Description, g.Color, g.X, g.Y,
		)
	}
}

func seedSubGenres(ctx context.Context) {
	for _, g := range subGenres {
		Pool.Exec(ctx,
			`INSERT INTO genres (name, description, color, x, y, parent_id)
			 VALUES ($1, $2, $3, $4, $5, (SELECT id FROM genres WHERE name = $6))
			 ON CONFLICT (name) DO NOTHING`,
			g.Name, g.Description, g.Color, g.X, g.Y, g.Parent,
		)
	}
}
