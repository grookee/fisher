<script>
  import TrackCard from './TrackCard.svelte';

  export let tracks = [];
  export let expandedId = null;
  export let activeId = null;
  export let pausedId = null;
  export let currentTime = 0;
  export let duration = 0;
  export let shareById = {};
  export let openLabel = 'Spotify';
  export let path = '';

  export let onToggle = () => {};
  export let onPreview = () => {};
  export let onOpen = () => {};
  export let onShare = () => {};

  const pathFor = (t) => (typeof path === 'function' ? path(t) : path);
  const shareFor = (id) => shareById[id] || 'share';
</script>

<div class="list">
  {#each tracks as t (t.id)}
    <TrackCard
      track={t}
      expanded={expandedId === t.id}
      active={activeId === t.id}
      paused={pausedId === t.id}
      currentTime={(activeId === t.id || pausedId === t.id) ? currentTime : 0}
      duration={(activeId === t.id || pausedId === t.id) ? duration : 0}
      path={pathFor(t)}
      openLabel={openLabel}
      shareLabel={shareFor(t.id)}
      onToggle={() => onToggle(t)}
      onPreview={() => onPreview(t)}
      onOpen={() => onOpen(t)}
      onShare={() => onShare(t)}
    />
  {/each}
</div>

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    animation: fade-up 0.18s ease-out;
  }

  @keyframes fade-up {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>
