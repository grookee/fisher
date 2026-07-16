<script>
  import TrackList from '$lib/components/tracks/TrackList.svelte';
  import TrackSkeleton from '$lib/components/tracks/TrackSkeleton.svelte';

  export let decades = [];
  export let moods = [];
  export let decade = '1980s';
  export let mood = '';
  export let loadingTime = false;
  export let timeTracks = [];
  export let expId = null;
  export let playId = null;
  export let pausedId = null;
  export let audioCurrentTime = 0;
  export let audioDuration = 0;
  export let shareById = {};
  export let pathForTrackGenre = '';
  export let openLabel = 'Spotify';

  export let onLoadTime = () => {};
  export let onToggle = () => {};
  export let onPreview = () => {};
  export let onOpen = () => {};
  export let onShare = () => {};
</script>

<div class="tm-controls">
  <div class="decade-picker">
    {#each decades as d}
      <button class="decade-btn" class:active={decade === d} on:click={() => { decade = d; onLoadTime(true); }}>{d}</button>
    {/each}
  </div>
  <select class="mood-select" bind:value={mood} on:change={() => onLoadTime(true)}>
    <option value="">any mood</option>
    {#each moods.slice(1) as m}
      <option value={m}>{m}</option>
    {/each}
  </select>
</div>

{#if loadingTime}
  <TrackSkeleton count={5} />
{:else if timeTracks.length > 0}
  <div class="section-label" style="margin-top:0">the {decade}</div>
  <TrackList
    tracks={timeTracks}
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
  <p class="no-tracks-message">No tracks found for this decade. Try another decade or mood.</p>
{/if}

<style>
  .section-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 1px; color: #555; margin-bottom: 0.6rem; margin-top: 0.5rem; }
  .no-tracks-message { color: #555; font-size: 0.85rem; text-align: center; padding: 1.5rem; }
  .tm-controls { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; flex-wrap: wrap; }
  .decade-picker { display: flex; gap: 0.25rem; flex-wrap: wrap; }
  .decade-btn { background: #1a1a1a; border: 1px solid #333; color: #aaa; padding: 0.35rem 0.65rem; border-radius: 6px; cursor: pointer; font-size: 0.75rem; font-weight: 600; transition: all 0.12s; }
  .decade-btn:hover { border-color: #666; color: #fff; transform: translateY(-1px); }
  .decade-btn.active { background: #818cf8; color: #000; border-color: #818cf8; }
  .mood-select { background: #1a1a1a; border: 1px solid #333; color: #fff; padding: 0.35rem 0.65rem; border-radius: 6px; font-size: 0.75rem; cursor: pointer; }
</style>
