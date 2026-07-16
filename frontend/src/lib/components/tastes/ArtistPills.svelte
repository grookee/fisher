<script>
  export let artists = [];
</script>

{#if artists?.length}
  <div class="section">
    <h2>top artists</h2>
    <div class="strip">
      {#each artists as a}
        <span class="pill">{a}</span>
      {/each}
    </div>
  </div>
{/if}

<style>
  .section { margin: 2rem 0; }
  h2 { font-size: 0.85rem; color: #666; text-transform: uppercase; letter-spacing: 0.1em; margin: 0 0 0.75rem; }
  .strip { display: flex; flex-wrap: wrap; gap: 0.4rem; }
  .pill { padding: 0.35rem 0.8rem; background: #151515; border-radius: 20px; font-size: 0.85rem; border: 1px solid #222; }
  .pill:hover { border-color: #444; background: #1a1a1a; }
</style>
