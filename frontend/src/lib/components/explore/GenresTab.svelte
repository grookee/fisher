<script>
  import TrackList from '$lib/components/tracks/TrackList.svelte';
  import TrackSkeleton from '$lib/components/tracks/TrackSkeleton.svelte';

  export let selGenre = null;
  export let loadingTracks = false;
  export let tracks = [];
  export let expId = null;
  export let playId = null;
  export let pausedId = null;
  export let audioCurrentTime = 0;
  export let audioDuration = 0;
  export let shareById = {};
  export let selPath = '';
  export let openLabel = 'Spotify';
  export let q = '';
  export let filtered = [];
  export let parents = [];
  export let related = [];
  export let loadingRelated = false;

  export let parent = () => null;
  export let subs = () => [];
  export let bg = () => '';
  export let pickGenre = () => {};
  export let onBack = () => {};
  export let onToggle = () => {};
  export let onPreview = () => {};
  export let onOpen = () => {};
  export let onShare = () => {};
  export let onPickRelated = () => {};

  const relationLabel = {
    subgenre_of: 'subgenre',
    influenced_by: 'influence',
    cooccurs_with: 'similar',
  };
</script>

{#if selGenre}
  <div class="section-label" style="margin-top:0">
    tracks — {selGenre.name}
    <button class="btn-back" on:click={onBack}>← back</button>
  </div>

  {#if loadingTracks}
    <TrackSkeleton count={5} />
  {:else if tracks.length > 0}
    <TrackList
      {tracks}
      expandedId={expId}
      activeId={playId}
      {pausedId}
      currentTime={audioCurrentTime}
      duration={audioDuration}
      {shareById}
      path={selPath}
      {openLabel}
      {onToggle}
      {onPreview}
      {onOpen}
      {onShare}
    />
  {:else}
    <p class="no-tracks-message">No tracks found for "{selGenre.name}".</p>
  {/if}

  {#if loadingRelated}
    <div class="section-label">related genres</div>
    <div class="sk-pills">
      {#each Array(5) as _}
        <div class="sk-pill"></div>
      {/each}
    </div>
  {:else if related.length > 0}
    <div class="section-label">related genres</div>
    <div class="related-pills">
      {#each related as r}
        <button
          class="pill related-pill"
          style="border-color: {r.color}; background: {r.color}14;"
          on:click={() => onPickRelated(r)}
          title="{relationLabel[r.relation_type] || 'related'} · match {Math.round((r.weight || 0) * 100)}%"
        >
          <span style="color: {r.color}">{r.name}</span>
          <span class="related-type">{relationLabel[r.relation_type] || 'related'}</span>
        </button>
      {/each}
    </div>
  {/if}
{/if}

<div class="section-label" class:section-label-top={selGenre}>genres</div>

{#if q}
  <div class="genre-pills">
    {#each filtered as g}
      <button
        class="pill"
        class:active={selGenre?.id === g.id}
        style="border-color: {g.color}; {bg(g)}"
        on:click={() => pickGenre(g)}
      >
        {#if g.parent_id && parent(g)}
          <span class="pill-parent" style="color: {parent(g).color}">{parent(g).name}</span>
        {/if}
        <span style="color: {g.color}">{g.name}</span>
      </button>
    {/each}
  </div>
{:else}
  {#each parents as p}
    <div class="pill-group">
      <div class="pill-parent-label" style="color: {p.color}">{p.name}</div>
      <div class="pills-row">
        <button
          class="pill parent-pill"
          class:active={selGenre?.id === p.id}
          style="border-color: {p.color}; background: {p.color}18;"
          on:click={() => pickGenre(p)}
        >{p.name}</button>

        {#each subs(p.id) as sub}
          <button
            class="pill"
            class:active={selGenre?.id === sub.id}
            style="border-color: {p.color}66; border-left: 2px solid {p.color};"
            on:click={() => pickGenre(sub)}
          >{sub.name}</button>
        {/each}
      </div>
    </div>
  {/each}
{/if}

<style>
  @keyframes fade-up {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  @keyframes sk-shimmer {
    0% { background-position: -200px 0; }
    100% { background-position: calc(200px + 100%) 0; }
  }

  .section-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 1px; color: #555; margin-bottom: 0.6rem; margin-top: 0.5rem; }
  .sk-pills { display: flex; gap: 0.4rem; flex-wrap: wrap; margin-bottom: 1rem; }
  .sk-pill { height: 30px; width: 110px; background: linear-gradient(90deg, #1a1a1a 0%, #2a2a2a 50%, #1a1a1a 100%); background-size: 200px 100%; border-radius: 16px; animation: sk-shimmer 1.5s ease-in-out infinite; }
  .related-pills { display: flex; flex-wrap: wrap; gap: 0.35rem; margin-bottom: 1.2rem; animation: fade-up 0.16s ease-out; }
  .related-pill { display: inline-flex; align-items: center; gap: 0.4rem; }
  .related-type { font-size: 0.6rem; color: #666; text-transform: uppercase; letter-spacing: 0.4px; }
  .section-label-top { margin-top: 0; }
  .btn-back { background: none; border: 1px solid #444; color: #aaa; padding: 0.2rem 0.6rem; border-radius: 4px; cursor: pointer; font-size: 0.7rem; margin-left: 0.5rem; vertical-align: middle; }
  .btn-back:hover { border-color: #818cf8; color: #fff; }

  .pill-group { margin-bottom: 0.5rem; }
  .pill-parent-label { font-size: 0.6rem; text-transform: uppercase; letter-spacing: 0.8px; margin-bottom: 0.2rem; font-weight: 600; }
  .pills-row { display: flex; flex-wrap: wrap; gap: 0.3rem; }
  .pill { background: #111; border: 1px solid #333; border-radius: 16px; padding: 0.35rem 0.75rem; cursor: pointer; font-size: 0.8rem; color: #ccc; transition: all 0.12s; white-space: nowrap; display: inline-flex; align-items: center; gap: 0.3rem; }
  .pill:hover { background: #1a1a1a; transform: translateY(-1px); }
  .pill.active { border-width: 2px; background: #ffffff08; }
  .parent-pill { font-weight: 700; font-size: 0.85rem; padding: 0.4rem 0.9rem; }
  .pill-parent { font-size: 0.6rem; color: #555; }
  .genre-pills { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-bottom: 1rem; animation: fade-up 0.16s ease-out; }
  .no-tracks-message { color: #555; font-size: 0.85rem; text-align: center; padding: 1.5rem; }
</style>
