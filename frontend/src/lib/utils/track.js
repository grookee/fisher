export const stats = [
  { key: 'danceability', label: 'danceability' },
  { key: 'energy', label: 'energy' },
  { key: 'valence', label: 'valence' },
  { key: 'acousticness', label: 'acoustic' },
  { key: 'instrumentalness', label: 'instrumental' },
  { key: 'speechiness', label: 'speech' },
  { key: 'tempo', label: 'tempo', unit: 'bpm' },
];

function toNum(value) {
  if (value == null || value === '') return null;
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}

function toInt(value) {
  if (value == null || value === '') return 0;
  const n = Number(value);
  return Number.isFinite(n) ? Math.round(n) : 0;
}

export function normalizeTrack(track) {
  if (!track) return track;

  const audio = track.audio_features ?? track.audioFeatures ?? track.audio_stats ?? track.audioStats ?? track.AudioFeatures ?? {};
  const releaseYear = toInt(track.release_year ?? track.releaseYear ?? track.year);
  const durationMs = toInt(track.duration_ms ?? track.durationMs ?? track.duration);
  const normalizedId =
    track.id ??
    track.track_id ??
    track.trackId ??
    track.spotify_uri ??
    track.spotifyUri ??
    track.uri ??
    track.URI ??
    '';

  return {
    ...track,
    id: normalizedId,
    title: track.title ?? track.Title ?? '',
    artist: track.artist ?? track.Artist ?? '',
    album: track.album ?? track.Album ?? '',
    album_art_url: track.album_art_url ?? track.albumArtURL ?? track.albumArt ?? track.AlbumArtURL ?? track.image ?? '',
    preview_url: track.preview_url ?? track.previewUrl ?? track.preview ?? track.PreviewURL ?? '',
    spotify_uri: track.spotify_uri ?? track.spotifyUri ?? track.uri ?? track.URI ?? '',
    duration_ms: durationMs,
    release_year: releaseYear,
    danceability: toNum(track.danceability ?? track.Danceability ?? audio.danceability ?? audio.Danceability),
    energy: toNum(track.energy ?? track.Energy ?? audio.energy ?? audio.Energy),
    valence: toNum(track.valence ?? track.Valence ?? audio.valence ?? audio.Valence),
    acousticness: toNum(track.acousticness ?? track.Acousticness ?? audio.acousticness ?? audio.Acousticness),
    instrumentalness: toNum(track.instrumentalness ?? track.Instrumentalness ?? audio.instrumentalness ?? audio.Instrumentalness),
    speechiness: toNum(track.speechiness ?? track.Speechiness ?? audio.speechiness ?? audio.Speechiness),
    tempo: toNum(track.tempo ?? track.Tempo ?? audio.tempo ?? audio.Tempo),
    genre: track.genre ?? track.genre_name ?? track.genreName ?? track.Genre ?? track.GenreName ?? '',
    genre_path: track.genre_path ?? track.genrePath ?? '',
  };
}

export function fmtMs(ms) {
  if (!ms) return '';
  const m = Math.floor(ms / 60000);
  const s = Math.floor((ms % 60000) / 1000);
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export function pct(v) {
  if (v == null) return null;
  return Math.round(v * 100);
}

export function bar(v) {
  if (v == null) return 0;
  return Math.round(v * 100);
}

export function hasStats(track) {
  if (!track) return false;
  return (
    (track.danceability ?? 0) > 0 ||
    (track.energy ?? 0) > 0 ||
    (track.valence ?? 0) > 0 ||
    (track.acousticness ?? 0) > 0 ||
    (track.instrumentalness ?? 0) > 0 ||
    (track.speechiness ?? 0) > 0 ||
    (track.tempo ?? 0) > 0
  );
}
