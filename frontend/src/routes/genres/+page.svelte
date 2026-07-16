<script>
  import { onDestroy, onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import { page } from '$app/stores';
  import { api } from '$lib/api/client';
  import { buildTrackExternalUrl, shareTrack } from '$lib/utils/share';
  import TrackSkeleton from '$lib/components/tracks/TrackSkeleton.svelte';
  import ExploreHeader from '$lib/components/explore/ExploreHeader.svelte';
  import ExploreTabBar from '$lib/components/explore/ExploreTabBar.svelte';
  import GenresTab from '$lib/components/explore/GenresTab.svelte';
  import SurpriseTab from '$lib/components/explore/SurpriseTab.svelte';
  import TimeMachineTab from '$lib/components/explore/TimeMachineTab.svelte';
  import MissedTab from '$lib/components/explore/MissedTab.svelte';
  import RecommendationsTab from '$lib/components/explore/RecommendationsTab.svelte';

  let genres = [];
  let selGenre = null;
  let q = '';

  let tab = 'genres';
  let decade = '1980s';
  let mood = '';

  let tracks = [];
  let feed = [];
  let timeTracks = [];
  let missed = [];
  let recs = [];
  let deepCuts = [];
  let related = [];

  let loading = true;
  let loadingTracks = false;
  let loadingFeed = false;
  let loadingTime = false;
  let loadingMissed = false;
  let loadingRecs = false;
  let loadingRelated = false;

  let playId = null;
  let pausedId = null;
  let expId = null;
  let audio = null;
  let platform = 'spotify';
  let audioCurrentTime = 0;
  let audioDuration = 0;
  let rafId = 0;

  const cancelRaf = () => { if (typeof cancelAnimationFrame !== 'undefined') cancelAnimationFrame(rafId); };
  const scheduleRaf = (fn) => { if (typeof requestAnimationFrame !== 'undefined') return requestAnimationFrame(fn); return 0; };

  let shareById = {};

  let feedLoaded = false;
  let missedLoaded = false;
  let recsLoaded = false;
  let trackReq = 0;
  let relatedReq = 0;

  const qCache = new Map();
  const timeCache = new Map();
  const CACHE_TTL = 60000;

  const decades = ['1950s', '1960s', '1970s', '1980s', '1990s', '2000s', '2010s', '2020s'];
  const moods = ['', 'chill', 'energetic', 'focus', 'party', 'sad', 'happy', 'dark', 'romantic', 'peaceful', 'angry', 'dreamy', 'groovy'];

  onMount(async () => {
    const [gRes, sRes] = await Promise.allSettled([
      api.tastes.genres(),
      api.settings.getAll(),
    ]);

    if (gRes.status === 'fulfilled') genres = gRes.value;
    if (sRes.status === 'fulfilled' && sRes.value?.open_platform) {
      platform = sRes.value.open_platform;
    }

    loading = false;

    if (typeof window !== 'undefined') window.addEventListener('keydown', handleKeydown);

    const focus = $page.url.searchParams.get('focus');
    if (!focus) return;
    const g = genres.find((x) => x.id === focus);
    if (g) pickGenre(g);
  });

  function handleKeydown(e) {
    if (e.code !== 'Space') return;
    const tag = e.target?.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || e.target?.isContentEditable) return;
    const activeId = playId || pausedId;
    if (!activeId) return;
    const allTracks = [...feed, ...tracks, ...timeTracks, ...missed];
    const t = allTracks.find((x) => x.id === activeId);
    if (t) { e.preventDefault(); playTrack(t); }
  }

  onDestroy(() => {
    cancelRaf();
    stopAudio();
    if (typeof window !== 'undefined') window.removeEventListener('keydown', handleKeydown);
  });

  function subs(parentId) {
    return genres.filter((g) => g.parent_id === parentId);
  }

  function parent(g) {
    if (!g?.parent_id) return null;
    return genres.find((x) => x.id === g.parent_id) || null;
  }

  function genrePath(g) {
    const parts = [g.name];
    let p = parent(g);
    while (p) {
      parts.unshift(p.name);
      p = parent(p);
    }
    return parts.join(' → ');
  }

  async function findTracks(query, limit = 30, { fallback = true } = {}) {
    const key = `${query.toLowerCase()}::${limit}::${fallback}`;
    const cached = qCache.get(key);
    if (cached && cached.expiresAt > Date.now()) return cached.data;

    let out = [];

    try {
      const res = await api.explore.genre({ name: query, limit });
      if (Array.isArray(res?.tracks) && (res.tracks.length || !fallback)) {
        out = res.tracks;
      }
    } catch {}

    if (!out.length && fallback) {
      try {
        const alt = await api.playlists.search(query);
        if (Array.isArray(alt) && alt.length) out = alt;
      } catch {}
    }

    qCache.set(key, { data: out, expiresAt: Date.now() + CACHE_TTL });
    return out;
  }

  async function resolvePreviews(trackList) {
    const needs = trackList.filter((t) => !t.preview_url);
    if (!needs.length) return trackList;

    const body = needs.map((t) => ({ id: t.id, title: t.title, artist: t.artist }));
    try {
      const res = await api.previews.resolve(body);
      if (!res?.previews) return trackList;
      const urlMap = {};
      for (const p of res.previews) {
        if (p.preview_url) urlMap[p.id] = p.preview_url;
      }
      return trackList.map((t) => (urlMap[t.id] ? { ...t, preview_url: urlMap[t.id] } : t));
    } catch {
      return trackList;
    }
  }

  async function pickGenre(g) {
    selGenre = g;
    expId = null;
    loadingTracks = true;
    loadingRelated = true;
    related = [];

    const reqId = ++trackReq;
    loadRelated(g.id, ++relatedReq);

    const out = await findTracks(g.name, 30);
    if (reqId !== trackReq) return;

    tracks = await resolvePreviews(out);
    loadingTracks = false;
  }

  async function loadRelated(genreId, reqId) {
    if (!genreId) {
      loadingRelated = false;
      return;
    }
    try {
      const res = await api.explore.related({ id: genreId, limit: 10 });
      if (reqId !== relatedReq) return;
      related = res.related || [];
    } catch {
      if (reqId !== relatedReq) return;
      related = [];
    }
    loadingRelated = false;
  }

  async function loadFeed(force = false) {
    if (!force && feedLoaded && feed.length) return;

    loadingFeed = true;
    feedLoaded = true;
    try {
      const res = await api.explore.feed({ limit: 30 }, force ? { noCache: true } : {});
      feed = await resolvePreviews(res.tracks || []);
    } catch {
      feed = [];
    }
    loadingFeed = false;
  }

  async function loadTime(force = false) {
    const key = `${decade}::${mood || ''}`;
    const cached = timeCache.get(key);
    if (!force && cached && cached.expiresAt > Date.now()) {
      timeTracks = cached.data;
      return;
    }

    loadingTime = true;
    let out = [];

    try {
      const params = { decade };
      if (mood) params.mood = mood;
      const res = await api.explore.timemachine(params);
      out = res.tracks || [];
    } catch {
      out = [];
    }

    if (!out.length) {
      const query = [decade, mood, 'music'].filter(Boolean).join(' ');
      out = await findTracks(query, 30);
    }

    timeTracks = await resolvePreviews(out);
    timeCache.set(key, { data: timeTracks, expiresAt: Date.now() + CACHE_TTL });
    loadingTime = false;
  }

  async function loadMissed(force = false) {
    if (!force && missedLoaded) return;

    loadingMissed = true;
    try {
      const res = await api.explore.missed();
      missed = res.genres || [];
    } catch {
      missed = [];
    }
    missedLoaded = true;
    loadingMissed = false;
  }

  async function loadRecs(force = false) {
    if (!force && recsLoaded) return;

    loadingRecs = true;
    try {
      const res = await api.explore.recommendations({ limit: 12 });
      recs = res.recommendations || [];
      deepCuts = res.deep_cuts || [];
    } catch {
      recs = [];
      deepCuts = [];
    }
    recsLoaded = true;
    loadingRecs = false;
  }

  function setTab(next) {
    if (tab === next) return;

    tab = next;
    selGenre = null;
    expId = null;
    stopAudio();

    if (next === 'surprise') loadFeed(false);
    if (next === 'timemachine') loadTime(false);
    if (next === 'missed') loadMissed(false);
    if (next === 'foryou') loadRecs(true);
  }

  function stopAudio() {
    cancelRaf();
    if (audio) {
      audio.pause();
      audio = null;
    }
    playId = null;
    pausedId = null;
    audioCurrentTime = 0;
    audioDuration = 0;
  }

  function tickAudio() {
    if (audio && !audio.paused) {
      audioCurrentTime = audio.currentTime;
      rafId = scheduleRaf(tickAudio);
    }
  }

  function toggleTrack(t) {
    if (expId === t.id) {
      expId = null;
      return;
    }

    expId = t.id;
    if (t.preview_url) {
      playTrack(t);
    }
  }

  function playTrack(t) {
    if (playId === t.id) {
      audio.pause();
      pausedId = t.id;
      playId = null;
      cancelRaf();
      return;
    }

    if (pausedId === t.id && audio) {
      audio.play();
      playId = t.id;
      pausedId = null;
      rafId = scheduleRaf(tickAudio);
      return;
    }

    stopAudio();
    if (!t.preview_url) return;

    audio = new Audio(t.preview_url);
    audio.play();
    playId = t.id;
    pausedId = null;
    audioCurrentTime = 0;
    audioDuration = 0;
    rafId = scheduleRaf(tickAudio);
    audio.onloadedmetadata = () => {
      audioDuration = audio.duration;
    };
    audio.onended = () => {
      playId = null;
      pausedId = null;
      audioCurrentTime = 0;
      audioDuration = 0;
      cancelRaf();
    };
  }

  function openTrack(t) {
    const url = buildTrackExternalUrl(t, platform);
    if (url) window.open(url, '_blank');
  }

  function markShare(id, text) {
    shareById = { ...shareById, [id]: text };
    setTimeout(() => {
      if (shareById[id] !== text) return;
      const next = { ...shareById };
      delete next[id];
      shareById = next;
    }, 1800);
  }

  async function shareTrackNow(t) {
    try {
      const res = await shareTrack(t, platform);
      if (res.ok && res.method === 'native') {
        markShare(t.id, 'shared');
        return;
      }
      if (res.ok && res.method === 'clipboard') {
        markShare(t.id, 'copied');
        return;
      }
      if (!res.cancelled) {
        markShare(t.id, 'unavailable');
      }
    } catch {
      markShare(t.id, 'failed');
    }
  }

  function platformName(p) {
    return {
      spotify: 'Spotify',
      apple_music: 'Apple Music',
      song_link: 'song.link',
    }[p] || p;
  }

  function bg(g) {
    return `background: radial-gradient(circle at ${50 + (g.x || 0) * 10}% ${50 + (g.y || 0) * 10}%, ${g.color}22 0%, transparent 70%)`;
  }

  function parentClr(g) {
    if (!g.parent_id) return g.color;
    const p = parent(g);
    return p ? p.color : '#6366f1';
  }

  function parentLbl(g) {
    if (!g.parent_id) return '';
    const p = parent(g);
    return p ? p.name : '';
  }

  function byGenreName(name) {
    if (!name) return null;
    const needle = name.trim().toLowerCase();
    return genres.find((g) => g.name.toLowerCase() === needle) || null;
  }

  function pathForTrackGenre(t) {
    const name = t?.genre || t?.genre_name || '';
    if (!name) return '';
    const g = byGenreName(name);
    return g ? genrePath(g) : name;
  }

  function pickMissedGenre(g) {
    setTab('genres');
    pickGenre(g);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function pickRecGenre(g) {
    setTab('genres');
    pickGenre(g);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function onRecSettingsChanged() {
    recsLoaded = false;
    loadRecs(true);
  }

  $: filtered = q
    ? genres.filter((g) => g.name.toLowerCase().includes(q.toLowerCase()))
    : genres;

  $: parents = !q ? genres.filter((g) => !g.parent_id) : [];
  $: openLabel = platformName(platform);
  $: selPath = selGenre ? genrePath(selGenre) : '';
</script>

<div class="explore-page">
  <ExploreHeader bind:query={q} />
  <ExploreTabBar {tab} onSelect={setTab} />

  {#if loading}
    <div class="skeleton-section">
      <div class="sk-title"></div>
      <div class="sk-pills">
        {#each Array(8) as _}
          <div class="sk-pill"></div>
        {/each}
      </div>
      <div class="sk-title" style="width:50%"></div>
      <TrackSkeleton count={5} />
    </div>
  {:else}
    {#key tab}
      <section class="tab-panel" in:fade={{ duration: 160 }} out:fade={{ duration: 120 }}>
        {#if tab === 'genres'}
          <GenresTab
            {selGenre}
            {loadingTracks}
            {tracks}
            {expId}
            {playId}
            {pausedId}
            {audioCurrentTime}
            {audioDuration}
            {shareById}
            {selPath}
            {openLabel}
            {q}
            {filtered}
            {parents}
            {related}
            {loadingRelated}
            {parent}
            {subs}
            {bg}
            {pickGenre}
            onBack={() => { selGenre = null; }}
            onToggle={toggleTrack}
            onPreview={playTrack}
            onOpen={openTrack}
            onShare={shareTrackNow}
            onPickRelated={pickGenre}
          />

        {:else if tab === 'surprise'}
          <SurpriseTab
            {loadingFeed}
            {feed}
            {expId}
            {playId}
            {pausedId}
            {audioCurrentTime}
            {audioDuration}
            {shareById}
            {pathForTrackGenre}
            {openLabel}
            onRefresh={loadFeed}
            onToggle={toggleTrack}
            onPreview={playTrack}
            onOpen={openTrack}
            onShare={shareTrackNow}
          />

        {:else if tab === 'timemachine'}
          <TimeMachineTab
            {decades}
            {moods}
            bind:decade
            bind:mood
            {loadingTime}
            {timeTracks}
            {expId}
            {playId}
            {pausedId}
            {audioCurrentTime}
            {audioDuration}
            {shareById}
            {pathForTrackGenre}
            {openLabel}
            onLoadTime={loadTime}
            onToggle={toggleTrack}
            onPreview={playTrack}
            onOpen={openTrack}
            onShare={shareTrackNow}
          />

        {:else if tab === 'missed'}
          <MissedTab
            {loadingMissed}
            {missed}
            {bg}
            {parentClr}
            {parentLbl}
            onPickMissed={pickMissedGenre}
          />

        {:else if tab === 'foryou'}
          <RecommendationsTab
            {loadingRecs}
            {recs}
            {deepCuts}
            allGenres={genres}
            {bg}
            onPickRec={pickRecGenre}
            onSettingsChanged={onRecSettingsChanged}
          />
        {/if}
      </section>
    {/key}
  {/if}
</div>

<style>
  @keyframes sk-shimmer {
    0% { background-position: -200px 0; }
    100% { background-position: calc(200px + 100%) 0; }
  }

  .explore-page {
    width: 100%;
    max-width: 1120px;
    margin: 0 auto;
    padding: 0 0.75rem 2.25rem;
  }

  .tab-panel { min-height: 280px; }
  .sk-title { height: 14px; width: 30%; background: linear-gradient(90deg, #1a1a1a 0%, #2a2a2a 50%, #1a1a1a 100%); background-size: 200px 100%; border-radius: 4px; margin-bottom: 0.8rem; animation: sk-shimmer 1.5s ease-in-out infinite; }
  .sk-pills { display: flex; gap: 0.5rem; flex-wrap: wrap; margin-bottom: 1rem; }
  .sk-pill { height: 32px; width: 90px; background: linear-gradient(90deg, #1a1a1a 0%, #2a2a2a 50%, #1a1a1a 100%); background-size: 200px 100%; border-radius: 16px; animation: sk-shimmer 1.5s ease-in-out infinite; }

  @media (max-width: 940px) {
    .explore-page { padding: 0 0.35rem 1.6rem; }
  }

  @media (max-width: 600px) {
    .explore-page { padding: 0 0.1rem 1.2rem; }
  }
</style>
