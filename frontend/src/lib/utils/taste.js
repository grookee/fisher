const AF_LABELS = {
  danceability: 'danceability',
  energy: 'energy',
  valence: 'valence',
  acousticness: 'acousticness',
  instrumentalness: 'instrumentalness',
  speechiness: 'speechiness',
  liveness: 'liveness',
};

const AF_COLORS = {
  danceability: '#f59e0b',
  energy: '#ef4444',
  valence: '#10b981',
  acousticness: '#3b82f6',
  instrumentalness: '#8b5cf6',
  speechiness: '#ec4899',
  liveness: '#14b8a6',
};

export function afLabel(key) {
  return AF_LABELS[key] || key;
}

export function afColor(key) {
  return AF_COLORS[key] || '#6366f1';
}

export function fmtMs(ms) {
  if (!ms) return '';
  const min = Math.floor(ms / 60000);
  const sec = Math.floor((ms % 60000) / 1000);
  return `${min}:${sec.toString().padStart(2, '0')}`;
}

export function topGenres(genres, limit = 20) {
  if (!Array.isArray(genres)) return [];
  return [...genres].sort((a, b) => b.weight - a.weight).slice(0, limit);
}
