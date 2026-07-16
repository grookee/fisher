package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

var spotifyToOurGenre = map[string]string{
	"rock": "Rock", "alternative rock": "Rock", "classic rock": "Rock", "hard rock": "Rock",
	"punk": "Punk", "punk rock": "Punk", "indie rock": "Indie", "alternative": "Indie",
	"grunge": "Rock", "metal": "Metal", "heavy metal": "Metal", "post-rock": "Post-Rock",
	"progressive rock": "Progressive Rock", "psychedelic rock": "Psychedelic Rock", "garage rock": "Garage Rock",
	"emo": "Punk", "post-hardcore": "Post-Hardcore", "hardcore": "Hardcore Punk", "hardcore punk": "Hardcore Punk",
	"metalcore": "Metal", "death metal": "Death Metal", "black metal": "Black Metal", "doom metal": "Doom Metal",
	"thrash metal": "Thrash Metal", "power metal": "Power Metal", "symphonic metal": "Symphonic Metal",
	"folk metal": "Folk Metal", "sludge metal": "Sludge Metal", "djent": "Djent",
	"pop": "Pop", "indie pop": "Indie Pop", "synth pop": "Synth Pop",
	"synth-pop": "Synth Pop", "art pop": "Art Pop", "dream pop": "Dream Pop", "chamber pop": "Pop",
	"dance pop": "Dance Pop", "electropop": "Electronic", "hyperpop": "Hyperpop", "k-pop": "K-Pop",
	"j-pop": "J-Pop", "power pop": "Rock",
	"hip hop": "Hip Hop", "hip-hop": "Hip Hop", "rap": "Hip Hop", "trap": "Trap",
	"drill": "Drill", "gangsta rap": "Gangsta Rap", "conscious rap": "Conscious Rap",
	"alternative hip hop": "Alternative Hip Hop", "grime": "Grime", "trip-hop": "Electronic",
	"trip hop": "Electronic", "boom bap": "Boom Bap", "cloud rap": "Cloud Rap", "phonk": "Phonk",
	"jersey club": "Jersey Club",
	"electronic": "Electronic", "edm": "Electronic", "house": "Electronic",
	"deep house": "Deep House", "tech house": "Tech House", "techno": "Techno",
	"trance": "Trance", "dubstep": "Dubstep", "drum and bass": "Drum and Bass",
	"drum & bass": "Drum and Bass", "liquid drum and bass": "Liquid Drum and Bass",
	"liquid drum & bass": "Liquid Drum and Bass",
	"ambient": "Ambient", "idm": "IDM", "breakbeat": "Breakbeat",
	"future bass": "Future Bass", "downtempo": "Ambient", "electronica": "Electronic",
	"industrial": "Electronic", "synthwave": "Synthwave", "vaporwave": "Synthwave",
	"hardstyle": "Electronic", "chiptune": "Electronic", "uk garage": "UK Garage",
	"footwork": "Footwork", "jungle": "Jungle",
	"r&b": "R&B", "rnb": "R&B", "contemporary r&b": "Contemporary R&B",
	"alternative r&b": "Alternative R&B", "pbr&b": "Alternative R&B",
	"neo soul": "Neo-Soul", "neo-soul": "Neo-Soul", "soul": "Soul",
	"northern soul": "Northern Soul", "psychedelic soul": "Psychedelic Soul",
	"southern soul": "Southern Soul",
	"funk": "Funk", "disco": "Disco", "p-funk": "P-Funk", "g-funk": "G-Funk", "boogie": "Boogie",
	"jazz": "Jazz", "bebop": "Bebop", "cool jazz": "Cool Jazz", "free jazz": "Free Jazz",
	"jazz fusion": "Jazz Fusion", "swing": "Swing", "smooth jazz": "Smooth Jazz", "acid jazz": "Acid Jazz",
	"modal jazz": "Modal Jazz",
	"classical": "Classical", "orchestral": "Classical", "opera": "Opera",
	"baroque": "Baroque", "contemporary classical": "Modern Classical", "neoclassical": "Modern Classical",
	"minimalism": "Minimalism", "chamber music": "Chamber Music", "choral": "Choral",
	"country": "Country", "americana": "Americana", "country rock": "Country Rock",
	"bluegrass": "Bluegrass", "folk": "Folk", "indie folk": "Indie Folk",
	"singer-songwriter": "Singer-Songwriter", "outlaw country": "Outlaw Country",
	"alt-country": "Alt-Country", "country pop": "Country Pop",
	"blues": "Blues", "delta blues": "Delta Blues", "electric blues": "Electric Blues",
	"chicago blues": "Chicago Blues", "texas blues": "Texas Blues",
	"reggae": "Reggae", "ska": "Ska", "dancehall": "Dancehall", "dub": "Dub",
	"rocksteady": "Rocksteady", "roots reggae": "Roots Reggae",
	"reggaeton": "Reggaeton", "latin": "Latin", "salsa": "Salsa", "bachata": "Bachata",
	"cumbia": "Cumbia", "bossa nova": "Bossa Nova", "samba": "Samba", "latin pop": "Latin",
	"latin rock": "Latin", "flamenco": "Flamenco", "merengue": "Merengue", "dembow": "Dembow",
	"world": "World", "world music": "World", "celtic": "Folk",
	"afrobeat": "Afrobeat", "highlife": "Highlife", "soca": "Soca", "kizomba": "Kizomba",
	"afro house": "Afro House",
	"gospel": "Gospel", "spiritual": "Gospel", "christian": "Gospel",
	"southern gospel": "Gospel",
	"drone": "Drone", "dark ambient": "Dark Ambient", "ambient pop": "Ambient Pop",
	"space music": "Space Music",
	"shoegaze": "Shoegaze", "lo-fi": "Lo-Fi", "bedroom pop": "Bedroom Pop",
	"noise pop": "Noise Pop", "post-punk": "Post-Punk Revival",
	"neofolk": "Neofolk", "traditional folk": "Traditional Folk",
	"pop punk": "Pop Punk", "skate punk": "Skate Punk", "anarcho-punk": "Anarcho-Punk",
	"crust punk": "Crust Punk", "stoner rock": "Stoner Rock", "blues rock": "Blues Rock",
	"post-metal": "Post-Metal",
}

var artistGenreMap = map[string][]string{
	"kendrick lamar":                {"hip hop", "rap"},
	"kanye west":                    {"hip hop", "rap"},
	"j cole":                        {"hip hop", "rap"},
	"drake":                         {"hip hop", "rap", "r&b"},
	"travis scott":                  {"hip hop", "rap", "trap"},
	"future":                        {"hip hop", "rap", "trap"},
	"21 savage":                     {"hip hop", "rap", "trap"},
	"metro boomin":                  {"hip hop", "rap", "trap"},
	"playboi carti":                 {"hip hop", "rap", "trap"},
	"lil uzi vert":                  {"hip hop", "rap", "trap"},
	"young thug":                    {"hip hop", "rap", "trap"},
	"gunna":                         {"hip hop", "rap", "trap"},
	"lil baby":                      {"hip hop", "rap", "trap"},
	"offset":                        {"hip hop", "rap", "trap"},
	"quavo":                         {"hip hop", "rap", "trap"},
	"takeoff":                       {"hip hop", "rap", "trap"},
	"migos":                         {"hip hop", "rap", "trap"},
	"lil wayne":                     {"hip hop", "rap"},
	"eminem":                        {"hip hop", "rap"},
	"jay-z":                         {"hip hop", "rap"},
	"nas":                           {"hip hop", "rap"},
	"snoop dogg":                    {"hip hop", "rap"},
	"dr dre":                        {"hip hop", "rap"},
	"tyler the creator":             {"hip hop", "rap", "alternative"},
	"frank ocean":                   {"r&b", "soul", "pop"},
	"the weeknd":                    {"r&b", "pop"},
	"bruno mars":                    {"pop", "funk", "r&b"},
	"doja cat":                      {"pop", "hip hop", "r&b"},
	"taylor swift":                  {"pop"},
	"billie eilish":                 {"pop", "electropop"},
	"olivia rodrigo":                {"pop"},
	"dua lipa":                      {"pop", "dance pop"},
	"harry styles":                  {"pop", "rock"},
	"ariana grande":                 {"pop", "r&b"},
	"beyonce":                       {"pop", "r&b"},
	"rihanna":                       {"pop", "r&b"},
	"adele":                         {"pop", "soul"},
	"ed sheeran":                    {"pop", "folk"},
	"post malone":                   {"pop", "hip hop"},
	"imagine dragons":               {"rock", "alternative"},
	"coldplay":                      {"rock", "alternative"},
	"arctic monkeys":                {"rock", "indie rock"},
	"the strokes":                   {"rock", "indie rock"},
	"phoebe bridgers":               {"indie rock", "folk", "singer-songwriter"},
	"mitski":                        {"indie rock", "alternative"},
	"mac demarco":                   {"indie rock", "dream pop"},
	"beach house":                   {"dream pop", "indie"},
	"bon iver":                      {"folk", "indie folk"},
	"father john misty":             {"folk", "indie rock"},
	"hozier":                        {"folk", "indie rock"},
	"florence + the machine":        {"indie rock", "pop"},
	"lana del rey":                  {"pop", "indie"},
	"lord huron":                    {"folk", "indie"},
	"gregory alan isakov":           {"folk", "singer-songwriter"},
	"iron & wine":                   {"folk", "singer-songwriter"},
	"the lumineers":                 {"folk", "indie folk"},
	"mumford & sons":                {"folk", "indie folk"},
	"the head and the heart":        {"folk", "indie"},
	"joey bada$$":                   {"hip hop", "rap"},
	"mac miller":                    {"hip hop", "rap"},
	"kid cudi":                      {"hip hop", "rap", "alternative"},
	"asap rocky":                    {"hip hop", "rap"},
	"denzel curry":                  {"hip hop", "rap"},
	"jpegmafia":                     {"hip hop", "rap", "experimental"},
	"danny brown":                   {"hip hop", "rap"},
	"run the jewels":                {"hip hop", "rap"},
	"mf doom":                       {"hip hop", "rap"},
	"madvillain":                    {"hip hop", "rap"},
	"j dilla":                       {"hip hop", "rap"},
	"tribe called quest":            {"hip hop", "rap", "jazz"},
	"de la soul":                    {"hip hop", "rap"},
	"gang starr":                    {"hip hop", "rap"},
	"wu-tang clan":                  {"hip hop", "rap"},
	"notorious b.i.g.":              {"hip hop", "rap"},
	"tupac":                         {"hip hop", "rap"},
	"lil nas x":                     {"pop", "hip hop"},
	"joji":                          {"r&b", "pop", "lo-fi"},
	"steve lacy":                    {"r&b", "funk"},
	"daniel caesar":                 {"r&b", "soul"},
	"h.e.r.":                        {"r&b", "soul"},
	"jhene aiko":                    {"r&b", "soul"},
	"sza":                           {"r&b", "soul"},
	"solange":                       {"r&b", "soul", "funk"},
	"erykah badu":                   {"soul", "neo soul", "r&b"},
	"d'angelo":                      {"soul", "neo soul", "r&b"},
	"lauryn hill":                   {"soul", "hip hop", "r&b"},
	"amy winehouse":                 {"soul", "r&b", "jazz"},
	"sam smith":                     {"pop", "soul"},
	"tame impala":                   {"psychedelic rock", "indie", "electronic"},
	"glass animals":                 {"indie", "electronic", "psychedelic rock"},
	"alt-j":                         {"indie rock", "alternative"},
	"mgmt":                          {"indie rock", "psychedelic rock", "electronic"},
	"gorillaz":                      {"alternative", "hip hop", "electronic"},
	"daft punk":                     {"electronic", "funk", "disco"},
	"justice":                       {"electronic", "funk"},
	"flume":                         {"electronic", "future bass"},
	"odesza":                        {"electronic"},
	"disclosure":                    {"electronic", "house"},
	"calvin harris":                 {"electronic", "edm", "pop"},
	"avicii":                        {"electronic", "edm"},
	"martin garrix":                 {"electronic", "edm"},
	"marshmello":                    {"electronic", "edm"},
	"diplo":                         {"electronic", "edm", "hip hop"},
	"skrillex":                      {"electronic", "dubstep", "edm"},
	"deadmau5":                      {"electronic", "house", "techno"},
	"aphex twin":                    {"electronic", "idm", "ambient"},
	"brian eno":                     {"ambient", "electronic"},
	"nils frahm":                    {"classical", "ambient", "electronic"},
	"max richter":                   {"classical", "contemporary classical"},
	"olafur arnalds":                {"classical", "contemporary classical"},
	"joep beving":                   {"classical", "contemporary classical"},
	"ludovico einaudi":              {"classical", "contemporary classical"},
	"hans zimmer":                   {"classical", "orchestral"},
	"john williams":                 {"classical", "orchestral"},
	"miles davis":                   {"jazz"},
	"john coltrane":                 {"jazz"},
	"billie holiday":                {"jazz", "blues"},
	"ella fitzgerald":               {"jazz"},
	"louis armstrong":               {"jazz"},
	"thelonious monk":               {"jazz", "bebop"},
	"charles mingus":                {"jazz"},
	"duke ellington":                {"jazz"},
	"count basie":                   {"jazz"},
	"herbie hancock":                {"jazz", "jazz fusion"},
	"robert glasper":                {"jazz", "jazz fusion", "neo soul"},
	"kamasi washington":             {"jazz", "jazz fusion"},
	"espers":                        {"jazz", "psychedelic rock", "soul"},
	"bob dylan":                     {"folk", "singer-songwriter", "rock"},
	"joni mitchell":                 {"folk", "singer-songwriter"},
	"leonard cohen":                 {"folk", "singer-songwriter"},
	"nick drake":                    {"folk", "singer-songwriter"},
	"simon & garfunkel":             {"folk", "rock"},
	"cat stevens":                   {"folk", "singer-songwriter"},
	"johnny cash":                   {"country", "folk"},
	"dolly parton":                  {"country", "folk"},
	"willie nelson":                 {"country"},
	"merle haggard":                 {"country"},
	"patsy cline":                   {"country"},
	"hank williams":                 {"country", "folk"},
	"chris stapleton":               {"country", "blues rock"},
	"sturgill simpson":              {"country", "rock"},
	"jason isbell":                  {"country", "americana"},
	"tyler childers":                {"country", "folk"},
	"colter wall":                   {"country"},
	"led zeppelin":                  {"rock", "hard rock", "blues"},
	"pink floyd":                    {"rock", "progressive rock", "psychedelic rock"},
	"queen":                         {"rock"},
	"the beatles":                   {"rock", "pop"},
	"rolling stones":                {"rock", "blues rock"},
	"david bowie":                   {"rock", "pop", "art pop"},
	"prince":                        {"funk", "pop", "rock"},
	"michael jackson":               {"pop", "r&b", "funk"},
	"stevie wonder":                 {"soul", "funk", "pop"},
	"marvin gaye":                   {"soul", "r&b", "funk"},
	"aretha franklin":               {"soul", "gospel"},
	"james brown":                   {"funk", "soul"},
	"fleetwood mac":                 {"rock", "pop"},
	"eagles":                        {"rock"},
	"nirvana":                       {"grunge", "rock"},
	"pearl jam":                     {"grunge", "rock"},
	"soundgarden":                   {"grunge", "rock"},
	"alice in chains":               {"grunge", "rock"},
	"metallica":                     {"metal", "thrash metal"},
	"megadeth":                      {"metal", "thrash metal"},
	"slayer":                        {"metal", "thrash metal"},
	"anthrax":                       {"metal", "thrash metal"},
	"iron maiden":                   {"metal", "heavy metal"},
	"black sabbath":                 {"metal", "heavy metal"},
	"judas priest":                  {"metal", "heavy metal"},
	"motorhead":                     {"metal", "rock"},
	"tool":                          {"metal", "progressive rock", "alternative"},
	"system of a down":              {"metal", "alternative"},
	"rammstein":                     {"metal", "industrial"},
	"linkin park":                   {"rock", "alternative", "electronic"},
	"foo fighters":                  {"rock"},
	"deftones":                      {"metal", "alternative rock"},
	"the smiths":                    {"rock", "indie rock"},
	"joy division":                  {"rock", "post-punk"},
	"radiohead":                     {"alternative rock", "art rock", "electronic"},
	"the cure":                      {"rock", "post-punk"},
	"depeche mode":                  {"electronic", "synth-pop"},
	"new order":                     {"electronic", "post-punk", "synth-pop"},
	"talking heads":                 {"rock", "new wave"},
	"rem":                           {"alternative rock", "rock"},
	"u2":                            {"rock"},
	"bob marley":                    {"reggae"},
	"peter tosh":                    {"reggae"},
	"jimmy cliff":                   {"reggae", "ska"},
	"damian marley":                 {"reggae", "hip hop"},
	"tiken jah fakoly":              {"reggae", "world"},
	"alpha blondy":                  {"reggae"},
	"bad bunny":                     {"latin", "reggaeton", "latin pop"},
	"j balvin":                      {"latin", "reggaeton", "latin pop"},
	"daddy yankee":                  {"latin", "reggaeton"},
	"ozuna":                         {"latin", "reggaeton"},
	"rosalia":                       {"latin", "flamenco", "pop"},
	"shakira":                       {"latin pop", "pop"},
	"luis miguel":                   {"latin", "latin pop"},
	"celia cruz":                    {"latin", "salsa"},
	"buena vista social club":       {"latin", "world"},
	"caetano veloso":                {"latin", "bossa nova", "world"},
	"antonio carlos jobim":          {"latin", "bossa nova"},
	"joao gilberto":                 {"latin", "bossa nova"},
	"fela kuti":                     {"world", "funk", "jazz"},
	"burna boy":                     {"world", "afrobeat", "reggae"},
	"wizkid":                        {"world", "afrobeat"},
	"davido":                        {"world", "afrobeat"},
	"youssou n'dour":                {"world"},
	"ali farka toure":               {"world", "blues"},
	"cesaria evora":                 {"world", "latin"},
	"anoushka shankar":              {"world", "classical"},
	"ritesh":                        {"world"},
	"sigur ros":                     {"ambient", "post-rock", "indie"},
	"m83":                           {"electronic", "dream pop", "ambient"},
	"boarding school":               {"electronic", "indie"},
	"moderat":                       {"electronic", "techno"},
	"modeselektor":                  {"electronic", "techno"},
	"leftfield":                     {"electronic", "ambient", "techno"},
	"the chemical brothers":         {"electronic", "big beat"},
	"prodigy":                       {"electronic", "big beat"},
	"fatboy slim":                   {"electronic", "big beat"},
	"basement jaxx":                 {"electronic", "house"},
	"the avalanches":                {"electronic", "plunderphonics"},
	"boards of canada":              {"electronic", "ambient", "idm"},
	"flying lotus":                  {"electronic", "idm", "beat"},
	"thundercat":                    {"soul", "funk", "electronic"},
	"anderson paak":                 {"soul", "funk", "hip hop", "r&b"},
	"kaytranada":                    {"electronic", "funk", "soul"},
	"tom misch":                     {"soul", "jazz", "electronic"},
	"jordan rakei":                  {"soul", "r&b", "electronic"},
	"sampa the great":              {"hip hop", "world"},
	"arcade fire":                   {"indie rock", "alternative"},
	"the national":                  {"indie rock", "alternative"},
	"modest mouse":                  {"indie rock", "alternative"},
	"brand new":                     {"emo", "alternative rock"},
	"taking back sunday":            {"emo", "punk rock"},
	"my chemical romance":           {"emo", "punk rock", "alternative"},
	"fall out boy":                  {"emo", "pop punk"},
	"paramore":                      {"emo", "pop punk"},
	"blink-182":                     {"pop punk", "punk rock"},
	"green day":                     {"punk rock", "alternative"},
	"the offspring":                 {"punk rock", "ska punk"},
	"rancid":                        {"punk rock", "ska punk"},
	"operation ivy":                 {"ska punk", "punk"},
	"streetlight manifesto":         {"ska punk", "punk"},
	"less than jake":                {"ska punk", "punk"},
	"mighty mighty bosstones":       {"ska punk", "alternative"},
	"pogány induló":                 {"hip hop", "rap"},
	"ekhoe":                         {"hip hop", "rap"},
	"beton.hofi":                    {"hip hop", "rap"},
	"óthvar pestis":                 {"hip hop", "rap"},
	"co lee":                        {"hip hop", "rap"},
	"carson coma":                   {"indie rock", "alternative"},
	"azahriah":                      {"pop", "hip hop"},
	"krúbi":                         {"hip hop", "rap"},
	"ken carson":                    {"hip hop", "rap", "trap"},
	"desh":                          {"hip hop", "rap"},
	"6363":                          {"hip hop", "rap"},
	"szalai":                        {"hip hop", "rap"},
	"baddie":                        {"pop"},
	"lil tecca":                     {"hip hop", "rap", "trap"},
	"youngboy never broke again":    {"hip hop", "rap"},
	"trippie redd":                  {"hip hop", "rap", "trap"},
	"juice wrld":                    {"hip hop", "rap", "emo"},
	"xxx":                           {"hip hop", "rap", "emo"},
	"lil peep":                      {"emo", "hip hop", "rap"},
	"lil pump":                      {"hip hop", "rap", "trap"},
	"smokepurpp":                    {"hip hop", "rap", "trap"},
	"da baby":                       {"hip hop", "rap", "trap"},
	"megan thee stallion":           {"hip hop", "rap"},
	"cardi b":                       {"hip hop", "rap"},
	"nicki minaj":                   {"hip hop", "rap", "pop"},
	"ice spice":                     {"hip hop", "rap"},
	"central cee":                   {"hip hop", "rap", "grime"},
	"j hus":                         {"hip hop", "rap", "grime"},
	"stormzy":                       {"hip hop", "rap", "grime"},
	"dave":                          {"hip hop", "rap"},
	"skepta":                        {"hip hop", "rap", "grime"},
	"aj tracey":                     {"hip hop", "rap", "grime"},
}

type TopArtist struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Genres []string `json:"genres"`
	Images []struct {
		URL string `json:"url"`
	} `json:"images"`
}

type TopTrackItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"release_date"`
		Images      []struct {
			URL string `json:"url"`
		} `json:"images"`
	} `json:"album"`
	PreviewURL string `json:"preview_url"`
	URI        string `json:"uri"`
	DurationMs int `json:"duration_ms"`
}

type AudioFeaturesResponse struct {
	Danceability     float64 `json:"danceability"`
	Energy           float64 `json:"energy"`
	Valence          float64 `json:"valence"`
	Acousticness     float64 `json:"acousticness"`
	Instrumentalness float64 `json:"instrumentalness"`
	Speechiness      float64 `json:"speechiness"`
	Liveness         float64 `json:"liveness"`
	Tempo            float64 `json:"tempo"`
}

type TrackResult struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Artist           string  `json:"artist"`
	Album            string  `json:"album"`
	AlbumArt         string  `json:"album_art_url"`
	Preview          string  `json:"preview_url"`
	URI              string  `json:"uri"`
	Duration         int     `json:"duration_ms"`
	ReleaseYear      int     `json:"release_year,omitempty"`
	Danceability     float64 `json:"danceability,omitempty"`
	Energy           float64 `json:"energy,omitempty"`
	Valence          float64 `json:"valence,omitempty"`
	Acousticness     float64 `json:"acousticness,omitempty"`
	Instrumentalness float64 `json:"instrumentalness,omitempty"`
	Speechiness      float64 `json:"speechiness,omitempty"`
	Tempo            float64 `json:"tempo,omitempty"`
}

type AnalyzeResult struct {
	TopArtists    []TopArtist            `json:"top_artists"`
	TopTracks     []TrackResult          `json:"top_tracks"`
	GenreCount    map[string]int         `json:"-"`
	AudioFeatures *AudioFeaturesResponse `json:"audio_features,omitempty"`
}

func AnalyzeTasteExtended(accessToken, timeRange string) (*AnalyzeResult, error) {
	hc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}))

	result := &AnalyzeResult{
		GenreCount: make(map[string]int),
	}

	if timeRange == "" {
		timeRange = "medium_term"
	}

	topArtistsResp, err := fetchTopArtists(hc, accessToken, timeRange)
	if err != nil {
		return nil, err
	}

	result.TopArtists = topArtistsResp.Items

	for _, a := range topArtistsResp.Items {
		name := strings.ToLower(a.Name)
		genres, found := artistGenreMap[name]
		if found {
			for _, g := range genres {
				result.GenreCount[g]++
			}
		}
	}

	topTracksResp, err := fetchTopTracks(hc, accessToken, timeRange)
	if err == nil {
		for _, t := range topTracksResp.Items {
			artistNames := make([]string, len(t.Artists))
			for i, a := range t.Artists {
				artistNames[i] = a.Name
			}
			albumArt := ""
			if len(t.Album.Images) > 0 {
				albumArt = t.Album.Images[0].URL
			}
			result.TopTracks = append(result.TopTracks, TrackResult{
				ID:          t.ID,
				Title:       t.Name,
				Artist:      strings.Join(artistNames, ", "),
				Album:       t.Album.Name,
				AlbumArt:    albumArt,
				Preview:     t.PreviewURL,
				URI:         t.URI,
				Duration:    t.DurationMs,
				ReleaseYear: extractYear(t.Album.ReleaseDate),
			})
		}

		ids := make([]string, len(topTracksResp.Items))
		for i, t := range topTracksResp.Items {
			ids[i] = t.ID
		}
		if len(ids) > 0 {
			perTrack, err := fetchPerTrackAudioFeatures(accessToken, ids)
			if err == nil {
				for i := range result.TopTracks {
					if af, ok := perTrack[result.TopTracks[i].ID]; ok {
						result.TopTracks[i].Danceability = af.Danceability
						result.TopTracks[i].Energy = af.Energy
						result.TopTracks[i].Valence = af.Valence
						result.TopTracks[i].Acousticness = af.Acousticness
						result.TopTracks[i].Instrumentalness = af.Instrumentalness
						result.TopTracks[i].Speechiness = af.Speechiness
						result.TopTracks[i].Tempo = af.Tempo
					}
				}
			}

			af, err := fetchAudioFeaturesWithToken(accessToken, ids)
			if err == nil {
				result.AudioFeatures = af
			}
		}
	}

	if len(result.GenreCount) == 0 && result.AudioFeatures != nil {
		af := result.AudioFeatures
		if af.Speechiness > 0.33 {
			result.GenreCount["hip hop"] += 3
			result.GenreCount["rap"] += 2
		}
		if af.Danceability > 0.7 && af.Energy > 0.7 {
			result.GenreCount["electronic"] += 2
			result.GenreCount["edm"] += 1
		}
		if af.Acousticness > 0.5 && af.Energy < 0.5 {
			result.GenreCount["folk"] += 2
			result.GenreCount["acoustic"] += 2
		}
		if af.Energy > 0.7 && af.Valence > 0.5 {
			result.GenreCount["pop"] += 2
		}
		if af.Instrumentalness > 0.5 {
			result.GenreCount["ambient"] += 1
			result.GenreCount["electronic"] += 1
		}
		if af.Energy < 0.4 && af.Acousticness < 0.3 {
			result.GenreCount["r&b"] += 2
			result.GenreCount["soul"] += 1
		}
	}

	fmt.Printf("[DEBUG] Analyze: got %d genre tags from Spotify (from map + audio features)\n", len(result.GenreCount))

	return result, nil
}

func fetchTopArtists(hc *http.Client, token, timeRange string) (*struct {
	Items []TopArtist `json:"items"`
}, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/me/top/artists?limit=20&time_range=%s", timeRange)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top artists: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify API returned status %d", resp.StatusCode)
	}
	var result struct {
		Items []TopArtist `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode top artists: %w", err)
	}
	return &result, nil
}

func fetchTopTracks(hc *http.Client, token, timeRange string) (*struct {
	Items []TopTrackItem `json:"items"`
}, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/me/top/tracks?limit=10&time_range=%s", timeRange)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top tracks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify API returned status %d", resp.StatusCode)
	}
	var result struct {
		Items []TopTrackItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode top tracks: %w", err)
	}
	return &result, nil
}

type audioFeaturesBatch struct {
	AudioFeatures []AudioFeaturesResponse `json:"audio_features"`
}

func fetchAudioFeaturesWithToken(token string, ids []string) (*AudioFeaturesResponse, error) {
	rawClient := &http.Client{}
	url := fmt.Sprintf("https://api.spotify.com/v1/audio-features?ids=%s", strings.Join(ids, ","))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := rawClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch audio features: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audio features API returned status %d", resp.StatusCode)
	}
	var batch audioFeaturesBatch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, fmt.Errorf("failed to decode audio features: %w", err)
	}

	if len(batch.AudioFeatures) == 0 {
		return nil, fmt.Errorf("no audio features returned")
	}

	avg := &AudioFeaturesResponse{}
	var count int
	for _, af := range batch.AudioFeatures {
		avg.Danceability += af.Danceability
		avg.Energy += af.Energy
		avg.Valence += af.Valence
		avg.Acousticness += af.Acousticness
		avg.Instrumentalness += af.Instrumentalness
		avg.Speechiness += af.Speechiness
		avg.Liveness += af.Liveness
		avg.Tempo += af.Tempo
		count++
	}

	if count > 0 {
		avg.Danceability /= float64(count)
		avg.Energy /= float64(count)
		avg.Valence /= float64(count)
		avg.Acousticness /= float64(count)
		avg.Instrumentalness /= float64(count)
		avg.Speechiness /= float64(count)
		avg.Liveness /= float64(count)
		avg.Tempo /= float64(count)
	}

	return avg, nil
}

func fetchPerTrackAudioFeatures(token string, ids []string) (map[string]AudioFeaturesResponse, error) {
	rawClient := &http.Client{}
	url := fmt.Sprintf("https://api.spotify.com/v1/audio-features?ids=%s", strings.Join(ids, ","))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := rawClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch audio features: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audio features API returned status %d", resp.StatusCode)
	}
	var batch audioFeaturesBatch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, fmt.Errorf("failed to decode audio features: %w", err)
	}

	result := make(map[string]AudioFeaturesResponse)
	for i, af := range batch.AudioFeatures {
		if i < len(ids) {
			result[ids[i]] = af
		}
	}
	return result, nil
}

func MapGenresToOurSystem(genreCount map[string]int, dbGenres map[string]string) map[string]float64 {
	ourGenreScores := make(map[string]float64)
	for spotifyGenre, count := range genreCount {
		ourName, found := spotifyToOurGenre[spotifyGenre]
		if !found {
			continue
		}
		genreID, hasID := dbGenres[ourName]
		if !hasID {
			continue
		}
		ourGenreScores[genreID] += float64(count)
	}
	if len(ourGenreScores) == 0 {
		return ourGenreScores
	}
	var maxWeight float64
	for _, w := range ourGenreScores {
		if w > maxWeight {
			maxWeight = w
		}
	}
	normalized := make(map[string]float64)
	for id, w := range ourGenreScores {
		normalized[id] = w / maxWeight
	}
	return normalized
}
