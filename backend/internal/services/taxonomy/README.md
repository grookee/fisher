# Genre taxonomy & adjacency graph pipeline

Builds Fisher's genre-adjacency graph for free, automatically, without
scraping Every Noise or any other third-party genre-map site.

## Sources

- **Wikidata** (SPARQL endpoint, no API key, no cost) - deep, editorially
  curated genre taxonomy: parent/subgenre relations, "influenced by" edges,
  country of origin, aliases.
- **Last.fm** (already integrated in Fisher, free API key) - empirical
  genre-to-genre co-occurrence, built by sampling top artists per genre tag
  and looking at those artists' own top tags.

Both passes merge into Fisher's existing `genres` table (matched
case-insensitively by name) plus three new tables: `genre_relations`
(weighted adjacency edges), `genre_aliases`, and `artist_genres`.

## Running it

```
cd backend
go run ./cmd/ingest-genres
```

Flags: `-skip-wikidata`, `-skip-lastfm`, `-artists-per-genre`, `-delay-ms`,
`-enrich-artist-countries`, `-max-artists-to-enrich`, `-recompute-layout`,
`-expand-geo`, `-artists-per-country`.

## Scheduling (free options)

- A cron job / systemd timer on whatever host runs the backend, e.g. weekly.
- A free GitHub Actions scheduled workflow that runs `go run ./cmd/ingest-genres`
  against your database (set `DATABASE_URL` / `LASTFM_API_KEY` as repo secrets).

## Consuming the graph

- `GET /api/explore/related?id={genre_id}` returns the weighted neighbors of
  a genre (subgenre/influence/co-occurrence edges), sorted by weight.
- `GET /api/explore/missed` (taste-based "missed genres" recommendations)
  prefers the adjacency graph automatically when it's populated, falling
  back to the old parent-sibling heuristic otherwise.

## Genre map x/y coordinates (optional, opt-in)

The `genres.x`/`y` columns that drive the frontend's 2D genre map are
traditionally hand-picked (see `internal/database/genre_seeds.go`), and any
genre discovered later by this pipeline sits at `0,0` until someone curates
it. Passing `-recompute-layout` to `cmd/ingest-genres` runs an extra step
after ingestion: `taxonomy.RecomputeLayout` (in `layout.go`) loads the full
`genre_relations` adjacency graph and runs a lightweight, dependency-free
force-directed (spring) layout over it, then overwrites `x`/`y` for every
genre that has at least one relation, so the map emerges organically from
the real graph instead of relying solely on manual curation. Genres with no
relations are left untouched. This step is opt-in and non-fatal: it's skipped
by default, and if it fails (e.g. the graph exceeds `LayoutOptions.MaxNodes`,
a safety cap against the algorithm's O(n^2) repulsion cost) the ingestion run
still succeeds; only the layout step is logged as failed.

## MusicBrainz artist country enrichment (optional, opt-in)

`internal/services/musicbrainz` is a tiny client for MusicBrainz's public API
(`https://musicbrainz.org/ws/2/`), which is free and requires no API key. It
sets a descriptive `User-Agent` (required by MusicBrainz's usage policy) and
exposes `LookupArtist(name)`, which searches for the best-matching artist and
returns their MBID plus country/area of origin.

`taxonomy.EnrichArtistCountries` bridges this into `artist_genres` (populated
by the Last.fm co-occurrence pass): it finds artists that don't have an
`mbid` yet, looks each one up on MusicBrainz, stores the `mbid`, and folds
their country of origin into `genres.countries` for every genre they're
tagged with - so genre-level regional data emerges automatically from real
artist data over time.

This is a separate, later pass from the main `taxonomy.Run` pipeline because
MusicBrainz's free-tier rate limit (1 request/second, enforced here with a
`time.Sleep` between calls - no parallel requests) makes it too slow to run
inline with ingestion. Pass `-enrich-artist-countries` to `cmd/ingest-genres`
to run it after the main ingestion, and `-max-artists-to-enrich` (default
200) to cap how many artists are looked up in that run, so it can be safely
re-run repeatedly (e.g. nightly) to gradually enrich the whole table without
one run taking hours:

```
cd backend
go run ./cmd/ingest-genres -skip-wikidata -skip-lastfm \
  -enrich-artist-countries -max-artists-to-enrich 200
```

## Ranking layer: "obscure but related" recommendations

`internal/services/ranking` turns the raw graph into actual recommendations.
`ranking.RecommendGenres(ctx, pool, userID, limit)` takes a user's top
`taste_genres`, walks their neighbors in `genre_relations`, and scores each
candidate genre as a weighted blend of:

- **taste similarity** - how strongly connected the candidate is to genres
  the user already likes, weighted by how much they like them
- **scene affinity** - whether the connection is empirical (Last.fm
  co-occurrence / Wikidata "influenced by") rather than just hierarchical
  ("subgenre of")
- **regional novelty** - whether the candidate's `genres.countries` (filled
  in by Wikidata and/or the MusicBrainz enrichment pass) introduce countries
  the user hasn't already explored via their current top genres
- **obscurity** - how sparsely connected the candidate is in the overall
  graph, as a proxy for "still under the radar"
- **friend novelty** - how few of the user's accepted friends already have
  that genre in their own taste profile

Weights live in `ranking.Weights` and default to
`0.40 / 0.20 / 0.15 / 0.15 / 0.10` respectively. The result is exposed via
`GET /api/explore/recommendations`, which returns each candidate genre with
its component scores, composite `score`, and a human-readable `reason`.
Empty results (no taste profile yet, or the graph hasn't been ingested) are
returned as an empty list rather than an error, so callers can fall back to
simpler heuristics.

## Growing artist coverage: per-country charts (optional, opt-in)

`taxonomy.ExpandArtistsByGeo` (in `geo_expand.go`) pulls Last.fm's free
`geo.gettopartists` chart for a curated list of countries (see
`DefaultGeoCountries` - deliberately mixes large and small music markets),
reads each artist's own top tags to map them onto Fisher's genre taxonomy,
and folds their chart country directly into `genres.countries` (no
MusicBrainz lookup needed for these, since the country is already known).

This is the main lever for growing artist coverage *beyond* what genre-tag
sampling alone finds, and for making sure small/underrepresented music
cultures are actually present in the graph. Pass `-expand-geo` to
`cmd/ingest-genres`, with `-artists-per-country` (default 30) to control
volume:

```
cd backend
go run ./cmd/ingest-genres -skip-wikidata -expand-geo -artists-per-country 50
```

Edit `DefaultGeoCountries` directly to add more countries/cultures - it's a
plain `[]string` of Last.fm-recognized country names.

## Populating the track catalog: `cmd/ingest-tracks`

Everything above builds genres and artist↔genre associations, but none of it
touches the `tracks` table - every genre/discovery flow in `handlers/explore.go`
and `handlers/discover.go` does a *live*, ephemeral Spotify search per
request and never persists results. `internal/services/catalog` +
`cmd/ingest-tracks` close that gap: they walk every artist already discovered
in `artist_genres` (by tag sampling and/or `-expand-geo`), resolve them on
Spotify, pull each artist's top tracks (up to 10, free via Spotify's
client-credentials flow - no user auth needed), and upsert them into `tracks`
+ `track_genres`.

Progress is tracked in `artist_track_progress`, so re-running the command
repeatedly (e.g. nightly) incrementally grows the catalog instead of
re-processing the same artists:

```
cd backend
go run ./cmd/ingest-tracks -max-artists 500 -market US
```

Flags: `-max-artists` (artists processed per run), `-market` (ISO 3166-1
alpha-2 code used for Spotify's top-tracks endpoint), `-delay-ms` (throttle
between Spotify calls).

## Recommended full pipeline, in order

```
cd backend
# 1. deep taxonomy + adjacency graph (Wikidata + Last.fm tag co-occurrence)
go run ./cmd/ingest-genres
# 2. widen artist/country coverage via per-country charts
go run ./cmd/ingest-genres -skip-wikidata -expand-geo -artists-per-country 50
# 3. fill in artist countries MusicBrainz didn't get from step 2
go run ./cmd/ingest-genres -skip-wikidata -skip-lastfm -enrich-artist-countries -max-artists-to-enrich 300
# 4. recompute the genre map layout from the finished graph
go run ./cmd/ingest-genres -skip-wikidata -skip-lastfm -recompute-layout
# 5. populate the actual track catalog from every artist discovered above
go run ./cmd/ingest-tracks -max-artists 500
```

Steps 2-5 are all resumable/idempotent (safe to re-run on a schedule with
higher `-max-artists`/`-max-artists-to-enrich` limits over time to keep
growing the catalog for free).

## Possible next steps

- Point `handlers/explore.go`'s genre-browsing endpoints at `tracks` +
  `track_genres` first (falling back to live Spotify search only when a
  genre has no stored tracks yet), so repeat traffic doesn't re-hit Spotify
  for the same genres.
- A "recency" signal in `ranking.RecommendGenres` using `tracks.release_year`
  now that track-level genre links exist, to complete the scoring model.
- Widen `cmd/ingest-tracks` beyond each artist's top 10 tracks (e.g. paging
  through albums) for deeper catalog coverage per artist, at the cost of
  more Spotify requests per artist.
