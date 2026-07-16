<script>
  export let loadingMissed = false;
  export let missed = [];

  export let bg = () => '';
  export let parentClr = () => '#6366f1';
  export let parentLbl = () => '';
  export let onPickMissed = () => {};
</script>

{#if loadingMissed}
  <div class="sk-pills">
    {#each Array(6) as _}
      <div class="sk-pill" style="width: 180px; height: 90px;"></div>
    {/each}
  </div>
{:else if missed.length > 0}
  <div class="section-label" style="margin-top:0">genres related to your taste you haven't explored yet</div>
  <div class="missed-grid">
    {#each missed as g}
      <button
        class="missed-card"
        style="border-color: {g.color}; {bg(g)}"
        on:click={() => onPickMissed(g)}
      >
        <span class="missed-badge" style="background: {parentClr(g)}">
          {parentLbl(g) || 'genre'}
        </span>
        <span class="missed-name" style="color: {g.color}">{g.name}</span>
        <span class="missed-desc">{g.description || g.reason}</span>
      </button>
    {/each}
  </div>
{:else}
  <div class="empty-state">
    <p>No missed gems yet.</p>
    <p class="hint">Analyze your taste on the <a href="/tastes">my taste</a> page first.</p>
  </div>
{/if}

<style>
  @keyframes sk-shimmer {
    0% { background-position: -200px 0; }
    100% { background-position: calc(200px + 100%) 0; }
  }

  @keyframes fade-up {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .section-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 1px; color: #555; margin-bottom: 0.6rem; margin-top: 0.5rem; }
  .sk-pills { display: flex; gap: 0.5rem; flex-wrap: wrap; margin-bottom: 1rem; }
  .sk-pill { height: 32px; width: 90px; background: linear-gradient(90deg, #1a1a1a 0%, #2a2a2a 50%, #1a1a1a 100%); background-size: 200px 100%; border-radius: 16px; animation: sk-shimmer 1.5s ease-in-out infinite; }
  .missed-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 0.6rem; animation: fade-up 0.16s ease-out; }
  .missed-card { background: #111; border: 1px solid #333; border-radius: 8px; padding: 0.8rem; cursor: pointer; transition: all 0.12s; display: flex; flex-direction: column; align-items: center; gap: 0.2rem; text-align: center; position: relative; }
  .missed-card:hover { transform: translateY(-2px); box-shadow: 0 4px 16px rgba(0,0,0,0.4); }
  .missed-badge { position: absolute; top: -1px; left: -1px; font-size: 0.55rem; color: #000; padding: 0.1rem 0.4rem; border-radius: 8px 0 8px 0; font-weight: 600; text-transform: uppercase; }
  .missed-name { font-weight: 700; font-size: 0.85rem; }
  .missed-desc { font-size: 0.7rem; color: #666; }
  .empty-state { text-align: center; padding: 2rem; color: #888; }
  .empty-state .hint { font-size: 0.8rem; margin-top: 0.3rem; }
  .empty-state .hint a { color: #818cf8; }
</style>
