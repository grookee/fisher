<script>
  import TrackList from '$lib/components/tracks/TrackList.svelte';
  import TrackSkeleton from '$lib/components/tracks/TrackSkeleton.svelte';

  export let loadingFeed = false;
  export let feed = [];
  export let expId = null;
  export let playId = null;
  export let pausedId = null;
  export let audioCurrentTime = 0;
  export let audioDuration = 0;
  export let shareById = {};
  export let pathForTrackGenre = '';
  export let openLabel = 'Spotify';

  export let onRefresh = () => {};
  export let onToggle = () => {};
  export let onPreview = () => {};
  export let onOpen = () => {};
  export let onShare = () => {};
</script>

<div class="surprise-header">
  <div class="section-label" style="margin-top:0">random picks from across all genres</div>
  <button class="btn-surprise-refresh" on:click={() => onRefresh(true)} disabled={loadingFeed}>
    ↻ {loadingFeed ? 'loading...' : 'refresh'}
  </button>
</div>

{#if loadingFeed}
  <TrackSkeleton count={5} />
{:else if feed.length > 0}
  <TrackList
    tracks={feed}
    expandedId={expId}
    activeId={playId}
    {pausedId}
    currentTime={audioCurrentTime}
    duration={audioDuration}
    {shareById}
    path={pathForTrackGenre}
    {openLabel}
    onToggle={onToggle}
    onPreview={onPreview}
    onOpen={onOpen}
    onShare={onShare}
  />
{:else}
  <div class="empty-state">
    <p>Discover music from across all genres.</p>
    <button class="btn-primary" style="margin-top:0.75rem" on:click={() => onRefresh(true)}>↻ surprise me</button>
  </div>
{/if}

<style>
  .section-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 1px; color: #555; margin-bottom: 0.6rem; margin-top: 0.5rem; }
  .surprise-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.75rem; }
  .btn-surprise-refresh { background: #10b981; color: #000; border: none; padding: 0.4rem 0.9rem; border-radius: 6px; font-weight: 600; cursor: pointer; font-size: 0.8rem; white-space: nowrap; transition: transform 0.12s, background 0.12s; }
  .btn-surprise-refresh:hover { background: #34d399; transform: translateY(-1px); }
  .btn-surprise-refresh:disabled { opacity: 0.6; cursor: not-allowed; }

  .empty-state { text-align: center; padding: 2rem; color: #888; }
  .btn-primary { background: #818cf8; color: #000; border: none; padding: 0.5rem 1.2rem; border-radius: 6px; cursor: pointer; font-weight: 600; font-size: 0.9rem; transition: transform 0.12s, background 0.12s; }
  .btn-primary:hover { background: #a5b4fc; transform: translateY(-1px); }
</style>
