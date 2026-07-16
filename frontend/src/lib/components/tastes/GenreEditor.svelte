<script>
  export let genres = [];
  export let weights = {};
  export let onSet = () => {};
</script>

<div class="grid edit-grid">
  {#each genres as g}
    <div class="card" style="border-color:{g.color}">
      <div class="head" style="color:{g.color}">
        {g.name}
        {#if g.description}
          <span class="desc">{g.description}</span>
        {/if}
      </div>
      <div class="bar-bg">
        <div class="bar-fill" style="width:{weights[g.id] ?? 0}%;background:{g.color}"></div>
      </div>
      <input
        type="range"
        min="0"
        max="100"
        value={weights[g.id] ?? 0}
        on:input={(e) => onSet(g.id, parseInt(e.currentTarget.value, 10))}
        class="slider"
        style="accent-color:{g.color}"
      />
      <div class="val">{weights[g.id] || 0}%</div>
    </div>
  {/each}
</div>

<style>
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 0.75rem; margin-bottom: 1.5rem; }
  .edit-grid { grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); }
  .card { background: #111; border: 1px solid #333; border-radius: 12px; padding: 1rem; display: flex; flex-direction: column; gap: 0.5rem; transition: all 0.15s; }
  .card:hover { background: #151515; transform: translateY(-1px); }
  .head { font-weight: 700; font-size: 0.9rem; display: flex; flex-direction: column; gap: 0.15rem; }
  .desc { font-size: 0.7rem; font-weight: 400; color: #666; }
  .bar-bg { height: 6px; background: #1a1a1a; border-radius: 3px; overflow: hidden; }
  .bar-fill { height: 100%; border-radius: 3px; transition: width 0.3s; }
  .slider { width: 100%; height: 4px; cursor: pointer; margin: 0; }
  .val { font-size: 0.8rem; color: #888; text-align: right; }
</style>
