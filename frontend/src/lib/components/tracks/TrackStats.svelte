<script>
  import { bar, hasStats, pct, stats } from '$lib/utils/track';

  export let track;
</script>

{#if hasStats(track)}
  <div class="stats">
    <div class="stats-label">audio stats</div>
    <div class="stats-grid">
      {#each stats as s}
        {#if track[s.key] != null}
          <div class="stat">
            <span class="stat-name">{s.label}</span>
            {#if s.key === 'tempo'}
              <span class="stat-val stat-val-wide">{Math.round(track[s.key])} {s.unit}</span>
            {:else}
              <div class="stat-bar-bg">
                <div class="stat-bar" style="width:{bar(track[s.key])}%"></div>
              </div>
              <span class="stat-val">{pct(track[s.key])}%</span>
            {/if}
          </div>
        {/if}
      {/each}
    </div>
  </div>
{:else}
  <div class="meta-row">
    <span class="hint">no audio stats available for this track</span>
  </div>
{/if}

<style>
  .stats { display: flex; flex-direction: column; gap: 0.45rem; }
  .stats-label { font-size: 0.6rem; text-transform: uppercase; letter-spacing: 0.8px; color: #555; }
  .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 0.45rem 0.7rem; }

  .stat {
    display: grid;
    grid-template-columns: minmax(78px, 90px) minmax(90px, 1fr) minmax(56px, auto);
    align-items: center;
    column-gap: 0.45rem;
    font-size: 0.72rem;
    min-width: 0;
  }

  .stat-name {
    color: #888;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .stat-bar-bg { width: 100%; min-width: 0; height: 5px; background: #222; border-radius: 3px; overflow: hidden; }
  .stat-bar { height: 100%; background: #818cf8; border-radius: 3px; transition: width 0.25s; }
  .stat-val {
    color: #aaa;
    text-align: right;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    min-width: 56px;
  }
  .stat-val-wide { min-width: 82px; }

  .meta-row { display: flex; gap: 0.4rem; align-items: center; flex-wrap: wrap; }
  .hint { font-size: 0.7rem; color: #555; font-style: italic; }

  @media (max-width: 600px) {
    .stats-grid { grid-template-columns: 1fr; }
    .stat { grid-template-columns: 72px minmax(80px, 1fr) 70px; column-gap: 0.35rem; }
    .stat-name, .stat-val { font-size: 0.68rem; }
  }
</style>
