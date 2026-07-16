package taxonomy

// GenreNode represents a single genre discovered from an external taxonomy
// source (currently Wikidata).
type GenreNode struct {
	QID       string   // Wikidata Q-id, e.g. "Q11401"; empty if not from Wikidata
	Name      string   // canonical English label
	Aliases   []string // alternate / native-language names
	Parents   []string // names of broader/parent genres ("subgenre of")
	Countries []string // country-of-origin labels
}

// Relation is a weighted edge in the genre adjacency graph. Persist() always
// stores relations symmetrically (both directions), since the product need
// is "find neighboring genres", not strict hierarchy (hierarchy is already
// tracked separately via genres.parent_id for the hand-seeded tree).
type Relation struct {
	From   string
	To     string
	Type   string // "subgenre_of" | "influenced_by" | "cooccurs_with"
	Weight float64
	Source string // "wikidata" | "lastfm"
}
