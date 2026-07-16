<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { isAuthenticated } from '$lib/stores/auth';
  import { goto } from '$app/navigation';

  let openPlatform = 'spotify';
  let lastfmUsername = '';
  let saving = false;
  let saved = false;
  let loaded = false;

  onMount(async () => {
    if (!$isAuthenticated) { goto('/auth'); return; }
    try {
      const settings = await api.settings.getAll();
      if (settings.open_platform) openPlatform = settings.open_platform;
      if (settings.lastfm_username) lastfmUsername = settings.lastfm_username;
    } catch {}
    loaded = true;
  });

  async function saveOpenPlatform() {
    saving = true;
    saved = false;
    try {
      await api.settings.set('open_platform', openPlatform);
      saved = true;
    } catch (e) { alert(e.message); }
    saving = false;
  }

  async function saveLastfm() {
    saving = true;
    saved = false;
    try {
      await api.settings.set('lastfm_username', lastfmUsername);
      saved = true;
    } catch (e) { alert(e.message); }
    saving = false;
  }
</script>

<div class="settings-page">
  <h1>settings</h1>

  {#if !loaded}
    <div class="loading">loading...</div>
  {:else}
    <div class="settings-section">
      <h2>open platform</h2>
      <p class="hint">choose where links open when you click a track</p>
      <div class="platform-grid">
        {#each [{ id: 'spotify', label: 'spotify', desc: 'open directly in spotify' }, { id: 'apple_music', label: 'apple music', desc: 'open in apple music' }, { id: 'song_link', label: 'song.link', desc: 'open a smart link' }] as p}
          <button
            class="platform-card"
            class:active={openPlatform === p.id}
            on:click={() => { openPlatform = p.id; saveOpenPlatform(); }}
          >
            <span class="platform-name">{p.label}</span>
            <span class="platform-desc">{p.desc}</span>
            {#if openPlatform === p.id}
              <span class="check">&#10003;</span>
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <div class="settings-section">
      <h2>last.fm</h2>
      <p class="hint">connect your last.fm account to include scrobble data in taste analysis</p>
      <div class="input-row">
        <input type="text" placeholder="last.fm username" bind:value={lastfmUsername} />
        <button class="btn-primary" on:click={saveLastfm} disabled={saving}>
          {saving ? 'saving...' : 'save'}
        </button>
      </div>
      {#if saved}
        <span class="saved-hint">saved</span>
      {/if}
    </div>
  {/if}
</div>

<style>
  .settings-page { max-width: 600px; margin: 0 auto; }
  .settings-page h1 { font-size: 1.8rem; margin: 0 0 2rem; }
  .settings-page h2 { font-size: 1rem; margin: 0 0 0.25rem; }
  .loading { color: #888; padding: 4rem; text-align: center; }
  .hint { font-size: 0.8rem; color: #666; margin: 0 0 1rem; }

  .settings-section { margin-bottom: 2.5rem; padding-bottom: 2rem; border-bottom: 1px solid #222; }
  .settings-section:last-child { border-bottom: none; }

  .platform-grid { display: flex; gap: 0.75rem; flex-wrap: wrap; }
  .platform-card { flex: 1; min-width: 160px; background: #111; border: 1px solid #333; border-radius: 10px; padding: 1rem; text-align: center; cursor: pointer; transition: all 0.15s; position: relative; }
  .platform-card:hover { background: #151515; border-color: #555; }
  .platform-card.active { border-color: #818cf8; background: #1a1a1a; }
  .platform-name { display: block; font-weight: 700; font-size: 0.9rem; margin-bottom: 0.25rem; }
  .platform-desc { display: block; font-size: 0.75rem; color: #666; }
  .check { position: absolute; top: 6px; right: 8px; color: #818cf8; font-size: 0.9rem; }

  .input-row { display: flex; gap: 0.5rem; align-items: center; }
  .input-row input { flex: 1; padding: 0.6rem 1rem; background: #151515; border: 1px solid #333; border-radius: 6px; color: #fff; font-size: 0.9rem; }
  .input-row input:focus { outline: none; border-color: #818cf8; }
  .btn-primary { background: #818cf8; color: #000; border: none; padding: 0.6rem 1.5rem; border-radius: 6px; font-weight: 600; cursor: pointer; white-space: nowrap; }
  .btn-primary:hover { background: #a5b4fc; }
  .btn-primary:disabled { opacity: 0.5; }
  .saved-hint { font-size: 0.8rem; color: #4caf50; margin-top: 0.5rem; display: inline-block; }
</style>
