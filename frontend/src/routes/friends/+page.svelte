<script>
  import { onDestroy, onMount } from 'svelte';
  import { slide } from 'svelte/transition';
  import { api } from '$lib/api/client';
  import { isAuthenticated } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { debounce } from '$lib/utils/debounce';

  let friends = [];
  let requests = [];
  let searchQuery = '';
  let searchResults = [];
  let loading = true;
  let searching = false;
  let searchSeq = 0;
  const searchCache = new Map();

  const debouncedSearchUsers = debounce((q) => {
    void searchUsers(q);
  }, 220);

  onMount(async () => {
    if (!$isAuthenticated) { goto('/auth'); return; }
    await loadData();
  });

  onDestroy(() => {
    debouncedSearchUsers.cancel();
  });

  async function loadData() {
    const [fRes, rRes] = await Promise.allSettled([
      api.friends.list(),
      api.friends.requests(),
    ]);
    if (fRes.status === 'fulfilled') friends = fRes.value;
    if (rRes.status === 'fulfilled') requests = rRes.value;
    loading = false;
  }

  async function searchUsers(q) {
    const query = q.trim();
    if (!query) { searchResults = []; searching = false; return; }

    const key = query.toLowerCase();
    if (searchCache.has(key)) {
      searchResults = searchCache.get(key);
      searching = false;
      return;
    }

    const reqSeq = ++searchSeq;
    searching = true;
    try {
      const result = await api.friends.search(query);
      if (reqSeq !== searchSeq) return;
      searchResults = result;
      searchCache.set(key, result);
    } catch {
      if (reqSeq !== searchSeq) return;
      searchResults = [];
    } finally {
      if (reqSeq === searchSeq) {
        searching = false;
      }
    }
  }

  async function sendRequest(friendId) {
    try {
      await api.friends.send(friendId);
      searchResults = [];
      searchQuery = '';
    } catch (e) { alert(e.message); }
  }

  async function acceptRequest(friendId) {
    await api.friends.accept(friendId);
    await loadData();
  }

  async function removeFriend(friendId) {
    if (!confirm('remove friend?')) return;
    await api.friends.remove(friendId);
    await loadData();
  }
</script>

<div class="friends-page">
  <h1>friends</h1>

  <div class="search-section">
    <input
      type="text"
      placeholder="search users by username or email..."
      bind:value={searchQuery}
      on:input={() => debouncedSearchUsers(searchQuery)}
    />
    {#if searching || searchResults.length > 0 || searchQuery.trim()}
      <div class="search-results" transition:slide={{ duration: 140 }}>
        {#if searching}
          <div class="searching-hint">searching...</div>
        {/if}
        {#each searchResults as user}
          <div class="user-item">
            <span class="user-name">{user.username}</span>
            <span class="user-email">{user.email}</span>
            <button class="btn-primary btn-sm" on:click={() => sendRequest(user.id)}>
              add friend
            </button>
          </div>
        {/each}
        {#if !searching && searchQuery.trim() && searchResults.length === 0}
          <div class="searching-hint">no users found</div>
        {/if}
      </div>
    {/if}
  </div>

  {#if requests.length > 0}
    <div class="section">
      <h2>pending requests ({requests.length})</h2>
      <div class="request-list">
        {#each requests as req}
          <div class="request-item">
            <span>{req.username}</span>
            <div class="request-actions">
              <button class="btn-primary btn-sm" on:click={() => acceptRequest(req.id)}>accept</button>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  {#if loading}
    <div class="loading">loading...</div>
  {:else if friends.length === 0}
    <div class="empty">no friends yet. search for users to add!</div>
  {:else}
    <div class="section">
      <h2>your friends ({friends.length})</h2>
      <div class="friend-list">
        {#each friends as friend}
          <div class="friend-card">
            <div class="friend-avatar">
              {#if friend.avatar_url}
                <img src={friend.avatar_url} alt="" />
              {:else}
                <div class="avatar-placeholder">{friend.username[0]}</div>
              {/if}
            </div>
            <div class="friend-info">
              <strong>{friend.username}</strong>
              <span class="friend-email">{friend.email}</span>
            </div>
            <div class="friend-actions">
              <a href="/tastes?user={friend.id}" class="btn-outline btn-sm">taste</a>
              <button class="btn-icon" on:click={() => removeFriend(friend.id)}>✕</button>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  @keyframes fade-up {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .friends-page { max-width: 760px; margin: 0 auto; padding: 0 0.25rem 1.5rem; }
  .friends-page h1 { font-size: 1.8rem; margin: 0 0 1.5rem; }
  .section { margin-bottom: 2rem; }
  .section h2 { font-size: 1rem; color: #888; margin: 0 0 0.75rem; }
  .search-section { margin-bottom: 2rem; }
  .search-section input {
    width: 100%;
    padding: 0.6rem 1rem;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    color: #fff;
    font-size: 0.95rem;
  }
  .search-section input:focus { outline: none; border-color: #818cf8; }
  .search-results {
    margin-top: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  .user-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 0.8rem;
    background: #151515;
    border-radius: 6px;
    transition: transform 0.12s, background 0.12s;
  }
  .user-item:hover { background: #1c1c1c; transform: translateY(-1px); }
  .user-name { flex: 1; font-weight: 500; }
  .user-email { color: #888; font-size: 0.85rem; }
  .request-list { display: flex; flex-direction: column; gap: 0.3rem; }
  .request-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.6rem 0.8rem;
    background: #151515;
    border-radius: 6px;
    transition: transform 0.12s, background 0.12s;
  }
  .request-item:hover { background: #1c1c1c; transform: translateY(-1px); }
  .friend-list { display: flex; flex-direction: column; gap: 0.3rem; }
  .friend-card {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 0.8rem;
    background: #151515;
    border-radius: 6px;
    transition: transform 0.12s, background 0.12s;
  }
  .friend-card:hover { background: #1c1c1c; transform: translateY(-1px); }
  .friend-avatar img, .avatar-placeholder {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    object-fit: cover;
  }
  .avatar-placeholder {
    background: #818cf8;
    color: #000;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 1rem;
  }
  .friend-info { flex: 1; }
  .friend-info strong { display: block; font-size: 0.9rem; }
  .friend-email { font-size: 0.8rem; color: #888; }
  .friend-actions { display: flex; gap: 0.4rem; align-items: center; }
  .btn-primary { background: #818cf8; color: #000; border: none; border-radius: 6px; cursor: pointer; font-weight: 600; }
  .btn-primary:hover { background: #a5b4fc; transform: translateY(-1px); }
  .btn-outline { background: transparent; color: #a0a0a0; border: 1px solid #333; border-radius: 6px; cursor: pointer; text-decoration: none; }
  .btn-outline:hover { border-color: #666; color: #fff; }
  .btn-sm { font-size: 0.8rem; padding: 0.35rem 0.7rem; }
  .btn-icon { background: none; border: none; color: #666; cursor: pointer; padding: 0.2rem; }
  .btn-icon:hover { color: #f87171; }
  .searching-hint { color: #777; font-size: 0.8rem; padding: 0.15rem 0.2rem; }
  .loading, .empty { color: #888; text-align: center; padding: 3rem; }

  @media (max-width: 680px) {
    .friends-page { padding: 0 0.05rem 1.2rem; }
    .user-item, .request-item, .friend-card { padding: 0.55rem 0.65rem; }
    .user-email { display: none; }
  }
</style>
