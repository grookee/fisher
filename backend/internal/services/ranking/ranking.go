package ranking

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Weights control how the composite "obscure but related" score blends
// signals. Exported so callers can tune them without forking the package.
// They're expected to sum to roughly 1.0.
var Weights = struct {
	TasteSimilarity float64
	SceneAffinity   float64
	RegionalNovelty float64
	Obscurity       float64
	FriendNovelty   float64
}{
	TasteSimilarity: 0.20,
	SceneAffinity:   0.20,
	RegionalNovelty: 0.20,
	Obscurity:       0.25,
	FriendNovelty:   0.15,
}

// GenreScore is a single scored genre recommendation, with the individual
// signal components exposed so the frontend (or a curious user) can see why
// something was suggested.
type GenreScore struct {
	GenreID         string   `json:"genre_id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Color           string   `json:"color"`
	X               float64  `json:"x"`
	Y               float64  `json:"y"`
	ParentID        *string  `json:"parent_id"`
	Countries       []string `json:"countries,omitempty"`
	TasteSimilarity float64  `json:"taste_similarity"`
	SceneAffinity   float64  `json:"scene_affinity"`
	RegionalNovelty float64  `json:"regional_novelty"`
	Obscurity       float64  `json:"obscurity"`
	FriendNovelty   float64  `json:"friend_novelty"`
	Score           float64  `json:"score"`
	Reason          string   `json:"reason"`
}

type topGenre struct {
	id     string
	weight float64
}

type candidateEdge struct {
	sourceID     string
	candidateID  string
	relationType string
	weight       float64
}

type genreMeta struct {
	name, description, color string
	x, y                     float64
	parentID                 *string
	countries                []string
}

// RecommendGenres computes an "obscure but related" genre recommendation
// list for a user. Returns two slices: primary recommendations and "deep
// cuts" (genres with no songs in the DB yet). ignoredIDs filters out
// specific genres, and obscureOnly restricts to high-obscurity results.
func RecommendGenres(ctx context.Context, pool *pgxpool.Pool, userID string, limit int, ignoredIDs []string, obscureOnly bool) ([]GenreScore, []GenreScore, error) {
	if limit <= 0 {
		limit = 12
	}

	ignored := make(map[string]bool, len(ignoredIDs))
	for _, id := range ignoredIDs {
		ignored[id] = true
	}

	topGenres, err := loadTopGenres(ctx, pool, userID, 8)
	if err != nil {
		return []GenreScore{}, []GenreScore{}, err
	}
	if len(topGenres) == 0 {
		return []GenreScore{}, []GenreScore{}, nil
	}

	topIDs := make([]string, len(topGenres))
	topWeight := make(map[string]float64, len(topGenres))
	for i, tg := range topGenres {
		topIDs[i] = tg.id
		topWeight[tg.id] = tg.weight
	}

	edges, err := loadCandidateEdges(ctx, pool, userID, topIDs)
	if err != nil {
		return []GenreScore{}, []GenreScore{}, err
	}

	type agg struct {
		tasteSimNumerator float64
		tasteSimDenom     float64
		sceneWeight       float64
		totalWeight       float64
	}
	aggregates := make(map[string]*agg)
	var candidateIDs []string
	for _, e := range edges {
		if ignored[e.candidateID] {
			continue
		}
		a, ok := aggregates[e.candidateID]
		if !ok {
			a = &agg{}
			aggregates[e.candidateID] = a
			candidateIDs = append(candidateIDs, e.candidateID)
		}
		userW := topWeight[e.sourceID]
		a.tasteSimNumerator += e.weight * userW
		a.tasteSimDenom += userW
		a.totalWeight += e.weight
		if e.relationType != "subgenre_of" {
			a.sceneWeight += e.weight
		}
	}

	metaByID, err := loadGenreMeta(ctx, pool, candidateIDs)
	if err != nil {
		return []GenreScore{}, []GenreScore{}, err
	}

	degree, err := loadDegree(ctx, pool, candidateIDs)
	if err != nil {
		degree = map[string]int{}
	}

	globalMaxDegree, err := loadGlobalMaxDegree(ctx, pool)
	if err != nil {
		globalMaxDegree = 1
		for _, d := range degree {
			if d > globalMaxDegree {
				globalMaxDegree = d
			}
		}
	}

	knownCountries, err := loadCountriesForGenres(ctx, pool, topIDs)
	if err != nil {
		knownCountries = map[string]bool{}
	}

	friendAdoption, hasFriends, err := loadFriendGenreAdoption(ctx, pool, userID, candidateIDs)
	if err != nil {
		friendAdoption = map[string]float64{}
		hasFriends = false
	}

	scores := make([]GenreScore, 0, len(candidateIDs))
	for _, cid := range candidateIDs {
		meta, ok := metaByID[cid]
		if !ok {
			continue
		}
		a := aggregates[cid]

		tasteSim := 0.0
		if a.tasteSimDenom > 0 {
			tasteSim = clamp01(a.tasteSimNumerator / a.tasteSimDenom)
		}

		sceneAffinity := 0.3 // hierarchy-only baseline; empirical edges push this up
		if a.totalWeight > 0 {
			sceneAffinity = clamp01(a.sceneWeight / a.totalWeight)
		}

		regionalNovelty := 0.5 // neutral when we don't have country data yet
		if len(meta.countries) > 0 {
			newCount := 0
			for _, c := range meta.countries {
				if !knownCountries[c] {
					newCount++
				}
			}
			regionalNovelty = clamp01(float64(newCount) / float64(len(meta.countries)))
		}

		obscurity := clamp01(1 - float64(degree[cid])/float64(globalMaxDegree))

		friendNovelty := 0.0
		if hasFriends {
			friendNovelty = 1.0
			if v, ok := friendAdoption[cid]; ok {
				friendNovelty = clamp01(1 - v)
			}
		}

		score := Weights.TasteSimilarity*tasteSim +
			Weights.SceneAffinity*sceneAffinity +
			Weights.RegionalNovelty*regionalNovelty +
			Weights.Obscurity*obscurity +
			Weights.FriendNovelty*friendNovelty

		scores = append(scores, GenreScore{
			GenreID:         cid,
			Name:            meta.name,
			Description:     meta.description,
			Color:           meta.color,
			X:               meta.x,
			Y:               meta.y,
			ParentID:        meta.parentID,
			Countries:       meta.countries,
			TasteSimilarity: round2(tasteSim),
			SceneAffinity:   round2(sceneAffinity),
			RegionalNovelty: round2(regionalNovelty),
			Obscurity:       round2(obscurity),
			FriendNovelty:   round2(friendNovelty),
			Score:           round2(score),
			Reason:          buildReason(tasteSim, sceneAffinity, regionalNovelty, obscurity, meta.countries),
		})
	}

	sort.Slice(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })

	var primary, deepCuts []GenreScore
	if obscureOnly {
		for _, s := range scores {
			if s.Obscurity >= 0.6 {
				primary = append(primary, s)
			}
		}
		primary = diversifyResults(primary, 3)
	} else {
		primary = diversifyResults(scores, 3)
	}

	if len(primary) > limit {
		primary = primary[:limit]
	}

	deepCuts, err = loadDeepCutGenres(ctx, pool, userID, ignored, 6)
	if err != nil {
		deepCuts = []GenreScore{}
	}

	return primary, deepCuts, nil
}

func loadTopGenres(ctx context.Context, pool *pgxpool.Pool, userID string, n int) ([]topGenre, error) {
	rows, err := pool.Query(ctx,
		`SELECT genre_id, weight FROM taste_genres WHERE user_id = $1 AND weight > 0.2 ORDER BY weight DESC LIMIT $2`,
		userID, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []topGenre
	for rows.Next() {
		var tg topGenre
		if err := rows.Scan(&tg.id, &tg.weight); err == nil {
			out = append(out, tg)
		}
	}
	return out, nil
}

func loadCandidateEdges(ctx context.Context, pool *pgxpool.Pool, userID string, topIDs []string) ([]candidateEdge, error) {
	hop1, err := loadHopEdges(ctx, pool, userID, topIDs, 1)
	if err != nil {
		return nil, err
	}

	hop2, err := loadHopEdges(ctx, pool, userID, topIDs, 2)
	if err != nil {
		return hop1, nil
	}

	seen := make(map[string]bool, len(hop1))
	edges := make([]candidateEdge, 0, len(hop1)+len(hop2))
	for _, e := range hop1 {
		edges = append(edges, e)
		seen[e.candidateID] = true
	}
	for _, e := range hop2 {
		if !seen[e.candidateID] {
			edges = append(edges, e)
		}
	}
	return edges, nil
}

func loadHopEdges(ctx context.Context, pool *pgxpool.Pool, userID string, topIDs []string, hop int) ([]candidateEdge, error) {
	placeholders := make([]string, len(topIDs))
	args := make([]interface{}, 0, len(topIDs)+1)
	args = append(args, userID)
	for i, id := range topIDs {
		placeholders[i] = "$" + strconv.Itoa(i+2)
		args = append(args, id)
	}
	inClause := strings.Join(placeholders, ",")

	var query string
	if hop == 1 {
		query = `SELECT gr.genre_id, gr.related_genre_id, gr.relation_type, gr.weight
			FROM genre_relations gr
			WHERE gr.genre_id IN (` + inClause + `)
			  AND gr.related_genre_id NOT IN (SELECT genre_id FROM taste_genres WHERE user_id = $1)`
	} else {
		query = `SELECT gr1.related_genre_id, gr2.related_genre_id, gr2.relation_type, gr2.weight * 0.5
			FROM genre_relations gr1
			JOIN genre_relations gr2 ON gr1.related_genre_id = gr2.genre_id
			WHERE gr1.genre_id IN (` + inClause + `)
			  AND gr2.related_genre_id NOT IN (SELECT genre_id FROM taste_genres WHERE user_id = $1)
			  AND gr2.related_genre_id NOT IN (
				SELECT related_genre_id FROM genre_relations WHERE genre_id IN (` + inClause + `)
			  )`
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []candidateEdge
	for rows.Next() {
		var e candidateEdge
		if err := rows.Scan(&e.sourceID, &e.candidateID, &e.relationType, &e.weight); err == nil {
			edges = append(edges, e)
		}
	}
	return edges, nil
}

func diversifyResults(scores []GenreScore, maxPerParent int) []GenreScore {
	if maxPerParent <= 0 {
		maxPerParent = 3
	}
	parentCounts := make(map[string]int)
	var result []GenreScore
	for _, s := range scores {
		parentKey := ""
		if s.ParentID != nil {
			parentKey = *s.ParentID
		}
		if parentCounts[parentKey] < maxPerParent {
			result = append(result, s)
			parentCounts[parentKey]++
		}
	}
	return result
}

func loadGenreMeta(ctx context.Context, pool *pgxpool.Pool, ids []string) (map[string]genreMeta, error) {
	if len(ids) == 0 {
		return map[string]genreMeta{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}
	query := `SELECT id, name, description, color, x, y, parent_id, countries FROM genres WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]genreMeta, len(ids))
	for rows.Next() {
		var id string
		var m genreMeta
		if err := rows.Scan(&id, &m.name, &m.description, &m.color, &m.x, &m.y, &m.parentID, &m.countries); err != nil {
			continue
		}
		out[id] = m
	}
	return out, nil
}

func loadDegree(ctx context.Context, pool *pgxpool.Pool, ids []string) (map[string]int, error) {
	if len(ids) == 0 {
		return map[string]int{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}
	query := `SELECT genre_id, COUNT(*) FROM genre_relations WHERE genre_id IN (` + strings.Join(placeholders, ",") + `) GROUP BY genre_id`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int, len(ids))
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err == nil {
			out[id] = count
		}
	}
	return out, nil
}

func loadCountriesForGenres(ctx context.Context, pool *pgxpool.Pool, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}
	query := `SELECT countries FROM genres WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var countries []string
		if err := rows.Scan(&countries); err != nil {
			continue
		}
		for _, c := range countries {
			out[c] = true
		}
	}
	return out, nil
}

// loadFriendGenreAdoption returns, for each candidate genre, the fraction of
// the user's accepted friends who already have that genre in their own
// taste profile - used to favor genres that are still novel within the
// user's social graph. The second return value indicates whether the user
// has any friends at all.
func loadFriendGenreAdoption(ctx context.Context, pool *pgxpool.Pool, userID string, candidateIDs []string) (map[string]float64, bool, error) {
	if len(candidateIDs) == 0 {
		return map[string]float64{}, false, nil
	}

	friendRows, err := pool.Query(ctx,
		`SELECT friend_id FROM friends WHERE user_id = $1 AND status = 'accepted'
		 UNION
		 SELECT user_id FROM friends WHERE friend_id = $1 AND status = 'accepted'`,
		userID,
	)
	if err != nil {
		return nil, false, err
	}
	var friendIDs []string
	for friendRows.Next() {
		var id string
		if err := friendRows.Scan(&id); err == nil {
			friendIDs = append(friendIDs, id)
		}
	}
	friendRows.Close()

	if len(friendIDs) == 0 {
		return map[string]float64{}, false, nil
	}

	friendPlaceholders := make([]string, len(friendIDs))
	candidatePlaceholders := make([]string, len(candidateIDs))
	args := make([]interface{}, 0, len(friendIDs)+len(candidateIDs))
	for i, id := range friendIDs {
		friendPlaceholders[i] = "$" + strconv.Itoa(i+1)
		args = append(args, id)
	}
	offset := len(friendIDs)
	for i, id := range candidateIDs {
		candidatePlaceholders[i] = "$" + strconv.Itoa(offset+i+1)
		args = append(args, id)
	}

	query := `SELECT genre_id, COUNT(DISTINCT user_id) FROM taste_genres
		WHERE user_id IN (` + strings.Join(friendPlaceholders, ",") + `)
		  AND genre_id IN (` + strings.Join(candidatePlaceholders, ",") + `)
		  AND weight > 0.2
		GROUP BY genre_id`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	out := make(map[string]float64, len(candidateIDs))
	total := float64(len(friendIDs))
	for rows.Next() {
		var genreID string
		var count int
		if err := rows.Scan(&genreID, &count); err == nil {
			out[genreID] = float64(count) / total
		}
	}
	return out, true, nil
}

func loadGlobalMaxDegree(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var max int
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(cnt), 1) FROM (SELECT COUNT(*) AS cnt FROM genre_relations GROUP BY genre_id) sub`,
	).Scan(&max)
	if err != nil {
		return 1, err
	}
	if max < 1 {
		max = 1
	}
	return max, nil
}

// loadDeepCutGenres returns genres that have no artist/track links in the
// database yet — truly unexplored territory. These are scored purely by
// how isolated they are in the genre graph (high obscurity).
func loadDeepCutGenres(ctx context.Context, pool *pgxpool.Pool, userID string, ignored map[string]bool, limit int) ([]GenreScore, error) {
	if limit <= 0 {
		limit = 6
	}

	rows, err := pool.Query(ctx,
		`SELECT g.id, g.name, g.description, g.color, g.x, g.y, g.parent_id, g.countries,
		        COALESCE(rc.cnt, 0) AS rel_count
		 FROM genres g
		 LEFT JOIN (
		   SELECT genre_id, COUNT(*) AS cnt FROM genre_relations GROUP BY genre_id
		 ) rc ON rc.genre_id = g.id
		 WHERE g.id NOT IN (SELECT genre_id FROM taste_genres WHERE user_id = $1)
		   AND g.id NOT IN (
		     SELECT DISTINCT ag.genre_id FROM artist_genres ag
		     JOIN tracks t ON t.artist = ag.artist_name
		   )
		 ORDER BY rel_count ASC, RANDOM()
		 LIMIT $2`,
		userID, limit*3,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	globalMax, err := loadGlobalMaxDegree(ctx, pool)
	if err != nil {
		globalMax = 1
	}

	var results []GenreScore
	for rows.Next() {
		var g GenreScore
		var relCount int
		if err := rows.Scan(&g.GenreID, &g.Name, &g.Description, &g.Color, &g.X, &g.Y, &g.ParentID, &g.Countries, &relCount); err != nil {
			continue
		}
		if ignored[g.GenreID] {
			continue
		}
		g.Obscurity = round2(clamp01(1 - float64(relCount)/float64(globalMax)))
		g.Reason = "a genre nobody on Fisher has explored yet"
		results = append(results, g)
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func buildReason(tasteSim, sceneAffinity, regionalNovelty, obscurity float64, countries []string) string {
	displayCountries := countries
	if len(displayCountries) > 3 {
		displayCountries = displayCountries[:3]
	}
	switch {
	case len(displayCountries) > 0 && regionalNovelty >= 0.6 && obscurity >= 0.5:
		return "an under-the-radar scene from " + strings.Join(displayCountries, "/") + ", close to what you already love"
	case sceneAffinity >= 0.6 && tasteSim >= 0.4:
		return "listeners who like your favorites also gravitate here"
	case obscurity >= 0.6:
		return "closely related, but still mostly undiscovered"
	case tasteSim >= 0.5:
		return "a natural next step from your current taste"
	default:
		return "adjacent to your taste"
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
