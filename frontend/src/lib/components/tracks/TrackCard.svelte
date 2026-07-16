<script>
  import { slide } from 'svelte/transition';
  import TrackDetails from './TrackDetails.svelte';
  import { fmtMs, normalizeTrack } from '$lib/utils/track';

  export let track;
  export let expanded = false;
  export let active = false;
  export let paused = false;
  export let currentTime = 0;
  export let duration = 0;
  export let path = '';
  export let openLabel = 'Spotify';
  export let shareLabel = 'share';

  export let onToggle = () => {};
  export let onPreview = () => {};
  export let onOpen = () => {};
  export let onShare = () => {};

  $: normalizedTrack = normalizeTrack(track);
  $: progress = duration > 0 ? (currentTime / duration) * 100 : 0;
  $: hasPreview = !!normalizedTrack?.preview_url;
  $: showPlayer = hasPreview && (active || paused);
</script>

<div class="track" class:expanded class:active class:paused={paused && !active}>
  {#if normalizedTrack?.album_art_url}
    <img src={normalizedTrack.album_art_url} alt={normalizedTrack.album} class="art" loading="lazy" />
  {:else}
    <div class="art-ph"></div>
  {/if}

  <div class="info" on:click={onToggle}>
    <span class="title">{normalizedTrack?.title}</span>
    <span class="artist">{normalizedTrack?.artist}</span>
  </div>

  <div class="actions">
    {#if hasPreview}
      <button class="btn-ico btn-play" class:playing={active} on:click|stopPropagation={onPreview}>
        {active ? '■' : '▶'}
      </button>
    {/if}
    <button class="btn-ico btn-expand" on:click|stopPropagation={onToggle}>
      {expanded ? '▾' : '▸'}
    </button>
  </div>

  {#if showPlayer}
    <div class="preview-player" class:preview-paused={paused && !active}>
      <div class="preview-track" on:click|stopPropagation={onPreview}>
        <div class="progress-bar">
          <div class="progress-fill" style="width: {progress}%"></div>
        </div>
      </div>
      <span class="preview-time">{fmtMs(currentTime * 1000)}</span>
    </div>
  {/if}

  <TrackDetails
    track={normalizedTrack}
    {expanded}
    {active}
    {paused}
    {path}
    {openLabel}
    {shareLabel}
    {onPreview}
    {onOpen}
    {onShare}
  />
</div>

<style>
  .track {
    display: flex;
    align-items: center;
    gap: 0.65rem;
    padding: 0.65rem 0.85rem;
    background: #111;
    border-radius: 10px;
    border: 1px solid #222;
    transition: background 0.12s, border-color 0.12s, transform 0.12s;
    flex-wrap: wrap;
  }

  .track:hover { background: #181818; transform: translateY(-1px); }
  .track.active { border-color: #818cf8; background: #13131f; }
  .track.paused { border-color: #818cf844; background: #12121a; }
  .track.expanded { border-color: #333; }

  .art { width: 40px; height: 40px; border-radius: 4px; object-fit: cover; flex-shrink: 0; }
  .art-ph { width: 40px; height: 40px; border-radius: 4px; background: #1a1a1a; flex-shrink: 0; }

  .info { flex: 1; min-width: 0; cursor: pointer; }
  .title { display: block; font-size: 0.85rem; font-weight: 500; }
  .artist { display: block; font-size: 0.75rem; color: #888; margin-top: 0.1rem; }

  .actions { display: flex; align-items: center; gap: 0.3rem; flex-shrink: 0; }
  .btn-ico {
    background: none;
    border: 1px solid #444;
    color: #fff;
    width: 30px;
    height: 30px;
    border-radius: 50%;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    flex-shrink: 0;
    transition: border-color 0.15s, background 0.15s;
  }
  .btn-ico:hover { border-color: #818cf8; }
  .btn-play.playing { border-color: #818cf8; background: #818cf820; color: #818cf8; }
  .btn-expand { color: #888; font-size: 0.8rem; }
  .btn-expand:hover { color: #fff; }

  .preview-player {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.35rem 0 0.15rem;
  }
  .preview-paused .preview-time { color: #818cf8; }
  .preview-track {
    flex: 1;
    cursor: pointer;
  }
  .progress-bar {
    width: 100%;
    height: 3px;
    background: #2a2a2a;
    border-radius: 2px;
    overflow: hidden;
  }
  .progress-fill {
    height: 100%;
    background: #818cf8;
    border-radius: 2px;
  }
  .preview-time {
    font-size: 0.65rem;
    color: #666;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    min-width: 30px;
  }

  @media (max-width: 940px) {
    .track { padding: 0.6rem 0.7rem; }
  }

  @media (max-width: 600px) {
    .track { border-radius: 8px; padding: 0.55rem 0.6rem; }
  }
</style>
