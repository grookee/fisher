<script>
  import { onDestroy, onMount } from 'svelte';
  import { slide } from 'svelte/transition';
  import { api } from '$lib/api/client';
  import { isAuthenticated, user } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { shareTrack } from '$lib/utils/share';
  import { debounce } from '$lib/utils/debounce';

  let playlists = [];
  let loading = true;
  let showCreate = false;
  let newTitle = '';
  let newDesc = '';
  let newPublic = false;
  let selectedPlaylist = null;
  let searchQuery = '';
  let searchResults = [];
  let editingPlaylist = null;
  let editTitle = '';
  let editDesc = '';
  let friendSearch = '';
  let friendResults = [];
  let shareStatusByTrack = {};
  let searchingTracks = false;
  let searchingFriends = false;
  let previewId = null;
  let previewPausedId = null;
  let previewAudio = null;
  let previewCurrentTime = 0;
  let previewDuration = 0;
  let previewRafId = 0;

  const cancelRaf = () => { if (typeof cancelAnimationFrame !== 'undefined') cancelAnimationFrame(previewRafId); };
  const scheduleRaf = (fn) => { if (typeof requestAnimationFrame !== 'undefined') return requestAnimationFrame(fn); return 0; };

  const playlistDetailsCache = new Map();
  const trackSearchCache = new Map();
  const friendSearchCache = new Map();
  let trackSearchSeq = 0;
  let friendSearchSeq = 0;

  const debouncedSearchTracks = debounce((q) => {
    void searchTracks(q);
  }, 220);

  const debouncedSearchFriends = debounce((q) => {
    void searchFriends(q);
  }, 220);

  onMount(async () => {
    if (!$isAuthenticated) {
      goto('/auth');
      return;
    }
    try {
      playlists = await api.playlists.list();
    } catch {}
    loading = false;
    if (typeof window !== 'undefined') window.addEventListener('keydown', handleKeydown);
  });

  onDestroy(() => {
    debouncedSearchTracks.cancel();
    debouncedSearchFriends.cancel();
    stopPreview();
    if (typeof window !== 'undefined') window.removeEventListener('keydown', handleKeydown);
  });

  function handleKeydown(e) {
    if (e.code !== 'Space') return;
    const tag = e.target?.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || e.target?.isContentEditable) return;
    const activeId = previewId || previewPausedId;
    if (!activeId) return;
    const allTracks = [...searchResults, ...(selectedPlaylist?.tracks || [])];
    const t = allTracks.find((x) => x.id === activeId);
    if (t) { e.preventDefault(); previewTrack(t); }
  }

  function tickPreview() {
    if (previewAudio && !previewAudio.paused) {
      previewCurrentTime = previewAudio.currentTime;
      previewRafId = scheduleRaf(tickPreview);
    }
  }

  function stopPreview() {
    cancelRaf();
    if (previewAudio) {
      previewAudio.pause();
      previewAudio = null;
    }
    previewId = null;
    previewPausedId = null;
    previewCurrentTime = 0;
    previewDuration = 0;
  }

  function previewTrack(track) {
    if (previewId === track.id) {
      previewAudio.pause();
      previewPausedId = track.id;
      previewId = null;
      cancelRaf();
      return;
    }

    if (previewPausedId === track.id && previewAudio) {
      previewAudio.play();
      previewId = track.id;
      previewPausedId = null;
      previewRafId = scheduleRaf(tickPreview);
      return;
    }

    stopPreview();
    if (!track.preview_url) return;
    previewAudio = new Audio(track.preview_url);
    previewAudio.play();
    previewId = track.id;
    previewPausedId = null;
    previewCurrentTime = 0;
    previewDuration = 0;
    previewRafId = scheduleRaf(tickPreview);
    previewAudio.onloadedmetadata = () => {
      previewDuration = previewAudio.duration;
    };
    previewAudio.onended = () => { previewId = null; previewPausedId = null; previewCurrentTime = 0; previewDuration = 0; cancelRaf(); };
  }

  async function createPlaylist() {
    if (!newTitle) return;
    try {
      const p = await api.playlists.create({ title: newTitle, description: newDesc, is_public: newPublic });
      playlists = [p, ...playlists];
      playlistDetailsCache.set(p.id, p);
      showCreate = false;
      newTitle = '';
      newDesc = '';
      newPublic = false;
    } catch (e) {
      alert(e.message);
    }
  }

  async function selectPlaylist(p) {
    if (playlistDetailsCache.has(p.id)) {
      selectedPlaylist = playlistDetailsCache.get(p.id);
    } else {
      const full = await api.playlists.get(p.id);
      playlistDetailsCache.set(p.id, full);
      selectedPlaylist = full;
    }
    if (selectedPlaylist?.tracks?.length) {
      const resolved = await resolvePreviews(selectedPlaylist.tracks);
      selectedPlaylist = { ...selectedPlaylist, tracks: resolved };
    }
  }

  async function deletePlaylist(id) {
    if (!confirm('delete this playlist?')) return;
    await api.playlists.delete(id);
    playlists = playlists.filter(p => p.id !== id);
    playlistDetailsCache.delete(id);
    if (selectedPlaylist?.id === id) selectedPlaylist = null;
  }

  async function searchTracks(q) {
    const query = q.trim();
    if (!query) { searchResults = []; searchingTracks = false; return; }

    const key = query.toLowerCase();
    if (trackSearchCache.has(key)) {
      searchResults = trackSearchCache.get(key);
      searchingTracks = false;
      return;
    }

    const reqSeq = ++trackSearchSeq;
    searchingTracks = true;
    try {
      const result = await api.playlists.search(query);
      if (reqSeq !== trackSearchSeq) return;
      const resolved = await resolvePreviews(result || []);
      searchResults = resolved;
      trackSearchCache.set(key, resolved);
    } catch {
      if (reqSeq !== trackSearchSeq) return;
      searchResults = [];
    } finally {
      if (reqSeq === trackSearchSeq) {
        searchingTracks = false;
      }
    }
  }

  async function resolvePreviews(trackList) {
    const needs = trackList.filter((t) => !t.preview_url);
    if (!needs.length) return trackList;
    const body = needs.map((t) => ({ id: t.id, title: t.title, artist: t.artist }));
    try {
      const res = await api.previews.resolve(body);
      if (!res?.previews) return trackList;
      const urlMap = {};
      for (const p of res.previews) { if (p.preview_url) urlMap[p.id] = p.preview_url; }
      return trackList.map((t) => (urlMap[t.id] ? { ...t, preview_url: urlMap[t.id] } : t));
    } catch { return trackList; }
  }

  async function addTrack(trackId) {
    if (!selectedPlaylist?.id || !trackId) return;
    if (selectedPlaylist.tracks?.some(t => t.id === trackId)) return;
    try {
      await api.playlists.update(selectedPlaylist.id, { add_tracks: [trackId] });
      selectedPlaylist = await api.playlists.get(selectedPlaylist.id);
      playlistDetailsCache.set(selectedPlaylist.id, selectedPlaylist);
    } catch (e) {
      alert(e.message || 'failed to add track');
    }
  }

  async function removeTrack(trackId) {
    await api.playlists.update(selectedPlaylist.id, { remove_tracks: [trackId] });
    selectedPlaylist = await api.playlists.get(selectedPlaylist.id);
    playlistDetailsCache.set(selectedPlaylist.id, selectedPlaylist);
  }

  function startEdit(p) {
    editingPlaylist = p;
    editTitle = p.title;
    editDesc = p.description || '';
  }

  async function saveEdit() {
    await api.playlists.update(editingPlaylist.id, { title: editTitle, description: editDesc });
    selectedPlaylist = await api.playlists.get(editingPlaylist.id);
    playlistDetailsCache.set(selectedPlaylist.id, selectedPlaylist);
    playlists = playlists.map(p => p.id === editingPlaylist.id ? { ...p, title: editTitle, description: editDesc } : p);
    editingPlaylist = null;
  }

  async function searchFriends(q) {
    const query = q.trim();
    if (!query) { friendResults = []; searchingFriends = false; return; }

    const key = query.toLowerCase();
    if (friendSearchCache.has(key)) {
      friendResults = friendSearchCache.get(key);
      searchingFriends = false;
      return;
    }

    const reqSeq = ++friendSearchSeq;
    searchingFriends = true;
    try {
      const result = await api.friends.search(query);
      if (reqSeq !== friendSearchSeq) return;
      friendResults = result;
      friendSearchCache.set(key, result);
    } catch {
      if (reqSeq !== friendSearchSeq) return;
      friendResults = [];
    } finally {
      if (reqSeq === friendSearchSeq) {
        searchingFriends = false;
      }
    }
  }

  async function addCollaborator(userId) {
    await api.playlists.addCollaborator(selectedPlaylist.id, { user_id: userId, permission: 'edit' });
    friendResults = [];
    friendSearch = '';
    selectedPlaylist = await api.playlists.get(selectedPlaylist.id);
    playlistDetailsCache.set(selectedPlaylist.id, selectedPlaylist);
  }

  function setShareStatus(trackId, status) {
    shareStatusByTrack = { ...shareStatusByTrack, [trackId]: status };
    setTimeout(() => {
      if (shareStatusByTrack[trackId] === status) {
        const next = { ...shareStatusByTrack };
        delete next[trackId];
        shareStatusByTrack = next;
      }
    }, 1800);
  }

  async function handleTrackShare(track) {
    try {
      const result = await shareTrack(track, 'song_link');
      if (result.ok && result.method === 'native') {
        setShareStatus(track.id, 'shared');
        return;
      }
      if (result.ok && result.method === 'clipboard') {
        setShareStatus(track.id, 'copied');
        return;
      }
      if (!result.cancelled) {
        setShareStatus(track.id, 'unavailable');
      }
    } catch {
      setShareStatus(track.id, 'failed');
    }
  }
</script>

<div class="playlists-page">
  <div class="page-header">
    <h1>playlists</h1>
    <button class="btn-primary" on:click={() => showCreate = !showCreate}>
      {showCreate ? 'cancel' : 'new playlist'}
    </button>
  </div>

  {#if showCreate}
    <div class="create-form">
      <input type="text" placeholder="playlist title" bind:value={newTitle} />
      <input type="text" placeholder="description (optional)" bind:value={newDesc} />
      <label class="checkbox-label">
        <input type="checkbox" bind:checked={newPublic} />
        public
      </label>
      <button class="btn-primary" on:click={createPlaylist}>create</button>
    </div>
  {/if}

  {#if loading}
    <div class="loading">loading...</div>
  {:else if playlists.length === 0}
    <div class="empty">no playlists yet. create one to get started.</div>
  {:else}
    <div class="playlist-layout">
      <div class="playlist-list">
        {#each playlists as p}
          <div
            class="playlist-item"
            class:active={selectedPlaylist?.id === p.id}
            on:click={() => selectPlaylist(p)}
          >
            <div class="playlist-info">
              <strong>{p.title}</strong>
              <span class="playlist-meta">
                {p.is_public ? 'public' : 'private'}
                {#if p.owner_id !== $user?.id}
                  · shared with you
                {/if}
              </span>
            </div>
            <button class="btn-icon" on:click|stopPropagation={() => deletePlaylist(p.id)}>✕</button>
          </div>
        {/each}
      </div>

      {#if selectedPlaylist}
        <div class="playlist-detail">
          <div class="detail-header">
            <div>
              <h2>{selectedPlaylist.title}</h2>
              <p class="desc">{selectedPlaylist.description}</p>
            </div>
            <div class="detail-actions">
              <button class="btn-outline" on:click={() => startEdit(selectedPlaylist)}>edit</button>
            </div>
          </div>

          {#if editingPlaylist?.id === selectedPlaylist.id}
            <div class="edit-form" transition:slide={{ duration: 140 }}>
              <input type="text" bind:value={editTitle} placeholder="title" />
              <input type="text" bind:value={editDesc} placeholder="description" />
              <button class="btn-primary" on:click={saveEdit}>save</button>
            </div>
          {/if}

          <div class="add-track-section">
            <h3>add tracks</h3>
            <input
              type="text"
              placeholder="search tracks..."
              bind:value={searchQuery}
              on:input={() => debouncedSearchTracks(searchQuery)}
            />
            {#if searchingTracks || searchResults.length > 0 || searchQuery.trim()}
              <div class="search-results" transition:slide={{ duration: 140 }}>
                {#if searchingTracks}
                  <div class="search-hint">searching tracks...</div>
                {/if}
                {#each searchResults as track}
                  <div class="search-track" on:click={() => addTrack(track.id)}>
                    <span>{track.title} — {track.artist}</span>
                    <div class="search-track-actions">
                      {#if track.preview_url}
                        <button class="btn-preview" on:click|stopPropagation={() => previewTrack(track)}>
                          {previewId === track.id ? '■' : '▶'}
                        </button>
                      {/if}
                      <button class="btn-add" on:click|stopPropagation={() => addTrack(track.id)}>+</button>
                    </div>
                  </div>
                {/each}
                {#if !searchingTracks && searchQuery.trim() && searchResults.length === 0}
                  <div class="search-hint">no tracks found</div>
                {/if}
              </div>
            {/if}
          </div>

          {#if selectedPlaylist.tracks?.length > 0}
            <div class="tracks">
              <h3>tracks ({selectedPlaylist.tracks.length})</h3>
              {#each selectedPlaylist.tracks as track}
                <div class="track" class:track-active={previewId === track.id || previewPausedId === track.id}>
                  <div class="track-art">
                    {#if track.album_art_url}
                      <img src={track.album_art_url} alt="" />
                    {/if}
                  </div>
                  <div class="track-info">
                    <span class="track-title">{track.title}</span>
                    <span class="track-artist">{track.artist}</span>
                    {#if previewId === track.id || previewPausedId === track.id}
                      <div class="track-progress">
                        <div class="track-progress-bar">
                          <div class="track-progress-fill" style="width: {previewDuration > 0 ? (previewCurrentTime / previewDuration) * 100 : 0}%"></div>
                        </div>
                      </div>
                    {/if}
                  </div>
                  {#if track.preview_url}
                    <button class="btn-preview" class:btn-preview-active={previewId === track.id || previewPausedId === track.id} on:click={() => previewTrack(track)}>
                      {previewId === track.id ? '■' : '▶'}
                    </button>
                  {/if}
                  <button class="btn-track-share" on:click={() => handleTrackShare(track)}>
                    {shareStatusByTrack[track.id] || 'share'}
                  </button>
                  <button class="btn-icon" on:click={() => removeTrack(track.id)}>✕</button>
                </div>
              {/each}
            </div>
          {:else}
            <p class="empty">no tracks in this playlist</p>
          {/if}

          <div class="collaborators">
            <h3>collaborators</h3>
            <input
              type="text"
              placeholder="search friends to add..."
              bind:value={friendSearch}
              on:input={() => debouncedSearchFriends(friendSearch)}
            />
            {#if searchingFriends || friendResults.length > 0 || friendSearch.trim()}
              <div class="friend-results" transition:slide={{ duration: 140 }}>
                {#if searchingFriends}
                  <div class="search-hint">searching friends...</div>
                {/if}
                {#each friendResults as friend}
                  <div class="friend-item" on:click={() => addCollaborator(friend.id)}>
                    <span>{friend.username}</span>
                    <button class="btn-add">add</button>
                  </div>
                {/each}
                {#if !searchingFriends && friendSearch.trim() && friendResults.length === 0}
                  <div class="search-hint">no friends found</div>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  @keyframes fade-up {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .playlists-page {
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 0.35rem 1.5rem;
  }
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.25rem;
    gap: 0.6rem;
    flex-wrap: wrap;
  }
  .page-header h1 {
    margin: 0;
    font-size: 1.8rem;
  }
  .btn-primary {
    background: #818cf8;
    color: #000;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-weight: 600;
  }
  .btn-primary:hover { background: #a5b4fc; transform: translateY(-1px); }
  .btn-outline {
    background: transparent;
    color: #a0a0a0;
    border: 1px solid #333;
    padding: 0.4rem 0.8rem;
    border-radius: 6px;
    cursor: pointer;
  }
  .btn-outline:hover { border-color: #666; color: #fff; }
  .btn-icon {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    padding: 0.2rem 0.4rem;
    font-size: 0.9rem;
  }
  .btn-icon:hover { color: #f87171; }
  .btn-add {
    background: none;
    border: 1px solid #444;
    color: #818cf8;
    padding: 0.2rem 0.6rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
  }
  .btn-add:hover { border-color: #818cf8; }
  .btn-track-share {
    background: none;
    border: 1px solid #4c1d95;
    color: #c4b5fd;
    padding: 0.2rem 0.55rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: lowercase;
    min-width: 64px;
  }
  .btn-track-share:hover {
    border-color: #7c3aed;
    color: #ddd6fe;
  }
  .loading, .empty {
    color: #888;
    text-align: center;
    padding: 3rem;
  }
  .create-form {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
    align-items: flex-end;
    flex-wrap: wrap;
  }
  .create-form input[type="text"] {
    padding: 0.5rem 0.8rem;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 6px;
    color: #fff;
    font-size: 0.9rem;
  }
  .create-form input[type="text"]:focus { outline: none; border-color: #818cf8; }
  .checkbox-label {
    font-size: 0.85rem;
    color: #888;
    display: flex;
    align-items: center;
    gap: 0.3rem;
  }
  .playlist-layout {
    display: grid;
    grid-template-columns: 300px 1fr;
    gap: 1.5rem;
  }
  .playlist-list {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  .playlist-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.7rem;
    background: #111;
    border-radius: 6px;
    border: 1px solid #222;
    cursor: pointer;
    transition: all 0.15s;
  }
  .playlist-item:hover { background: #181818; transform: translateY(-1px); }
  .playlist-item.active { border-color: #818cf8; }
  .playlist-info strong { display: block; font-size: 0.9rem; }
  .playlist-meta { font-size: 0.75rem; color: #666; }
  .playlist-detail {
    background: #111;
    border-radius: 8px;
    padding: 1.5rem;
    border: 1px solid #222;
  }
  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }
  .detail-header h2 { margin: 0; font-size: 1.3rem; }
  .desc { color: #888; font-size: 0.85rem; margin: 0.3rem 0 1rem; }
  .detail-actions { display: flex; gap: 0.5rem; }
  .edit-form { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
  .edit-form input {
    padding: 0.4rem 0.7rem;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 6px;
    color: #fff;
  }
  .add-track-section { margin-bottom: 1.5rem; }
  .add-track-section h3 { font-size: 0.9rem; color: #888; margin: 0 0 0.5rem; }
  .add-track-section input {
    width: 100%;
    padding: 0.5rem 0.8rem;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 6px;
    color: #fff;
  }
  .search-results, .friend-results {
    margin-top: 0.3rem;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    max-height: 200px;
    overflow-y: auto;
  }
  .search-track, .friend-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.4rem 0.6rem;
    background: #1a1a1a;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85rem;
  }
  .search-track:hover, .friend-item:hover { background: #222; }
  .search-hint { color: #777; font-size: 0.8rem; padding: 0.2rem 0.25rem; }
  .tracks { margin-bottom: 1.5rem; }
  .tracks h3 { font-size: 0.9rem; color: #888; margin: 0 0 0.5rem; }
  .track {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.45rem 0;
    border-bottom: 1px solid #1d1d1d;
  }
  .track-art img {
    width: 32px;
    height: 32px;
    border-radius: 4px;
    object-fit: cover;
  }
  .track-info { flex: 1; }
  .track-title { font-size: 0.85rem; }
  .track-artist { font-size: 0.75rem; color: #888; display: block; }
  .btn-preview {
    background: none;
    border: 1px solid #444;
    color: #fff;
    width: 26px;
    height: 26px;
    border-radius: 50%;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.65rem;
    flex-shrink: 0;
  }
  .btn-preview:hover { border-color: #818cf8; }
  .btn-preview-active { border-color: #818cf8; background: rgba(129, 140, 248, 0.15); }
  .track-active { background: rgba(129, 140, 248, 0.03); }
  .track-progress { margin-top: 4px; }
  .track-progress-bar { width: 100%; height: 2px; background: #333; border-radius: 1px; overflow: hidden; }
  .track-progress-fill { height: 100%; background: #818cf8; border-radius: 1px; }
  .search-track-actions { display: flex; align-items: center; gap: 0.3rem; }
  .collaborators h3 { font-size: 0.9rem; color: #888; margin: 0 0 0.5rem; }
  .collaborators input {
    width: 100%;
    padding: 0.5rem 0.8rem;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 6px;
    color: #fff;
  }

  @media (max-width: 940px) {
    .playlist-layout {
      grid-template-columns: 1fr;
    }
    .playlist-list {
      max-height: 260px;
      overflow-y: auto;
      padding-right: 0.2rem;
    }
  }

  @media (max-width: 680px) {
    .playlists-page { padding: 0 0.05rem 1.2rem; }
    .playlist-detail { padding: 1rem; }
    .edit-form { flex-wrap: wrap; }
    .detail-header { flex-direction: column; gap: 0.75rem; }
    .track { gap: 0.45rem; }
    .btn-track-share { min-width: 58px; font-size: 0.68rem; }
  }
</style>
