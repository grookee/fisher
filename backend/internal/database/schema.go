package database

var schemaMigrations = []string{
	`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
	`CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(100) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		avatar_url TEXT DEFAULT '',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS genres (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name VARCHAR(100) UNIQUE NOT NULL,
		description TEXT DEFAULT '',
		color VARCHAR(7) DEFAULT '#6366f1',
		x FLOAT DEFAULT 0,
		y FLOAT DEFAULT 0,
		parent_id UUID REFERENCES genres(id)
	)`,
	`CREATE TABLE IF NOT EXISTS tracks (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		title VARCHAR(255) NOT NULL,
		artist VARCHAR(255) NOT NULL,
		album VARCHAR(255) DEFAULT '',
		album_art_url TEXT DEFAULT '',
		duration_ms INT DEFAULT 0,
		spotify_uri VARCHAR(255) DEFAULT '',
		apple_music_id VARCHAR(255) DEFAULT '',
		preview_url TEXT DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS track_genres (
		track_id UUID REFERENCES tracks(id) ON DELETE CASCADE,
		genre_id UUID REFERENCES genres(id) ON DELETE CASCADE,
		PRIMARY KEY (track_id, genre_id)
	)`,
	`CREATE TABLE IF NOT EXISTS playlists (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		title VARCHAR(255) NOT NULL,
		description TEXT DEFAULT '',
		owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		is_public BOOLEAN DEFAULT false,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS playlist_tracks (
		playlist_id UUID REFERENCES playlists(id) ON DELETE CASCADE,
		track_id UUID REFERENCES tracks(id) ON DELETE CASCADE,
		position INT DEFAULT 0,
		added_at TIMESTAMPTZ DEFAULT NOW(),
		PRIMARY KEY (playlist_id, track_id)
	)`,
	`CREATE TABLE IF NOT EXISTS collaborations (
		playlist_id UUID REFERENCES playlists(id) ON DELETE CASCADE,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		permission VARCHAR(20) DEFAULT 'edit',
		PRIMARY KEY (playlist_id, user_id)
	)`,
	`CREATE TABLE IF NOT EXISTS friends (
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		friend_id UUID REFERENCES users(id) ON DELETE CASCADE,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		PRIMARY KEY (user_id, friend_id)
	)`,
	`CREATE TABLE IF NOT EXISTS taste_profiles (
		user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		top_artists TEXT[] DEFAULT '{}',
		updated_at TIMESTAMPTZ DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS taste_genres (
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		genre_id UUID REFERENCES genres(id) ON DELETE CASCADE,
		weight FLOAT DEFAULT 0,
		PRIMARY KEY (user_id, genre_id)
	)`,
	`CREATE TABLE IF NOT EXISTS taste_shares (
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		shared_with UUID REFERENCES users(id) ON DELETE CASCADE,
		PRIMARY KEY (user_id, shared_with)
	)`,
	`CREATE TABLE IF NOT EXISTS oauth_states (
		state VARCHAR(255) PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		service VARCHAR(50) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS user_accounts (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		service VARCHAR(50) NOT NULL,
		service_user_id VARCHAR(255) DEFAULT '',
		access_token TEXT DEFAULT '',
		refresh_token TEXT DEFAULT '',
		token_expiry TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(user_id, service)
	)`,
	`CREATE TABLE IF NOT EXISTS user_settings (
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		key VARCHAR(100) NOT NULL,
		value TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (user_id, key)
	)`,
	`DO $$ BEGIN
		ALTER TABLE taste_profiles ADD COLUMN top_tracks JSONB DEFAULT '[]';
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN release_year INT DEFAULT 0;
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN danceability FLOAT DEFAULT 0;
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN energy FLOAT DEFAULT 0;
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN valence FLOAT DEFAULT 0;
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN acousticness FLOAT DEFAULT 0;
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN instrumentalness FLOAT DEFAULT 0;
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN speechiness FLOAT DEFAULT 0;
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN tempo FLOAT DEFAULT 0;
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE genres ADD COLUMN slug VARCHAR(150);
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE genres ADD COLUMN wikidata_qid VARCHAR(20);
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE genres ADD COLUMN source VARCHAR(30) DEFAULT 'seed';
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE genres ADD COLUMN countries TEXT[] DEFAULT '{}';
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`DO $$ BEGIN
		ALTER TABLE genres ADD COLUMN updated_at TIMESTAMPTZ DEFAULT NOW();
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_genres_wikidata_qid ON genres(wikidata_qid) WHERE wikidata_qid IS NOT NULL`,
	`CREATE INDEX IF NOT EXISTS idx_genres_lower_name ON genres (LOWER(name))`,
	`CREATE TABLE IF NOT EXISTS genre_aliases (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		genre_id UUID NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
		alias VARCHAR(150) NOT NULL,
		source VARCHAR(30) DEFAULT 'wikidata',
		UNIQUE(genre_id, alias)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_genre_aliases_lower_alias ON genre_aliases (LOWER(alias))`,
	`CREATE TABLE IF NOT EXISTS genre_relations (
		genre_id UUID NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
		related_genre_id UUID NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
		relation_type VARCHAR(30) NOT NULL,
		weight FLOAT DEFAULT 0.5,
		source VARCHAR(30) DEFAULT '',
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		PRIMARY KEY (genre_id, related_genre_id, relation_type)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_genre_relations_genre ON genre_relations(genre_id)`,
	`CREATE INDEX IF NOT EXISTS idx_genre_relations_related ON genre_relations(related_genre_id)`,
	`CREATE TABLE IF NOT EXISTS artist_genres (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		artist_name VARCHAR(255) NOT NULL,
		mbid VARCHAR(64) DEFAULT '',
		genre_id UUID NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
		confidence FLOAT DEFAULT 0.5,
		source VARCHAR(30) DEFAULT 'lastfm',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(artist_name, genre_id, source)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_artist_genres_lower_artist ON artist_genres (LOWER(artist_name))`,
	`CREATE INDEX IF NOT EXISTS idx_artist_genres_genre ON artist_genres(genre_id)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_tracks_spotify_uri ON tracks(spotify_uri) WHERE spotify_uri <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_track_genres_genre ON track_genres(genre_id)`,
	`CREATE TABLE IF NOT EXISTS artist_track_progress (
		artist_name VARCHAR(255) PRIMARY KEY,
		spotify_artist_id VARCHAR(64) DEFAULT '',
		fetched_at TIMESTAMPTZ DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS musicbrainz_cache (
		artist_name VARCHAR(255) PRIMARY KEY,
		mbid VARCHAR(64) DEFAULT '',
		country VARCHAR(10) DEFAULT '',
		area VARCHAR(255) DEFAULT '',
		fetched_at TIMESTAMPTZ DEFAULT NOW()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_musicbrainz_cache_fetched ON musicbrainz_cache(fetched_at)`,
	`DO $$ BEGIN
		ALTER TABLE tracks ADD COLUMN deezer_id VARCHAR(64) DEFAULT '';
	EXCEPTION WHEN duplicate_column THEN END $$`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_tracks_deezer_id ON tracks(deezer_id) WHERE deezer_id <> ''`,
	// Backfill: normalize ISO country codes to full names and deduplicate.
	// This runs via a DO block that uses a temporary function.
	`DO $$
	DECLARE
		r RECORD;
		normalized TEXT[];
		item TEXT;
		seen TEXT[];
	BEGIN
		FOR r IN SELECT id, countries FROM genres WHERE array_length(countries, 1) > 0 LOOP
			normalized := '{}';
			seen := '{}';
			FOREACH item IN ARRAY r.countries LOOP
				CASE upper(item)
					WHEN 'US' THEN item := 'United States';
					WHEN 'GB' THEN item := 'United Kingdom';
					WHEN 'KR' THEN item := 'South Korea';
					WHEN 'HU' THEN item := 'Hungary';
					WHEN 'CZ' THEN item := 'Czech Republic';
					WHEN 'NZ' THEN item := 'New Zealand';
					WHEN 'JP' THEN item := 'Japan';
					WHEN 'BR' THEN item := 'Brazil';
					WHEN 'DE' THEN item := 'Germany';
					WHEN 'FR' THEN item := 'France';
					WHEN 'IT' THEN item := 'Italy';
					WHEN 'ES' THEN item := 'Spain';
					WHEN 'PT' THEN item := 'Portugal';
					WHEN 'NL' THEN item := 'Netherlands';
					WHEN 'SE' THEN item := 'Sweden';
					WHEN 'NO' THEN item := 'Norway';
					WHEN 'DK' THEN item := 'Denmark';
					WHEN 'FI' THEN item := 'Finland';
					WHEN 'IS' THEN item := 'Iceland';
					WHEN 'PL' THEN item := 'Poland';
					WHEN 'RU' THEN item := 'Russia';
					WHEN 'UA' THEN item := 'Ukraine';
					WHEN 'RO' THEN item := 'Romania';
					WHEN 'BG' THEN item := 'Bulgaria';
					WHEN 'RS' THEN item := 'Serbia';
					WHEN 'HR' THEN item := 'Croatia';
					WHEN 'BA' THEN item := 'Bosnia and Herzegovina';
					WHEN 'SI' THEN item := 'Slovenia';
					WHEN 'SK' THEN item := 'Slovakia';
					WHEN 'MK' THEN item := 'North Macedonia';
					WHEN 'AL' THEN item := 'Albania';
					WHEN 'GR' THEN item := 'Greece';
					WHEN 'TR' THEN item := 'Turkey';
					WHEN 'IL' THEN item := 'Israel';
					WHEN 'EG' THEN item := 'Egypt';
					WHEN 'MA' THEN item := 'Morocco';
					WHEN 'NG' THEN item := 'Nigeria';
					WHEN 'GH' THEN item := 'Ghana';
					WHEN 'KE' THEN item := 'Kenya';
					WHEN 'ZA' THEN item := 'South Africa';
					WHEN 'IN' THEN item := 'India';
					WHEN 'PK' THEN item := 'Pakistan';
					WHEN 'BD' THEN item := 'Bangladesh';
					WHEN 'LK' THEN item := 'Sri Lanka';
					WHEN 'NP' THEN item := 'Nepal';
					WHEN 'TH' THEN item := 'Thailand';
					WHEN 'VN' THEN item := 'Vietnam';
					WHEN 'PH' THEN item := 'Philippines';
					WHEN 'ID' THEN item := 'Indonesia';
					WHEN 'MY' THEN item := 'Malaysia';
					WHEN 'SG' THEN item := 'Singapore';
					WHEN 'CN' THEN item := 'China';
					WHEN 'TW' THEN item := 'Taiwan';
					WHEN 'HK' THEN item := 'Hong Kong';
					WHEN 'MN' THEN item := 'Mongolia';
					WHEN 'AU' THEN item := 'Australia';
					WHEN 'CA' THEN item := 'Canada';
					WHEN 'MX' THEN item := 'Mexico';
					WHEN 'CO' THEN item := 'Colombia';
					WHEN 'AR' THEN item := 'Argentina';
					WHEN 'CL' THEN item := 'Chile';
					WHEN 'PE' THEN item := 'Peru';
					WHEN 'VE' THEN item := 'Venezuela';
					WHEN 'EC' THEN item := 'Ecuador';
					WHEN 'BO' THEN item := 'Bolivia';
					WHEN 'PY' THEN item := 'Paraguay';
					WHEN 'UY' THEN item := 'Uruguay';
					WHEN 'CU' THEN item := 'Cuba';
					WHEN 'JM' THEN item := 'Jamaica';
					WHEN 'DO' THEN item := 'Dominican Republic';
					WHEN 'CR' THEN item := 'Costa Rica';
					WHEN 'PA' THEN item := 'Panama';
					WHEN 'GT' THEN item := 'Guatemala';
					WHEN 'HN' THEN item := 'Honduras';
					WHEN 'NI' THEN item := 'Nicaragua';
					WHEN 'HT' THEN item := 'Haiti';
					WHEN 'IE' THEN item := 'Ireland';
					WHEN 'AT' THEN item := 'Austria';
					WHEN 'CH' THEN item := 'Switzerland';
					WHEN 'BE' THEN item := 'Belgium';
					WHEN 'EE' THEN item := 'Estonia';
					WHEN 'LV' THEN item := 'Latvia';
					WHEN 'LT' THEN item := 'Lithuania';
					WHEN 'BY' THEN item := 'Belarus';
					WHEN 'MD' THEN item := 'Moldova';
					WHEN 'GE' THEN item := 'Georgia';
					WHEN 'AM' THEN item := 'Armenia';
					WHEN 'AZ' THEN item := 'Azerbaijan';
					WHEN 'KZ' THEN item := 'Kazakhstan';
					WHEN 'UZ' THEN item := 'Uzbekistan';
					WHEN 'KG' THEN item := 'Kyrgyzstan';
					WHEN 'TJ' THEN item := 'Tajikistan';
					WHEN 'TM' THEN item := 'Turkmenistan';
					WHEN 'AE' THEN item := 'United Arab Emirates';
					WHEN 'SA' THEN item := 'Saudi Arabia';
					WHEN 'KW' THEN item := 'Kuwait';
					WHEN 'QA' THEN item := 'Qatar';
					WHEN 'BH' THEN item := 'Bahrain';
					WHEN 'OM' THEN item := 'Oman';
					WHEN 'JO' THEN item := 'Jordan';
					WHEN 'LB' THEN item := 'Lebanon';
					WHEN 'IQ' THEN item := 'Iraq';
					WHEN 'IR' THEN item := 'Iran';
					WHEN 'SY' THEN item := 'Syria';
					WHEN 'YE' THEN item := 'Yemen';
					WHEN 'SD' THEN item := 'Sudan';
					WHEN 'TN' THEN item := 'Tunisia';
					WHEN 'DZ' THEN item := 'Algeria';
					WHEN 'LY' THEN item := 'Libya';
					WHEN 'CM' THEN item := 'Cameroon';
					WHEN 'SN' THEN item := 'Senegal';
					WHEN 'ML' THEN item := 'Mali';
					WHEN 'NE' THEN item := 'Niger';
					WHEN 'UG' THEN item := 'Uganda';
					WHEN 'TZ' THEN item := 'Tanzania';
					WHEN 'MZ' THEN item := 'Mozambique';
					WHEN 'ZM' THEN item := 'Zambia';
					WHEN 'ZW' THEN item := 'Zimbabwe';
					WHEN 'NA' THEN item := 'Namibia';
					WHEN 'BW' THEN item := 'Botswana';
					WHEN 'MU' THEN item := 'Mauritius';
					WHEN 'MG' THEN item := 'Madagascar';
					WHEN 'KH' THEN item := 'Cambodia';
					WHEN 'LA' THEN item := 'Laos';
					WHEN 'MM' THEN item := 'Myanmar';
					WHEN 'MO' THEN item := 'Macao';
					WHEN 'AF' THEN item := 'Afghanistan';
				ELSE
					item := item;
				END CASE;
				IF NOT (item = ANY(seen)) THEN
					seen := array_append(seen, item);
					normalized := array_append(normalized, item);
				END IF;
			END LOOP;
			IF normalized != r.countries THEN
				UPDATE genres SET countries = normalized, updated_at = NOW() WHERE id = r.id;
			END IF;
		END LOOP;
	END $$;`,
}
