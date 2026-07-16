<script>
  import { onDestroy, onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { isAuthenticated, user } from '$lib/stores/auth';
  import { debug } from '$lib/utils/debug';
  import { topGenres } from '$lib/utils/taste';
  import GenreCards from '$lib/components/tastes/GenreCards.svelte';
  import GenreEditor from '$lib/components/tastes/GenreEditor.svelte';
  import AudioStats from '$lib/components/tastes/AudioStats.svelte';
  import ArtistPills from '$lib/components/tastes/ArtistPills.svelte';
  import TopTracks from '$lib/components/tastes/TopTracks.svelte';
  import FriendsTasteGrid from '$lib/components/tastes/FriendsTasteGrid.svelte';
  import AnalyzeControls from '$lib/components/tastes/AnalyzeControls.svelte';

  let profile = null;
  let genres = [];
  let friends = [];
  let accounts = [];

  let loading = true;
  let saving = false;
  let analyzing = false;
  let edit = false;
  let dirty = false;

  let viewId = null;
  let weights = {};
  let time = 'medium_term';
  let source = 'spotify';
  let previewId = null;
  let previewPausedId = null;
  let previewAudio = null;
  let previewCurrentTime = 0;
  let previewDuration = 0;
  let previewRafId = 0;

  const cancelRaf = () => { if (typeof cancelAnimationFrame !== 'undefined') cancelAnimationFrame(previewRafId); };
  const scheduleRaf = (fn) => { if (typeof requestAnimationFrame !== 'undefined') return requestAnimationFrame(fn); return 0; };

  const initWeights = () => {
    weights = {};
    if (!profile?.genres) return;
    profile.genres.forEach((g) => {
      weights[g.genre_id] = Math.round(g.weight * 100);
    });
  };

  async function resolveTopTracks(p) {
    if (!p?.top_tracks?.length) return p;
    const needs = p.top_tracks.filter((t) => !t.preview_url);
    if (!needs.length) return p;
    const body = needs.map((t) => ({ id: t.id || t.spotify_uri || t.title, title: t.title, artist: t.artist }));
    try {
      const res = await api.previews.resolve(body);
      if (!res?.previews) return p;
      const urlMap = {};
      for (const pr of res.previews) { if (pr.preview_url) urlMap[pr.id] = pr.preview_url; }
      const resolved = p.top_tracks.map((t) => {
        const key = t.id || t.spotify_uri || t.title;
        return urlMap[key] ? { ...t, preview_url: urlMap[key] } : t;
      });
      return { ...p, top_tracks: resolved };
    } catch { return p; }
  }

  onMount(async () => {
    if (!$isAuthenticated) {
      goto('/auth');
      return;
    }

    viewId = $page.url.searchParams.get('user');

    const [gRes, aRes] = await Promise.allSettled([
      api.tastes.genres(),
      api.auth.accounts(),
    ]);

    if (gRes.status === 'fulfilled') genres = gRes.value;
    else debug('failed to load genres');

    if (aRes.status === 'fulfilled') accounts = aRes.value;
    else debug('failed to load accounts');

    if (viewId && viewId !== $user?.id) {
      try {
        profile = await api.tastes.profile(viewId);
        profile = await resolveTopTracks(profile);
      } catch {
        debug('failed to load friend profile');
      }
    } else {
      const [pRes, fRes] = await Promise.allSettled([
        api.tastes.profile(),
        api.tastes.friendsTastes(),
      ]);

      if (pRes.status === 'fulfilled') {
        profile = pRes.value;
        profile = await resolveTopTracks(profile);
      } else debug('failed to load profile');

      if (fRes.status === 'fulfilled') friends = fRes.value;
      else debug('failed to load friends tastes');

      initWeights();
    }

    loading = false;
    if (typeof window !== 'undefined') window.addEventListener('keydown', handleKeydown);
    debug('tastes page loaded', { viewId, hasProfile: !!profile, genreCount: profile?.genres?.length });
  });

  const hasSpotify = () => accounts.some((a) => a.service === 'spotify');

  const gName = (id) => genres.find((g) => g.id === id)?.name || id;
  const gColor = (id) => genres.find((g) => g.id === id)?.color || '#6366f1';

  const hasW = () => {
    return Object.values(weights).some((w) => w > 0) || (profile?.genres?.length > 0);
  };

  function setW(genreId, val) {
    weights[genreId] = Math.max(0, Math.min(100, val));
    dirty = true;
  }

  async function save() {
    saving = true;

    const data = Object.entries(weights)
      .filter(([, w]) => w > 0)
      .map(([genre_id, weight]) => ({ genre_id, weight: weight / 100 }));

    try {
      await api.tastes.updateGenres(data);
      profile = await api.tastes.profile();
      initWeights();
      dirty = false;
      edit = false;
    } catch (e) {
      alert(e.message);
    }

    saving = false;
  }

  async function analyze() {
    analyzing = true;

    try {
      debug('analyzing taste', { time, source });
      const result = await api.tastes.analyze({ time_range: time, source });
      profile = await resolveTopTracks(result);
      initWeights();
      dirty = false;
    } catch (e) {
      debug('analyze failed', e.message);
      if (e.message === 'spotify_reauth_required') {
        alert('Spotify authorization needs to be refreshed. Please unlink and reconnect Spotify in the sidebar.');
        try { accounts = await api.auth.accounts(); } catch {}
      } else if (e.message.includes('last.fm')) {
        alert(e.message);
      } else {
        alert(e.message);
      }
    }

    analyzing = false;
  }

  function openTrack(track) {
    let url = '#';
    if (track.spotify_uri) {
      url = `https://open.spotify.com/track/${track.spotify_uri.split(':').pop()}`;
    }
    window.open(url, '_blank');
  }

  function handleKeydown(e) {
    if (e.code !== 'Space') return;
    const tag = e.target?.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || e.target?.isContentEditable) return;
    const activeId = previewId || previewPausedId;
    if (!activeId) return;
    const t = (profile?.top_tracks || []).find((x) => x.id === activeId);
    if (t) { e.preventDefault(); previewTrack(t); }
  }

  onDestroy(() => {
    cancelRaf();
    if (previewAudio) { previewAudio.pause(); previewAudio = null; }
    if (typeof window !== 'undefined') window.removeEventListener('keydown', handleKeydown);
  });

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

  $: top15 = topGenres(profile?.genres, 15);
  $: top20 = topGenres(profile?.genres, 20);
</script>

<div class="tastes-page">
  {#if loading}
    <div class="loading">loading...</div>

  {:else if viewId && profile}
    <div class="header">
      <h1>{profile.user_id === $user?.id ? 'my taste' : "friend's taste"}</h1>
    </div>

    <GenreCards items={top15} nameFor={gName} colorFor={gColor} />
    <AudioStats audio={profile.audio_features} />
    <ArtistPills artists={profile.top_artists} />
    <TopTracks tracks={profile.top_tracks} onOpen={openTrack} onPreview={previewTrack} {previewId} pausedId={previewPausedId} currentTime={previewCurrentTime} duration={previewDuration} />

  {:else}
    <div class="header">
      <h1>my taste profile</h1>
      {#if hasW() && !edit}
        <div class="header-actions">
          <button class="btn-outline" on:click={() => (edit = true)}>edit</button>
        </div>
      {/if}
    </div>

    {#if !hasW() && !edit}
      <div class="empty-state">
        <div class="empty-card">
          <div class="empty-icon">♪</div>
          <h2>discover your sound</h2>
          <p>
            analyze your listening history to see genre affinity, audio features,
            and top tracks, or set your taste manually.
          </p>

          {#if hasSpotify()}
            <div class="empty-controls">
              <AnalyzeControls bind:time bind:source busy={analyzing} label="analyze" big={true} onRun={analyze} />
            </div>
          {/if}

          <button class="btn-outline" on:click={() => (edit = true)}>or set it manually</button>
        </div>
      </div>

    {:else if edit}
      <GenreEditor genres={genres} {weights} onSet={setW} />

      <div class="save-bar">
        {#if hasSpotify()}
          <AnalyzeControls bind:time bind:source busy={analyzing} label="auto-analyze" onRun={analyze} />
        {/if}

        <button class="btn-primary" on:click={save} disabled={saving || !dirty}>
          {saving ? 'saving...' : 'save my taste'}
        </button>
        {#if !dirty}
          <span class="saved-hint">all saved</span>
        {/if}
      </div>

    {:else}
      <GenreCards items={top20} nameFor={gName} colorFor={gColor} />

      <div class="action-bar">
        {#if hasSpotify()}
          <AnalyzeControls bind:time bind:source busy={analyzing} label="re-analyze" onRun={analyze} />
        {/if}
        <button class="btn-outline" on:click={() => (edit = true)}>edit</button>
      </div>

      <AudioStats audio={profile?.audio_features} />
      <ArtistPills artists={profile?.top_artists} />
      <TopTracks tracks={profile?.top_tracks} onOpen={openTrack} onPreview={previewTrack} {previewId} pausedId={previewPausedId} currentTime={previewCurrentTime} duration={previewDuration} />
    {/if}

    <FriendsTasteGrid tastes={friends} nameFor={gName} colorFor={gColor} />
  {/if}
</div>

<style>
  .tastes-page { width: 100%; }
  .tastes-page h1 { font-size: 1.8rem; margin: 0; }

  .loading { color: #888; text-align: center; padding: 4rem; }
  .header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; margin-bottom: 2rem; flex-wrap: wrap; }
  .header-actions { display: flex; gap: 0.5rem; align-items: center; }

  .btn-primary {
    background: #818cf8;
    color: #000;
    border: none;
    padding: 0.6rem 1.5rem;
    border-radius: 6px;
    font-weight: 600;
    cursor: pointer;
  }
  .btn-primary:hover { background: #a5b4fc; }
  .btn-primary:disabled { opacity: 0.5; cursor: default; }

  .btn-outline {
    background: transparent;
    color: #a0a0a0;
    border: 1px solid #333;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    font-weight: 500;
    cursor: pointer;
    font-size: 0.85rem;
  }
  .btn-outline:hover { border-color: #666; color: #fff; }

  .empty-state { display: flex; justify-content: center; padding: 2rem 0 4rem; }
  .empty-card { text-align: center; max-width: 480px; background: #111; border: 1px solid #222; border-radius: 16px; padding: 3rem 2.5rem; }
  .empty-icon { font-size: 3rem; margin-bottom: 0.5rem; }
  .empty-card h2 { font-size: 1.4rem; color: #e0e0e0; text-transform: none; letter-spacing: normal; margin-bottom: 0.5rem; }
  .empty-card p { color: #888; font-size: 0.95rem; line-height: 1.5; margin-bottom: 1.5rem; }
  .empty-controls { display: flex; justify-content: center; margin-bottom: 0.75rem; }

  .save-bar { display: flex; align-items: center; gap: 1rem; padding: 1rem 0 2rem; flex-wrap: wrap; }
  .action-bar { display: flex; align-items: center; gap: 1rem; padding: 0 0 1.5rem; flex-wrap: wrap; }
  .saved-hint { font-size: 0.8rem; color: #4caf50; }
</style>
