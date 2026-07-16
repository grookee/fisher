<script>
  export let query = '';
</script>

<div class="explore-header">
  <h1>explore</h1>
  <div class="header-actions">
    <div class="search-bar">
      <input type="text" placeholder="search genres..." bind:value={query} />
    </div>
  </div>
</div>

<style>
  .explore-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.5rem; gap: 1rem; flex-wrap: wrap; }
  .explore-header h1 { margin: 0; font-size: 1.6rem; }
  .header-actions { display: flex; gap: 0.5rem; align-items: center; flex-wrap: wrap; }
  .search-bar input { padding: 0.45rem 0.9rem; background: #1a1a1a; border: 1px solid #333; border-radius: 8px; color: #fff; font-size: 0.85rem; width: 200px; }
  .search-bar input:focus { outline: none; border-color: #818cf8; }

  @media (max-width: 940px) {
    .search-bar input { width: min(60vw, 240px); }
  }

  @media (max-width: 600px) {
    .explore-header h1 { font-size: 1.35rem; }
  }
</style>
