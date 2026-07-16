<script>
  import TrackCard from '$lib/components/tracks/TrackCard.svelte';
  import { normalizeTrack } from '$lib/utils/track';
  import { buildTrackExternalUrl, shareTrack } from '$lib/utils/share';

  export let tracks = [];
  export let onOpen = () => {};
  export let onPreview = () => {};
  export let previewId = null;
  export let pausedId = null;
  export let currentTime = 0;
  export let duration = 0;

  let expandedIds = new Set();
  let normalizedTracks = [];

  $: normalizedTracks = (tracks || []).map((t, i) => {
    const nt = normalizeTrack(t);
    const fallbackId = nt?.id || nt?.spotify_uri || `${nt?.title || 'track'}::${nt?.artist || ''}::${i}`;
    return { ...nt, id: fallbackId };
  });

  function toggleExpand(id) {
    if (expandedIds.has(id)) expandedIds.delete(id);
    else expandedIds.add(id);
    expandedIds = expandedIds;
  }
</script>

{#if normalizedTracks?.length}
  <div class="section">
    <h2>top tracks</h2>
    <div class="list">
      {#each normalizedTracks as t (t.id)}
        {@const active = previewId === t.id}
        {@const paused = pausedId === t.id}
        <TrackCard
          track={t}
          expanded={expandedIds.has(t.id)}
          {active}
          {paused}
          {currentTime}
          {duration}
          openLabel="Spotify"
          onToggle={() => toggleExpand(t.id)}
          onPreview={() => onPreview(t)}
          onOpen={() => onOpen(t)}
          onShare={() => shareTrack(t)}
        />
      {/each}
    </div>
  </div>
{/if}

<style>
  .section { margin: 2rem 0; }
  h2 { font-size: 0.85rem; color: #666; text-transform: uppercase; letter-spacing: 0.1em; margin: 0 0 0.75rem; }
  .list { display: flex; flex-direction: column; gap: 0.4rem; }
</style>
