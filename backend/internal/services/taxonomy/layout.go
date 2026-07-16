package taxonomy

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LayoutOptions controls the force-directed layout recompute. Keep
// iterations/nodes bounded so this stays fast even on a modest free-tier VM.
type LayoutOptions struct {
	Iterations int     // number of simulation steps, e.g. 150
	Repulsion  float64 // repulsion constant between all node pairs
	Attraction float64 // attraction constant along edges, scaled by edge weight
	MaxNodes   int     // safety cap on how many genres to lay out in one run (0 = no cap)
}

func DefaultLayoutOptions() LayoutOptions {
	return LayoutOptions{
		Iterations: 150,
		Repulsion:  0.02,
		Attraction: 0.9,
		MaxNodes:   7000,
	}
}

type layoutNode struct {
	id     string
	x, y   float64
	vx, vy float64
}

type layoutEdge struct {
	from, to int // indexes into the node slice
	weight   float64
}

// RecomputeLayout runs a basic Fruchterman-Reingold-style force-directed
// layout over the genre adjacency graph (genre_relations) and writes fresh
// x/y coordinates back onto every genre that has at least one relation.
// Genres with zero relations are left untouched (their existing x/y, usually
// a hand-picked seed value or 0,0, is preserved).
//
// This is intentionally simple (O(n^2) repulsion per iteration) rather than
// using a proper graph-layout library, since it only needs to run
// occasionally as a batch job, not on the request path. It is skipped
// automatically (with an error) if the graph has more nodes than
// opts.MaxNodes, since O(n^2) becomes too slow beyond a few thousand nodes
// on typical free-tier hardware - increase MaxNodes deliberately if needed.
func RecomputeLayout(ctx context.Context, pool *pgxpool.Pool, opts LayoutOptions) (int, error) {
	start := time.Now()
	log.Printf("layout: loading genre_relations graph...")

	// Load every genre that participates in at least one relation.
	rows, err := pool.Query(ctx, `SELECT DISTINCT genre_id FROM genre_relations`)
	if err != nil {
		return 0, fmt.Errorf("load genre_relations node set: %w", err)
	}
	idToIndex := make(map[string]int)
	var nodes []layoutNode
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		if _, exists := idToIndex[id]; exists {
			continue
		}
		idToIndex[id] = len(nodes)
		nodes = append(nodes, layoutNode{id: id})
	}
	rows.Close()

	if len(nodes) == 0 {
		log.Printf("layout: no nodes in genre_relations, nothing to do")
		return 0, nil
	}
	if opts.MaxNodes > 0 && len(nodes) > opts.MaxNodes {
		return 0, fmt.Errorf("graph has %d nodes, exceeding MaxNodes=%d; increase LayoutOptions.MaxNodes to proceed", len(nodes), opts.MaxNodes)
	}

	edgeRows, err := pool.Query(ctx, `SELECT genre_id, related_genre_id, weight FROM genre_relations`)
	if err != nil {
		return 0, fmt.Errorf("load genre_relations edges: %w", err)
	}
	var edges []layoutEdge
	for edgeRows.Next() {
		var from, to string
		var weight float64
		if err := edgeRows.Scan(&from, &to, &weight); err != nil {
			continue
		}
		fi, ok1 := idToIndex[from]
		ti, ok2 := idToIndex[to]
		if !ok1 || !ok2 || fi == ti {
			continue
		}
		edges = append(edges, layoutEdge{from: fi, to: ti, weight: weight})
	}
	edgeRows.Close()

	log.Printf("layout: loaded %d nodes, %d edges — running %d iterations...", len(nodes), len(edges), opts.Iterations)

	// Deterministic pseudo-random initial positions on a circle, so repeated
	// runs converge similarly rather than depending on Go's global rand state.
	rnd := rand.New(rand.NewSource(42))
	for i := range nodes {
		angle := rnd.Float64() * 2 * math.Pi
		radius := 0.3 + rnd.Float64()*0.4
		nodes[i].x = radius * math.Cos(angle)
		nodes[i].y = radius * math.Sin(angle)
	}

	if opts.Iterations <= 0 {
		opts.Iterations = 150
	}

	// maxDisp caps the maximum position change per iteration (a standard
	// "temperature" in force-directed layouts). Without this, O(n²) repulsion
	// on large graphs causes positions to overflow to +Inf, and Inf-Inf = NaN.
	maxDisp := 0.5

	iterStart := time.Now()
	for iter := 0; iter < opts.Iterations; iter++ {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		cooling := 1.0 - float64(iter)/float64(opts.Iterations)

		// Log progress every 25 iterations
		if (iter+1)%25 == 0 || iter+1 == opts.Iterations {
			elapsed := time.Since(iterStart)
			log.Printf("layout: iteration %d/%d (%.1f%%) [%.1f iter/sec, %s elapsed]",
				iter+1, opts.Iterations, float64(iter+1)/float64(opts.Iterations)*100,
				float64(iter+1)/elapsed.Seconds(), elapsed.Round(time.Millisecond))
		}

		// Repulsion between every pair of nodes.
		for i := range nodes {
			for j := i + 1; j < len(nodes); j++ {
				dx := nodes[i].x - nodes[j].x
				dy := nodes[i].y - nodes[j].y
				distSq := dx*dx + dy*dy
				if distSq < 1e-6 {
					distSq = 1e-6
				}
				if !math.IsInf(distSq, 0) && !math.IsNaN(distSq) {
					force := opts.Repulsion / distSq
					dist := math.Sqrt(distSq)
					fx := force * dx / dist
					fy := force * dy / dist
					nodes[i].vx += fx
					nodes[i].vy += fy
					nodes[j].vx -= fx
					nodes[j].vy -= fy
				}
			}
		}

		// Attraction along edges, scaled by relation weight.
		for _, e := range edges {
			dx := nodes[e.to].x - nodes[e.from].x
			dy := nodes[e.to].y - nodes[e.from].y
			distSq := dx*dx + dy*dy
			if math.IsInf(distSq, 0) || math.IsNaN(distSq) {
				continue
			}
			dist := math.Sqrt(distSq)
			if dist < 1e-6 {
				dist = 1e-6
			}
			force := opts.Attraction * e.weight * dist
			fx := force * dx / dist
			fy := force * dy / dist
			nodes[e.from].vx += fx
			nodes[e.from].vy += fy
			nodes[e.to].vx -= fx
			nodes[e.to].vy -= fy
		}

		// Apply velocity with cooling, clamped to prevent overflow, then reset.
		for i := range nodes {
			dx := nodes[i].vx * cooling * 0.05
			dy := nodes[i].vy * cooling * 0.05
			if math.IsNaN(dx) || math.IsInf(dx, 0) {
				dx = 0
			}
			if math.IsNaN(dy) || math.IsInf(dy, 0) {
				dy = 0
			}
			if dx > maxDisp {
				dx = maxDisp
			} else if dx < -maxDisp {
				dx = -maxDisp
			}
			if dy > maxDisp {
				dy = maxDisp
			} else if dy < -maxDisp {
				dy = -maxDisp
			}
			nodes[i].x += dx
			nodes[i].y += dy
			nodes[i].vx = 0
			nodes[i].vy = 0
		}
	}

	// Normalize coordinates into roughly [-1, 1] to match the existing
	// hand-picked genre seed coordinate range.
	var maxAbs float64
	for _, n := range nodes {
		ax := math.Abs(n.x)
		ay := math.Abs(n.y)
		if math.IsNaN(ax) || math.IsInf(ax, 0) {
			continue
		}
		if math.IsNaN(ay) || math.IsInf(ay, 0) {
			continue
		}
		if ax > maxAbs {
			maxAbs = ax
		}
		if ay > maxAbs {
			maxAbs = ay
		}
	}
	if maxAbs < 1e-6 {
		maxAbs = 1
	}

	updated := 0
	for _, n := range nodes {
		if ctx.Err() != nil {
			return updated, ctx.Err()
		}
		fx := n.x / maxAbs
		fy := n.y / maxAbs
		if math.IsNaN(fx) || math.IsInf(fx, 0) || math.IsNaN(fy) || math.IsInf(fy, 0) {
			log.Printf("layout: skipping %q — converged to NaN/Inf (%v, %v)", n.id, n.x, n.y)
			continue
		}
		_, err := pool.Exec(ctx,
			`UPDATE genres SET x = $2, y = $3 WHERE id = $1`,
			n.id, fx, fy,
		)
		if err == nil {
			updated++
		}
	}

	elapsed := time.Since(start)
	log.Printf("layout: finished in %s — %d/%d genres repositioned", elapsed.Round(time.Millisecond), updated, len(nodes))

	return updated, nil
}
