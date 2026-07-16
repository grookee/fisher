<script>
  export let count = 5;
</script>

<div class="sk-tracks">
  {#each Array(count) as _}
    <div class="sk-track">
      <div class="sk-track-art"></div>
      <div class="sk-track-lines">
        <div class="sk-line w-70"></div>
        <div class="sk-line w-40"></div>
      </div>
    </div>
  {/each}
</div>

<style>
  @keyframes sk-shimmer {
    0% { background-position: -200px 0; }
    100% { background-position: calc(200px + 100%) 0; }
  }

  .sk-tracks { display: flex; flex-direction: column; gap: 0.4rem; }
  .sk-track { display: flex; align-items: center; gap: 0.6rem; padding: 0.5rem 0.7rem; background: #111; border-radius: 8px; }
  .sk-track-art {
    width: 40px;
    height: 40px;
    border-radius: 4px;
    background: linear-gradient(90deg, #1a1a1a 0%, #2a2a2a 50%, #1a1a1a 100%);
    background-size: 200px 100%;
    flex-shrink: 0;
    animation: sk-shimmer 1.5s ease-in-out infinite;
  }

  .sk-track-lines { flex: 1; display: flex; flex-direction: column; gap: 0.4rem; }
  .sk-line {
    height: 12px;
    background: linear-gradient(90deg, #1a1a1a 0%, #2a2a2a 50%, #1a1a1a 100%);
    background-size: 200px 100%;
    border-radius: 4px;
    animation: sk-shimmer 1.5s ease-in-out infinite;
  }
  .sk-line.w-70 { width: 70%; }
  .sk-line.w-40 { width: 40%; }
</style>
