<script>
  export let items = [];
  export let nameFor = (id) => id;
  export let colorFor = () => '#6366f1';
</script>

<div class="grid">
  {#each items as g}
    <div class="card" style="border-color:{colorFor(g.genre_id)}">
      <div class="head" style="color:{colorFor(g.genre_id)}">{nameFor(g.genre_id)}</div>
      <div class="bar-bg">
        <div class="bar-fill" style="width:{g.weight * 100}%;background:{colorFor(g.genre_id)}"></div>
      </div>
      <div class="val">{Math.round(g.weight * 100)}%</div>
    </div>
  {/each}
</div>

<style>
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 0.75rem; margin-bottom: 1.5rem; }
  .card { background: #111; border: 1px solid #333; border-radius: 12px; padding: 1rem; display: flex; flex-direction: column; gap: 0.5rem; transition: all 0.15s; }
  .card:hover { background: #151515; transform: translateY(-1px); }
  .head { font-weight: 700; font-size: 0.9rem; }
  .bar-bg { height: 6px; background: #1a1a1a; border-radius: 3px; overflow: hidden; }
  .bar-fill { height: 100%; border-radius: 3px; transition: width 0.3s; }
  .val { font-size: 0.8rem; color: #888; text-align: right; }
</style>
