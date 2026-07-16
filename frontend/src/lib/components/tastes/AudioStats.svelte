<script>
  import { afColor, afLabel } from '$lib/utils/taste';

  export let audio = null;
</script>

{#if audio}
  <div class="af-section">
    <h2>audio features</h2>
    <div class="af-grid">
      {#each Object.entries(audio) as [key, val]}
        {#if key !== 'tempo'}
          <div class="af-item">
            <div class="af-label" style="color:{afColor(key)}">{afLabel(key)}</div>
            <div class="af-bar-track">
              <div class="af-bar-fill" style="width:{val * 100}%;background:{afColor(key)}"></div>
            </div>
            <div class="af-val">{Math.round(val * 100)}%</div>
          </div>
        {/if}
      {/each}

      {#if audio.tempo}
        <div class="af-item">
          <div class="af-label" style="color:#6366f1">tempo</div>
          <div class="af-val tempo-val">
            {Math.round(audio.tempo)} <span class="af-unit">bpm</span>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .af-section { margin-bottom: 2rem; }
  .af-section h2 { font-size: 0.85rem; color: #666; text-transform: uppercase; letter-spacing: 0.1em; margin: 0 0 0.75rem; }
  .af-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 0.6rem; }
  .af-item { display: flex; align-items: center; gap: 0.6rem; }
  .af-label { font-size: 0.8rem; font-weight: 600; width: 110px; flex-shrink: 0; text-transform: capitalize; }
  .af-bar-track { flex: 1; height: 6px; background: #1a1a1a; border-radius: 3px; overflow: hidden; }
  .af-bar-fill { height: 100%; border-radius: 3px; transition: width 0.3s; }
  .af-val { width: 40px; text-align: right; font-size: 0.8rem; color: #888; flex-shrink: 0; }
  .tempo-val { width: 100%; text-align: center; font-size: 1.2rem; font-weight: 700; }
  .af-unit { font-size: 0.75rem; color: #666; font-weight: 400; }
</style>
