const API_URL = import.meta.env.VITE_API_URL ?? '';

const GET_CACHE_TTL_MS = 120000;
const getCache = new Map();
const inflightGetRequests = new Map();

function cloneData(data) {
  if (typeof structuredClone === 'function') return structuredClone(data);
  return JSON.parse(JSON.stringify(data));
}

function getCacheKey(path, token) {
  return `${token || 'anon'}::${path}`;
}

function clearGetCache() {
  getCache.clear();
}

async function request(path, options = {}) {
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null;
  const method = (options.method || 'GET').toUpperCase();
  const useGetCache = method === 'GET' && !options.noCache;
  const cacheKey = getCacheKey(path, token);

  if (useGetCache) {
    const cached = getCache.get(cacheKey);
    if (cached && cached.expiresAt > Date.now()) {
      return cloneData(cached.data);
    }

    const inflight = inflightGetRequests.get(cacheKey);
    if (inflight) {
      return cloneData(await inflight);
    }
  }

  const { noCache: _noCache, ...fetchOptions } = options;
  const headers = {
    'Content-Type': 'application/json',
    ...fetchOptions.headers,
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const doRequest = async () => {
    const res = await fetch(`${API_URL}${path}`, {
      ...fetchOptions,
      headers,
    });

    if (res.status === 204) return null;

    const data = await res.json();

    if (!res.ok) {
      throw new Error(data.error || 'request failed');
    }

    return data;
  };

  if (useGetCache) {
    const promise = doRequest();
    inflightGetRequests.set(cacheKey, promise);
    try {
      const data = await promise;
      getCache.set(cacheKey, {
        data: cloneData(data),
        expiresAt: Date.now() + GET_CACHE_TTL_MS,
      });
      return cloneData(data);
    } finally {
      inflightGetRequests.delete(cacheKey);
    }
  }

  const data = await doRequest();
  clearGetCache();
  return data;
}

export const api = {
  auth: {
    register: (body) => request('/api/auth/register', { method: 'POST', body: JSON.stringify(body) }),
    login: (body) => request('/api/auth/login', { method: 'POST', body: JSON.stringify(body) }),
    me: () => request('/api/auth/me'),
    accounts: () => request('/api/auth/accounts'),
    unlink: (service) => request(`/api/auth/accounts/${service}`, { method: 'DELETE' }),
    spotifyAuthorize: () => request('/api/auth/spotify/authorize'),
  },
  playlists: {
    list: () => request('/api/playlists'),
    get: (id) => request(`/api/playlists/${id}`),
    create: (body) => request('/api/playlists', { method: 'POST', body: JSON.stringify(body) }),
    update: (id, body) => request(`/api/playlists/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    delete: (id) => request(`/api/playlists/${id}`, { method: 'DELETE' }),
    search: (q) => request(`/api/playlists/search?q=${encodeURIComponent(q)}`),
    addCollaborator: (id, body) => request(`/api/playlists/${id}/collaborators`, { method: 'POST', body: JSON.stringify(body) }),
    removeCollaborator: (id, userId) => request(`/api/playlists/${id}/collaborators/${userId}`, { method: 'DELETE' }),
  },
  settings: {
    getAll: () => request('/api/settings'),
    set: (key, value) => request('/api/settings', { method: 'PUT', body: JSON.stringify({ key, value }) }),
  },
  discover: {
    get: (params = {}) => {
      const qs = new URLSearchParams(params).toString();
      return request(`/api/discover${qs ? '?' + qs : ''}`);
    },
  },
  explore: {
    feed: (params = {}, opts = {}) => {
      const qs = new URLSearchParams(params).toString();
      return request(`/api/explore/feed${qs ? '?' + qs : ''}`, opts);
    },
    missed: () => request('/api/explore/missed'),
    timemachine: (params = {}) => {
      const qs = new URLSearchParams(params).toString();
      return request(`/api/explore/timemachine${qs ? '?' + qs : ''}`);
    },
    genre: (params = {}) => {
      const qs = new URLSearchParams(params).toString();
      return request(`/api/explore/genre${qs ? '?' + qs : ''}`);
    },
    related: (params = {}) => {
      const qs = new URLSearchParams(params).toString();
      return request(`/api/explore/related${qs ? '?' + qs : ''}`);
    },
    recommendations: (params = {}) => {
      const qs = new URLSearchParams(params).toString();
      return request(`/api/explore/recommendations${qs ? '?' + qs : ''}`);
    },
  },
  friends: {
    list: () => request('/api/friends'),
    requests: () => request('/api/friends/requests'),
    send: (friendId) => request('/api/friends/request', { method: 'POST', body: JSON.stringify({ friend_id: friendId }) }),
    accept: (friendId) => request('/api/friends/accept', { method: 'POST', body: JSON.stringify({ friend_id: friendId }) }),
    remove: (friendId) => request(`/api/friends?friend_id=${friendId}`, { method: 'DELETE' }),
    search: (q) => request(`/api/friends/search?q=${encodeURIComponent(q)}`),
  },
  tastes: {
    genres: () => request('/api/tastes/genres'),
    analyze: (params = {}) => {
      const qs = new URLSearchParams(params).toString();
      return request(`/api/tastes/analyze${qs ? '?' + qs : ''}`, { method: 'POST' });
    },
    profile: (userId) => request(userId ? `/api/tastes/profile/${userId}` : '/api/tastes/profile'),
    updateGenres: (genres) => request('/api/tastes/genres', { method: 'POST', body: JSON.stringify({ genres }) }),
    share: (userIds) => request('/api/tastes/share', { method: 'POST', body: JSON.stringify({ user_ids: userIds }) }),
    unshare: (userId) => request('/api/tastes/unshare', { method: 'POST', body: JSON.stringify({ user_id: userId }) }),
    friendsTastes: () => request('/api/tastes/friends'),
  },
  previews: {
    resolve: (tracks) => request('/api/previews/resolve', { method: 'POST', body: JSON.stringify({ tracks }) }),
  },
};
