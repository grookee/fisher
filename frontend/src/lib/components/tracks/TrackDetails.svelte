<script>
  import { slide } from 'svelte/transition';
  import TrackStats from './TrackStats.svelte';
  import { fmtMs } from '$lib/utils/track';

  export let track;
  export let expanded = false;
  export let active = false;
  export let paused = false;
  export let path = '';
  export let openLabel = 'Spotify';
  export let shareLabel = 'share';

  export let onPreview = () => {};
  export let onOpen = () => {};
  export let onShare = () => {};

  $: hasPreview = !!track.preview_url;
  $: metaPath = path || track.genre_path || track.genre;
</script>

{#if expanded}
  <div class="details" transition:slide={{ duration: 150 }}>
    <div class="meta-row">
      {#if track.release_year}
        <span class="badge">year: {track.release_year}</span>
      {/if}
      {#if track.duration_ms}
        <span class="badge">{fmtMs(track.duration_ms)}</span>
      {/if}
      {#if hasPreview}
        <span class="badge badge-preview">preview available</span>
      {/if}
    </div>

    {#if metaPath}
      <div class="meta-row">
        <span class="path">{metaPath}</span>
      </div>
    {/if}

    <TrackStats {track} />

    <div class="details-actions">
      {#if hasPreview}
        <button class="btn-preview" on:click|stopPropagation={onPreview}>
          {active ? '■ stop preview' : paused ? '▶ resume preview' : '▶ play preview'}
        </button>
      {/if}
      <button class="btn-open" on:click|stopPropagation={onOpen}>
        open in {openLabel} ↗
      </button>
      <button class="btn-share" on:click|stopPropagation={onShare}>
        {shareLabel}
      </button>
    </div>
  </div>
{/if}

<style>
  .details {
    width: 100%;
    padding: 0.85rem 0.25rem 0.45rem;
    border-top: 1px solid #222;
    margin-top: 0.45rem;
    display: flex;
    flex-direction: column;
    gap: 0.55rem;
  }

  .meta-row { display: flex; gap: 0.4rem; flex-wrap: wrap; align-items: center; }
  .badge { font-size: 0.7rem; color: #aaa; background: #1a1a1a; padding: 0.15rem 0.5rem; border-radius: 4px; border: 1px solid #2a2a2a; }
  .badge-preview { color: #818cf8; border-color: #818cf833; }
  .path { font-size: 0.75rem; color: #666; font-style: italic; }

  .details-actions { display: flex; gap: 0.4rem; flex-wrap: wrap; }
  .btn-preview {
    background: #818cf8;
    color: #000;
    border: none;
    padding: 0.3rem 0.7rem;
    border-radius: 4px;
    font-size: 0.7rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.12s;
  }
  .btn-preview:hover { background: #6366f1; }
  .btn-open {
    background: transparent;
    color: #a0a0a0;
    border: 1px solid #333;
    padding: 0.3rem 0.7rem;
    border-radius: 4px;
    font-size: 0.7rem;
    font-weight: 500;
    cursor: pointer;
  }
  .btn-open:hover { border-color: #666; color: #fff; }

  .btn-share {
    background: transparent;
    color: #c4b5fd;
    border: 1px solid #4c1d95;
    padding: 0.3rem 0.7rem;
    border-radius: 4px;
    font-size: 0.7rem;
    font-weight: 600;
    cursor: pointer;
    min-width: 70px;
    text-transform: lowercase;
  }
  .btn-share:hover { border-color: #7c3aed; color: #ddd6fe; }

  @media (max-width: 940px) {
    .details { padding: 0.75rem 0.1rem 0.35rem; }
  }

  @media (max-width: 600px) {
    .details { padding: 0.7rem 0 0.3rem; }
  }
</style>
