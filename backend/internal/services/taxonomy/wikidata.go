package taxonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const wikidataEndpoint = "https://query.wikidata.org/sparql"

// musicGenreQID is Wikidata's item for "music genre" (Q188451).
const musicGenreQID = "wd:Q188451"
const musicGenreFilter = "wdt:P31/wdt:P279* " + musicGenreQID
const wikidataMaxAttempts = 3

type sparqlBindingValue struct {
	Value string `json:"value"`
}

type sparqlResponse struct {
	Results struct {
		Bindings []map[string]sparqlBindingValue `json:"bindings"`
	} `json:"results"`
}

func runSPARQL(ctx context.Context, query string) (*sparqlResponse, error) {
	u := wikidataEndpoint + "?query=" + url.QueryEscape(query) + "&format=json"
	client := &http.Client{Timeout: 90 * time.Second}

	log.Printf("wikidata: executing SPARQL query (%d bytes)...", len(query))
	start := time.Now()

	var lastErr error
	for attempt := 1; attempt <= wikidataMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		// Wikidata's usage policy asks bulk/bot clients to identify themselves.
		req.Header.Set("User-Agent", "FisherGenreTaxonomyBot/1.0 (self-hosted music discovery app)")
		req.Header.Set("Accept", "application/sparql-results+json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("wikidata request failed: %w", err)
			log.Printf("wikidata: attempt %d/%d failed: %v", attempt, wikidataMaxAttempts, err)
		} else {
			if resp.StatusCode == http.StatusOK {
				var out sparqlResponse
				decodeErr := json.NewDecoder(resp.Body).Decode(&out)
				resp.Body.Close()
				if decodeErr != nil {
					return nil, fmt.Errorf("failed to decode wikidata response: %w", decodeErr)
				}
				elapsed := time.Since(start)
				log.Printf("wikidata: query completed in %s, returned %d results", elapsed.Round(time.Millisecond), len(out.Results.Bindings))
				return &out, nil
			}

			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			status := resp.StatusCode
			retryAfter := resp.Header.Get("Retry-After")
			resp.Body.Close()

			message := strings.TrimSpace(string(body))
			if message == "" {
				message = "empty response body"
			}
			lastErr = fmt.Errorf("wikidata returned status %d: %s", status, message)
			log.Printf("wikidata: attempt %d/%d returned status %d: %s", attempt, wikidataMaxAttempts, status, message)

			// Retry only on transient endpoint failures / throttling.
			if status != http.StatusTooManyRequests && status < 500 {
				return nil, lastErr
			}

			if attempt < wikidataMaxAttempts {
				delay := time.Duration(attempt*2) * time.Second
				if retryAfter != "" {
					if seconds, parseErr := strconv.Atoi(strings.TrimSpace(retryAfter)); parseErr == nil && seconds > 0 {
						delay = time.Duration(seconds) * time.Second
					}
				}
				log.Printf("wikidata: retrying in %s...", delay.Round(time.Second))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
				}
				continue
			}
		}

		if attempt < wikidataMaxAttempts {
			delay := time.Duration(attempt*2) * time.Second
			log.Printf("wikidata: retrying in %s...", delay.Round(time.Second))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return nil, lastErr
}

func qidFromURI(uri string) string {
	idx := strings.LastIndex(uri, "/")
	if idx == -1 {
		return uri
	}
	return uri[idx+1:]
}

func appendUnique(list []string, v string) []string {
	for _, existing := range list {
		if strings.EqualFold(existing, v) {
			return list
		}
	}
	return append(list, v)
}

// FetchGenreTaxonomy queries Wikidata's public SPARQL endpoint (free, no API
// key) for the "music genre" tree: every genre directly tagged as an
// instance of music genre, its parent genre ("subclass of"), the genres
// that influenced it, its country of origin, and known aliases. This is the
// backbone of Fisher's deep genre taxonomy + adjacency graph, built entirely
// from an open, editorially-curated dataset instead of scraping any
// third-party genre-map site.
func FetchGenreTaxonomy(ctx context.Context) ([]GenreNode, []Relation, error) {
	start := time.Now()
	nodes := map[string]*GenreNode{}

	getOrCreate := func(qid, label string) *GenreNode {
		n, ok := nodes[qid]
		if !ok {
			n = &GenreNode{QID: qid, Name: label}
			nodes[qid] = n
		} else if n.Name == "" {
			n.Name = label
		}
		return n
	}

	var relations []Relation

	// 1) genres + parent genre ("subclass of") + country of origin.
	// Uses P31/P279* to include genres that are typed through a superclass of
	// "music genre", not only direct instance-of statements.
	log.Printf("wikidata: fetching genre taxonomy (genres + parents + countries)...")
	parentQuery := fmt.Sprintf(`SELECT ?genre ?genreLabel ?parent ?parentLabel ?country ?countryLabel WHERE {
		?genre %s .
		OPTIONAL { ?genre wdt:P279 ?parent . ?parent %s . }
		OPTIONAL { ?genre wdt:P495 ?country . }
		SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
	} LIMIT 30000`, musicGenreFilter, musicGenreFilter)

	resp, err := runSPARQL(ctx, parentQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("wikidata taxonomy query: %w", err)
	}
	for _, b := range resp.Results.Bindings {
		genreURI, ok := b["genre"]
		if !ok {
			continue
		}
		qid := qidFromURI(genreURI.Value)
		label := b["genreLabel"].Value
		if label == "" {
			continue
		}
		node := getOrCreate(qid, label)

		if country, ok := b["countryLabel"]; ok && country.Value != "" {
			node.Countries = appendUnique(node.Countries, country.Value)
		}
		if _, hasParent := b["parent"]; hasParent {
			parentLabel := b["parentLabel"].Value
			if parentLabel != "" && !strings.EqualFold(parentLabel, label) {
				node.Parents = appendUnique(node.Parents, parentLabel)
				relations = append(relations, Relation{
					From: label, To: parentLabel, Type: "subgenre_of", Weight: 0.9, Source: "wikidata",
				})
			}
		}
	}

	log.Printf("wikidata: processed %d genres, %d parent relations from first query", len(nodes), len(relations))

	// 2) influenced-by edges (restricted to genre-to-genre influence only).
	log.Printf("wikidata: fetching influenced-by relations...")
	influenceQuery := fmt.Sprintf(`SELECT ?genreLabel ?influencedLabel WHERE {
		?genre %s .
		?genre wdt:P737 ?influenced .
		?influenced %s .
		SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
	} LIMIT 30000`, musicGenreFilter, musicGenreFilter)

	if resp, err := runSPARQL(ctx, influenceQuery); err == nil {
		influenceCount := 0
		for _, b := range resp.Results.Bindings {
			label := b["genreLabel"].Value
			infLabel := b["influencedLabel"].Value
			if label == "" || infLabel == "" || strings.EqualFold(label, infLabel) {
				continue
			}
			relations = append(relations, Relation{
				From: label, To: infLabel, Type: "influenced_by", Weight: 0.6, Source: "wikidata",
			})
			influenceCount++
		}
		log.Printf("wikidata: processed %d influenced-by relations", influenceCount)
	}

	// 3) aliases (alternate / native-language names) - useful for matching
	// genre strings that come back from Spotify/Last.fm later.
	log.Printf("wikidata: fetching genre aliases...")
	aliasQuery := fmt.Sprintf(`SELECT ?genreLabel ?alias WHERE {
		?genre %s .
		?genre skos:altLabel ?alias .
		FILTER(LANG(?alias) = "en")
		SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
	} LIMIT 30000`, musicGenreFilter)

	if resp, err := runSPARQL(ctx, aliasQuery); err == nil {
		aliasCount := 0
		labelIndex := make(map[string]*GenreNode, len(nodes))
		for _, n := range nodes {
			labelIndex[strings.ToLower(n.Name)] = n
		}
		for _, b := range resp.Results.Bindings {
			label := b["genreLabel"].Value
			alias := b["alias"].Value
			if label == "" || alias == "" {
				continue
			}
			if n, ok := labelIndex[strings.ToLower(label)]; ok {
				n.Aliases = appendUnique(n.Aliases, alias)
				aliasCount++
			}
		}
		log.Printf("wikidata: processed %d aliases", aliasCount)
	}

	out := make([]GenreNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, *n)
	}

	elapsed := time.Since(start)
	log.Printf("wikidata: taxonomy fetch complete in %s — %d genres, %d relations", elapsed.Round(time.Second), len(out), len(relations))

	return out, relations, nil
}
