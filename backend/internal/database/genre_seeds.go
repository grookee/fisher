package database

type genreSeed struct {
	Name        string
	Description string
	Color       string
	X, Y        float64
}

type subGenreSeed struct {
	Name        string
	Description string
	Color       string
	X, Y        float64
	Parent      string
}

var rootGenres = []genreSeed{
	{"Rock", "Classic and modern rock music", "#e74c3c", -0.5, -0.3},
	{"Pop", "Popular mainstream music", "#f39c12", 0.3, -0.5},
	{"Hip Hop", "Rap, beats, and hip hop culture", "#9b59b6", -0.1, -0.2},
	{"Electronic", "Electronic dance music and beyond", "#3498db", 0.5, 0.1},
	{"Jazz", "Jazz, blues, and improvisation", "#1abc9c", -0.6, 0.4},
	{"Classical", "Orchestral and classical compositions", "#2ecc71", -0.7, -0.1},
	{"R&B", "Rhythm and blues, soul", "#e91e63", 0.1, 0.3},
	{"Country", "Country and folk music", "#795548", -0.3, 0.6},
	{"Metal", "Heavy metal and hard rock", "#c0392b", -0.4, -0.6},
	{"Indie", "Independent and alternative music", "#00bcd4", 0.4, -0.3},
	{"Folk", "Traditional and acoustic folk", "#8bc34a", -0.2, 0.5},
	{"Punk", "Punk rock and hardcore", "#ff5722", -0.3, -0.5},
	{"Soul", "Soul and funk music", "#ff4081", 0.2, 0.2},
	{"Reggae", "Reggae and dancehall", "#4caf50", -0.1, 0.7},
	{"Blues", "Delta blues to modern blues", "#3f51b5", -0.6, 0.2},
	{"Latin", "Latin American music", "#ff9800", 0.6, 0.3},
	{"Ambient", "Ambient and atmospheric", "#607d8b", 0.7, -0.1},
	{"Funk", "Funk and disco", "#ff6f00", 0.3, 0.4},
	{"Gospel", "Gospel and spiritual", "#e040fb", -0.5, 0.5},
	{"World", "World music and global sounds", "#009688", 0.6, 0.6},
}

var subGenres = []subGenreSeed{
	{"Liquid Drum and Bass", "Melodic, atmospheric drum and bass", "#3498db", 0.55, 0.05, "Electronic"},
	{"Deep House", "Soulful, moody house music", "#3498db", 0.45, 0.15, "Electronic"},
	{"Tech House", "Minimal, groove-driven house", "#3498db", 0.50, 0.12, "Electronic"},
	{"Techno", "Driving, repetitive electronic beats", "#3498db", 0.52, 0.08, "Electronic"},
	{"Trance", "Euphoric, melodic electronic", "#3498db", 0.48, 0.06, "Electronic"},
	{"Dubstep", "Heavy bass, wobble-driven electronic", "#3498db", 0.55, 0.15, "Electronic"},
	{"IDM", "Intelligent, experimental electronic", "#3498db", 0.58, 0.02, "Electronic"},
	{"Future Bass", "Bright, melodic bass music", "#3498db", 0.53, 0.10, "Electronic"},
	{"Synthwave", "Retro 80s-inspired electronic", "#3498db", 0.47, 0.03, "Electronic"},
	{"Breakbeat", "Broken-beat electronic music", "#3498db", 0.54, 0.13, "Electronic"},
	{"UK Garage", "Rhythmic UK club music", "#3498db", 0.51, 0.14, "Electronic"},
	{"Footwork", "Fast, syncopated Chicago electronic", "#3498db", 0.56, 0.09, "Electronic"},
	{"Jungle", "Fast breakbeats and bass", "#3498db", 0.57, 0.07, "Electronic"},
	{"Hardcore", "Fast, intense electronic", "#3498db", 0.49, 0.11, "Electronic"},
	{"Drum and Bass", "Fast breakbeats and sub-bass", "#3498db", 0.53, 0.06, "Electronic"},

	{"Alternative Rock", "Guitar-driven alternative", "#e74c3c", -0.48, -0.28, "Rock"},
	{"Post-Rock", "Atmospheric, instrumental rock", "#e74c3c", -0.52, -0.32, "Rock"},
	{"Psychedelic Rock", "Trippy, experimental rock", "#e74c3c", -0.46, -0.34, "Rock"},
	{"Progressive Rock", "Complex, ambitious rock", "#e74c3c", -0.44, -0.30, "Rock"},
	{"Stoner Rock", "Heavy, riff-driven rock", "#e74c3c", -0.50, -0.28, "Rock"},
	{"Garage Rock", "Raw, lo-fi rock", "#e74c3c", -0.47, -0.35, "Rock"},
	{"Blues Rock", "Rock with blues influences", "#e74c3c", -0.53, -0.25, "Rock"},

	{"Death Metal", "Extreme, heavy metal", "#c0392b", -0.38, -0.62, "Metal"},
	{"Black Metal", "Atmospheric, raw extreme metal", "#c0392b", -0.42, -0.64, "Metal"},
	{"Doom Metal", "Slow, heavy metal", "#c0392b", -0.36, -0.58, "Metal"},
	{"Thrash Metal", "Fast, aggressive metal", "#c0392b", -0.44, -0.60, "Metal"},
	{"Power Metal", "Epic, melodic metal", "#c0392b", -0.40, -0.56, "Metal"},
	{"Symphonic Metal", "Metal with orchestral elements", "#c0392b", -0.43, -0.63, "Metal"},
	{"Folk Metal", "Metal with folk instruments", "#c0392b", -0.39, -0.59, "Metal"},
	{"Sludge Metal", "Heavy, abrasive metal", "#c0392b", -0.37, -0.61, "Metal"},
	{"Post-Metal", "Atmospheric, experimental metal", "#c0392b", -0.41, -0.65, "Metal"},
	{"Djent", "Modern, polyrhythmic metal", "#c0392b", -0.45, -0.57, "Metal"},

	{"Trap", "Hi-hat driven hip hop", "#9b59b6", -0.05, -0.15, "Hip Hop"},
	{"Drill", "Dark, aggressive hip hop", "#9b59b6", -0.08, -0.18, "Hip Hop"},
	{"Gangsta Rap", "Street-oriented hip hop", "#9b59b6", -0.12, -0.22, "Hip Hop"},
	{"Conscious Rap", "Socially aware hip hop", "#9b59b6", -0.06, -0.23, "Hip Hop"},
	{"Alternative Hip Hop", "Experimental hip hop", "#9b59b6", -0.03, -0.16, "Hip Hop"},
	{"Grime", "UK electronic hip hop", "#9b59b6", -0.09, -0.19, "Hip Hop"},
	{"Boom Bap", "Sample-based classic hip hop", "#9b59b6", -0.14, -0.24, "Hip Hop"},
	{"Cloud Rap", "Atmospheric, ethereal hip hop", "#9b59b6", -0.04, -0.17, "Hip Hop"},
	{"Phonk", "Cowbell-driven Memphis hip hop", "#9b59b6", -0.07, -0.20, "Hip Hop"},
	{"Jersey Club", "Fast, dance-oriented hip hop", "#9b59b6", -0.10, -0.21, "Hip Hop"},

	{"Bebop", "Fast, complex jazz", "#1abc9c", -0.58, 0.42, "Jazz"},
	{"Cool Jazz", "Relaxed, smooth jazz", "#1abc9c", -0.62, 0.38, "Jazz"},
	{"Free Jazz", "Avant-garde, experimental jazz", "#1abc9c", -0.55, 0.45, "Jazz"},
	{"Jazz Fusion", "Jazz with rock and funk", "#1abc9c", -0.57, 0.44, "Jazz"},
	{"Swing", "Big band jazz", "#1abc9c", -0.63, 0.36, "Jazz"},
	{"Smooth Jazz", "Radio-friendly jazz", "#1abc9c", -0.59, 0.41, "Jazz"},
	{"Acid Jazz", "Jazz with funk and hip hop", "#1abc9c", -0.56, 0.43, "Jazz"},
	{"Modal Jazz", "Mode-based jazz improvisation", "#1abc9c", -0.61, 0.39, "Jazz"},

	{"Baroque", "Ornate classical period", "#2ecc71", -0.72, -0.08, "Classical"},
	{"Romantic", "Expressive 19th century classical", "#2ecc71", -0.68, -0.05, "Classical"},
	{"Modern Classical", "20th/21st century classical", "#2ecc71", -0.65, -0.12, "Classical"},
	{"Minimalism", "Repetitive, stripped-down classical", "#2ecc71", -0.69, -0.09, "Classical"},
	{"Opera", "Dramatic vocal classical", "#2ecc71", -0.71, -0.06, "Classical"},
	{"Chamber Music", "Small ensemble classical", "#2ecc71", -0.67, -0.10, "Classical"},
	{"Choral", "Vocal ensemble classical", "#2ecc71", -0.70, -0.07, "Classical"},

	{"Drone", "Sustained, minimal ambient", "#607d8b", 0.72, -0.08, "Ambient"},
	{"Dark Ambient", "Ominous, brooding ambient", "#607d8b", 0.68, -0.05, "Ambient"},
	{"Ambient Pop", "Pop with ambient textures", "#607d8b", 0.65, -0.12, "Ambient"},
	{"Space Music", "Cosmic, floating ambient", "#607d8b", 0.70, -0.09, "Ambient"},

	{"Indie Rock", "Alternative guitar music", "#00bcd4", 0.42, -0.28, "Indie"},
	{"Indie Pop", "Melodic, accessible indie", "#00bcd4", 0.38, -0.32, "Indie"},
	{"Dream Pop", "Ethereal, atmospheric indie", "#00bcd4", 0.44, -0.26, "Indie"},
	{"Shoegaze", "Wall of sound guitar indie", "#00bcd4", 0.40, -0.30, "Indie"},
	{"Lo-Fi", "Loosely produced indie", "#00bcd4", 0.36, -0.34, "Indie"},
	{"Bedroom Pop", "Home-recorded indie pop", "#00bcd4", 0.43, -0.27, "Indie"},
	{"Noise Pop", "Melodic indie with noise", "#00bcd4", 0.41, -0.29, "Indie"},
	{"Post-Punk Revival", "Modern take on post-punk", "#00bcd4", 0.45, -0.31, "Indie"},

	{"Indie Folk", "Modern acoustic folk", "#8bc34a", -0.18, 0.52, "Folk"},
	{"Singer-Songwriter", "Solo artist folk", "#8bc34a", -0.22, 0.48, "Folk"},
	{"Neofolk", "Dark, experimental folk", "#8bc34a", -0.16, 0.54, "Folk"},
	{"Traditional Folk", "Heritage folk music", "#8bc34a", -0.24, 0.46, "Folk"},
	{"Americana", "Roots-based American folk", "#8bc34a", -0.20, 0.50, "Folk"},

	{"Hardcore Punk", "Fast, intense punk", "#ff5722", -0.28, -0.52, "Punk"},
	{"Pop Punk", "Melodic, catchy punk", "#ff5722", -0.32, -0.48, "Punk"},
	{"Post-Hardcore", "Experimental punk", "#ff5722", -0.26, -0.54, "Punk"},
	{"Skate Punk", "Fast, upbeat punk", "#ff5722", -0.30, -0.50, "Punk"},
	{"Anarcho-Punk", "Political punk", "#ff5722", -0.34, -0.46, "Punk"},
	{"Crust Punk", "Raw, extreme punk", "#ff5722", -0.25, -0.55, "Punk"},

	{"Bluegrass", "Acoustic string band country", "#795548", -0.28, 0.62, "Country"},
	{"Country Rock", "Rock-influenced country", "#795548", -0.32, 0.58, "Country"},
	{"Outlaw Country", "Rebellious country", "#795548", -0.26, 0.64, "Country"},
	{"Country Pop", "Pop-friendly country", "#795548", -0.34, 0.56, "Country"},
	{"Alt-Country", "Alternative country", "#795548", -0.30, 0.60, "Country"},

	{"Neo-Soul", "Modern soul music", "#ff4081", 0.22, 0.22, "Soul"},
	{"Northern Soul", "60s soul revival", "#ff4081", 0.18, 0.18, "Soul"},
	{"Psychedelic Soul", "Trippy, experimental soul", "#ff4081", 0.24, 0.24, "Soul"},
	{"Southern Soul", "Deep southern US soul", "#ff4081", 0.16, 0.16, "Soul"},

	{"Alternative R&B", "Experimental R&B", "#e91e63", 0.12, 0.32, "R&B"},
	{"Contemporary R&B", "Modern rhythm and blues", "#e91e63", 0.08, 0.28, "R&B"},

	{"Delta Blues", "Early Mississippi blues", "#3f51b5", -0.58, 0.18, "Blues"},
	{"Chicago Blues", "Electric urban blues", "#3f51b5", -0.62, 0.22, "Blues"},
	{"Electric Blues", "Amplified blues", "#3f51b5", -0.56, 0.20, "Blues"},
	{"Texas Blues", "Guitar-driven blues", "#3f51b5", -0.60, 0.24, "Blues"},

	{"Dub", "Instrumental, spacey reggae", "#4caf50", -0.08, 0.72, "Reggae"},
	{"Dancehall", "Upbeat club reggae", "#4caf50", -0.12, 0.68, "Reggae"},
	{"Ska", "Fast, horn-driven reggae predecessor", "#4caf50", -0.06, 0.74, "Reggae"},
	{"Rocksteady", "Slow, smooth reggae", "#4caf50", -0.14, 0.66, "Reggae"},
	{"Roots Reggae", "Spiritual, conscious reggae", "#4caf50", -0.10, 0.70, "Reggae"},

	{"Salsa", "Dance-oriented Latin music", "#ff9800", 0.62, 0.32, "Latin"},
	{"Bachata", "Romantic guitar Latin", "#ff9800", 0.58, 0.28, "Latin"},
	{"Reggaeton", "Urban Latin dance", "#ff9800", 0.64, 0.34, "Latin"},
	{"Cumbia", "Colombian dance music", "#ff9800", 0.56, 0.30, "Latin"},
	{"Bossa Nova", "Brazilian jazz-influenced", "#ff9800", 0.60, 0.26, "Latin"},
	{"Samba", "Brazilian carnival music", "#ff9800", 0.66, 0.36, "Latin"},
	{"Flamenco", "Spanish guitar and dance", "#ff9800", 0.54, 0.28, "Latin"},
	{"Merengue", "Fast Dominican dance", "#ff9800", 0.63, 0.33, "Latin"},
	{"Dembow", "Minimalist Dominican dance", "#ff9800", 0.59, 0.35, "Latin"},

	{"Disco", "70s dance music", "#ff6f00", 0.28, 0.42, "Funk"},
	{"P-Funk", "Psychedelic funk", "#ff6f00", 0.32, 0.38, "Funk"},
	{"G-Funk", "West coast hip hop funk", "#ff6f00", 0.34, 0.44, "Funk"},
	{"Boogie", "Post-disco funk", "#ff6f00", 0.30, 0.40, "Funk"},

	{"Afrobeat", "West African funk/jazz", "#009688", 0.62, 0.62, "World"},
	{"Afro House", "African electronic house", "#009688", 0.64, 0.58, "World"},
	{"Highlife", "Ghanaian dance music", "#009688", 0.58, 0.64, "World"},
	{"Soca", "Caribbean carnival music", "#009688", 0.60, 0.60, "World"},
	{"Kizomba", "Angolan dance music", "#009688", 0.63, 0.59, "World"},

	{"Dance Pop", "Upbeat, club-friendly pop", "#f39c12", 0.32, -0.48, "Pop"},
	{"Art Pop", "Experimental pop", "#f39c12", 0.28, -0.52, "Pop"},
	{"Hyperpop", "Maximalist internet pop", "#f39c12", 0.34, -0.46, "Pop"},
	{"J-Pop", "Japanese pop", "#f39c12", 0.26, -0.54, "Pop"},
	{"K-Pop", "Korean pop", "#f39c12", 0.30, -0.50, "Pop"},
	{"Synth Pop", "Synthesizer-driven pop", "#f39c12", 0.36, -0.44, "Pop"},
}
