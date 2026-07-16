<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { token, user, isAuthenticated, fetchUser } from '$lib/stores/auth';
  import { api } from '$lib/api/client';

  let accounts = [];
  let linking = false;

  onMount(async () => {
    if ($token) {
      try {
        await Promise.all([fetchUser(), loadAccounts()]);
      } catch {
        token.set('');
        user.set(null);
      }
    }
  });

  async function loadAccounts() {
    try {
      accounts = await api.auth.accounts();
    } catch {}
  }

  async function handleLinkSpotify() {
    linking = true;
    try {
      const { url } = await api.auth.spotifyAuthorize();
      window.location.href = url;
    } catch (e) {
      alert(e.message);
      linking = false;
    }
  }

  async function handleUnlink(service) {
    await api.auth.unlink(service);
    accounts = accounts.filter(a => a.service !== service);
  }

  async function handleLogout() {
    token.set('');
    user.set(null);
    accounts = [];
  }
</script>

<div class="app-shell">
  <nav class="sidebar">
    <div class="logo">
      <a href="/">FISHER</a>
    </div>
    <div class="nav-links">
      <a href="/genres" class="nav-link" class:active={$page.url.pathname.startsWith('/genres')}>explore</a>
      <a href="/playlists" class="nav-link" class:active={$page.url.pathname.startsWith('/playlists')}>playlists</a>
      <a href="/friends" class="nav-link" class:active={$page.url.pathname.startsWith('/friends')}>friends</a>
      <a href="/tastes" class="nav-link" class:active={$page.url.pathname.startsWith('/tastes')}>my taste</a>
      <a href="/settings" class="nav-link" class:active={$page.url.pathname.startsWith('/settings')}>settings</a>
    </div>
    <div class="auth-section">
      {#if $isAuthenticated && $user}
        <div class="user-info">
          <span class="username">{$user.username}</span>
          <button class="btn-outline btn-sm" on:click={handleLogout}>logout</button>
        </div>
        <div class="connected-accounts">
          <span class="section-label">connected services</span>
          <div class="account-row">
            <span class="service-name spotify">spotify</span>
            {#if accounts.find(a => a.service === 'spotify')}
              <span class="linked-badge">linked</span>
              <button class="btn-outline btn-xs" on:click={() => handleUnlink('spotify')}>unlink</button>
            {:else}
              <button class="btn-spotify btn-xs" on:click={handleLinkSpotify} disabled={linking}>
                {linking ? 'connecting...' : 'connect'}
              </button>
            {/if}
          </div>
          <div class="account-row">
            <span class="service-name apple">apple music</span>
            {#if accounts.find(a => a.service === 'apple')}
              <span class="linked-badge">linked</span>
              <button class="btn-outline btn-xs" on:click={() => handleUnlink('apple')}>unlink</button>
            {:else}
              <span class="not-available">search available</span>
            {/if}
          </div>
        </div>
      {:else if $isAuthenticated}
        <div class="user-info">loading...</div>
      {:else}
        <div class="auth-buttons">
          <a href="/auth" class="btn-primary btn-sm">sign in</a>
        </div>
      {/if}
    </div>
  </nav>
  <main class="main-content">
    <div class="page-slot">
      <slot />
    </div>
  </main>
</div>

<style>
  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #0a0a0a;
    color: #e0e0e0;
  }
  :global(*) {
    box-sizing: border-box;
  }
  :global(a) {
    color: #818cf8;
    text-decoration: none;
  }
  :global(a:hover) {
    color: #a5b4fc;
  }
  .app-shell {
    display: flex;
    min-height: 100vh;
  }
  .sidebar {
    width: 228px;
    background: #101010;
    border-right: 1px solid #222;
    padding: 1.4rem 1.1rem;
    display: flex;
    flex-direction: column;
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    gap: 0.2rem;
  }
  .logo a {
    font-size: 1.45rem;
    font-weight: 800;
    letter-spacing: 0.15em;
    color: #818cf8;
    display: block;
    text-align: center;
    margin-bottom: 1.5rem;
  }
  .nav-links {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    flex: 1;
  }
  .nav-link {
    padding: 0.52rem 0.75rem;
    border-radius: 8px;
    color: #a0a0a0;
    font-size: 0.92rem;
    transition: background 0.15s, color 0.15s, transform 0.15s;
  }
  .nav-link:hover {
    background: #1a1a1a;
    color: #fff;
    transform: translateY(-1px);
  }
  .nav-link.active {
    background: #1a1a1a;
    color: #dbe0ff;
    border: 1px solid #2d325f;
  }
  .auth-section {
    padding-top: 1rem;
    border-top: 1px solid #222;
  }
  .user-info {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }
  .username {
    font-size: 0.85rem;
    color: #a0a0a0;
  }
  .auth-buttons {
    display: flex;
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
  .btn-primary:hover {
    background: #a5b4fc;
  }
  .btn-outline {
    background: transparent;
    color: #a0a0a0;
    border: 1px solid #333;
    padding: 0.35rem 0.75rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.8rem;
  }
  .btn-outline:hover {
    border-color: #666;
    color: #fff;
  }
  .btn-sm {
    font-size: 0.8rem;
    padding: 0.35rem 0.75rem;
  }
  .btn-xs {
    font-size: 0.7rem;
    padding: 0.2rem 0.5rem;
  }
  .connected-accounts {
    padding-top: 0.75rem;
    margin-top: 0.75rem;
    border-top: 1px solid #222;
  }
  .section-label {
    font-size: 0.7rem;
    color: #555;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    display: block;
    margin-bottom: 0.5rem;
  }
  .account-row {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin-bottom: 0.4rem;
    font-size: 0.8rem;
  }
  .service-name {
    width: 80px;
    font-weight: 500;
  }
  .service-name.spotify { color: #1db954; }
  .service-name.apple { color: #fa243c; }

  .linked-badge {
    color: #4caf50;
    font-size: 0.7rem;
  }
  .not-available {
    color: #555;
    font-size: 0.7rem;
    font-style: italic;
  }
  .btn-spotify {
    background: #1db954;
    color: #000;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 600;
  }
  .btn-spotify:hover { background: #1ed760; }
  .btn-spotify:disabled { opacity: 0.5; }
  .main-content {
    margin-left: 228px;
    flex: 1;
    padding: 1.75rem 1.75rem 2rem;
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  .page-slot { width: 100%; max-width: 1400px; }

  @media (max-width: 940px) {
    .app-shell {
      flex-direction: column;
    }

    .sidebar {
      position: sticky;
      top: 0;
      width: 100%;
      height: auto;
      z-index: 40;
      border-right: none;
      border-bottom: 1px solid #222;
      padding: 0.8rem 0.8rem 0.7rem;
      background: rgba(16, 16, 16, 0.92);
      backdrop-filter: blur(10px);
    }

    .logo a {
      margin-bottom: 0.65rem;
      font-size: 1.15rem;
    }

    .nav-links {
      flex-direction: row;
      gap: 0.35rem;
      overflow-x: auto;
      padding-bottom: 0.2rem;
      margin-bottom: 0.4rem;
    }

    .nav-link {
      white-space: nowrap;
      font-size: 0.84rem;
      padding: 0.42rem 0.65rem;
    }

    .auth-section {
      padding-top: 0.5rem;
    }

    .main-content {
      margin-left: 0;
      max-width: 100%;
      padding: 1rem 0.9rem 1.5rem;
      align-items: stretch;
    }
  }
</style>
