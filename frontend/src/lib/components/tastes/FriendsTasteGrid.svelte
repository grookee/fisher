<script>
  export let tastes = [];
  export let nameFor = (id) => id;
  export let colorFor = () => '#6366f1';
</script>

{#if tastes?.length}
  <div class="section">
    <h2>friends who shared their taste</h2>
    <div class="grid">
      {#each tastes as ft}
        <div class="card">
          <div class="genres">
            {#each ft.genres.slice(0, 5) as g}
              <span class="mini" style="background:{colorFor(g.genre_id)}22;color:{colorFor(g.genre_id)}">
                {nameFor(g.genre_id)}
              </span>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .section { margin: 2rem 0; }
  h2 { font-size: 0.85rem; color: #666; text-transform: uppercase; letter-spacing: 0.1em; margin: 0 0 0.75rem; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 0.75rem; }
  .card { padding: 1rem; background: #111; border-radius: 10px; border: 1px solid #222; }
  .genres { display: flex; flex-wrap: wrap; gap: 0.3rem; }
  .mini { padding: 0.2rem 0.6rem; border-radius: 4px; font-size: 0.75rem; }
</style>
