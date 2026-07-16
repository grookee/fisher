export function buildTrackExternalUrl(track, preferredPlatform = 'spotify') {
  if (!track) return null;

  const spotifyUrl = track.spotify_uri ? `https://open.spotify.com/track/${track.spotify_uri}` : null;
  const appleUrl = track.apple_music_id ? `https://music.apple.com/album/${track.apple_music_id}` : null;
  const songLink = track.spotify_uri ? `https://song.link/s/${track.spotify_uri}` : null;

  if (preferredPlatform === 'spotify' && spotifyUrl) return spotifyUrl;
  if (preferredPlatform === 'apple_music' && appleUrl) return appleUrl;
  if (preferredPlatform === 'song_link' && songLink) return songLink;

  return spotifyUrl || appleUrl || songLink;
}

export function buildTrackSharePayload(track, preferredPlatform = 'spotify') {
  const url = buildTrackExternalUrl(track, preferredPlatform);
  if (!url || !track) return null;

  const artist = track.artist || 'Unknown Artist';
  const title = track.title || 'Track';

  return {
    title: `${title} — ${artist}`,
    text: `Check out \"${title}\" by ${artist}`,
    url,
  };
}

export async function shareTrack(track, preferredPlatform = 'spotify') {
  const payload = buildTrackSharePayload(track, preferredPlatform);
  if (!payload) {
    return { ok: false, error: 'No shareable link available for this track' };
  }

  if (typeof navigator !== 'undefined' && typeof navigator.share === 'function') {
    try {
      await navigator.share(payload);
      return { ok: true, method: 'native', url: payload.url };
    } catch (err) {
      if (err?.name === 'AbortError') {
        return { ok: false, cancelled: true };
      }
    }
  }

  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(payload.url);
    return { ok: true, method: 'clipboard', url: payload.url };
  }

  return { ok: false, error: 'Sharing is not supported in this browser' };
}
