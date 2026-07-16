<script>
  export let time = 'medium_term';
  export let source = 'spotify';
  export let busy = false;
  export let label = 'analyze';
  export let big = false;
  export let onRun = () => {};
</script>

<div class="controls">
  <select class="sel" bind:value={time}>
    <option value="short_term">past month</option>
    <option value="medium_term">past 6 months</option>
    <option value="long_term">all time</option>
  </select>
  <select class="sel" bind:value={source}>
    <option value="spotify">spotify</option>
    <option value="combined">+ last.fm</option>
    <option value="lastfm">last.fm</option>
  </select>
  <button class="btn" class:big on:click={onRun} disabled={busy}>
    {busy ? 'analyzing...' : label}
  </button>
</div>

<style>
  .controls { display: flex; gap: 0.4rem; align-items: center; flex-wrap: wrap; }
  .sel {
    background: #151515;
    color: #e0e0e0;
    border: 1px solid #333;
    border-radius: 6px;
    padding: 0.4rem 0.6rem;
    font-size: 0.8rem;
    cursor: pointer;
  }
  .sel:focus { outline: none; border-color: #818cf8; }

  .btn {
    background: #1db954;
    color: #000;
    border: none;
    padding: 0.6rem 1.5rem;
    border-radius: 6px;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
    transition: transform 0.12s, background 0.12s;
  }
  .btn:hover { background: #1ed760; transform: translateY(-1px); }
  .btn:disabled { opacity: 0.5; cursor: default; }
  .btn.big { padding: 0.8rem 2rem; font-size: 1rem; }
</style>
