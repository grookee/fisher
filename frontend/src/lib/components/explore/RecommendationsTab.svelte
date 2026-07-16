<script>
  import { api } from '$lib/api/client';

  export let loadingRecs = false;
  export let recs = [];
  export let deepCuts = [];
  export let allGenres = [];

  export let bg = () => '';
  export let onPickRec = () => {};
  export let onSettingsChanged = () => {};

  let showSettings = false;
  let obscureOnly = false;
  let ignoredGenreIds = [];
  let saving = false;

  async function loadSettings() {
    try {
      const settings = await api.settings.getAll();
      if (settings.obscure_only === 'true' || settings.obscure_only === '1') {
        obscureOnly = true;
      }
      if (settings.ignored_genres) {
        ignoredGenreIds = JSON.parse(settings.ignored_genres);
      }
    } catch {}
  }

  loadSettings();

  function isIgnored(genreId) {
    return ignoredGenreIds.includes(genreId);
  }

  async function toggleIgnore(genreId) {
    if (ignoredGenreIds.includes(genreId)) {
      ignoredGenreIds = ignoredGenreIds.filter((id) => id !== genreId);
    } else {
      ignoredGenreIds = [...ignoredGenreIds, genreId];
    }
    await saveSettings();
  }

  async function toggleObscureOnly() {
    obscureOnly = !obscureOnly;
    await saveSettings();
  }

  async function saveSettings() {
    saving = true;
    try {
      await Promise.all([
        api.settings.set('obscure_only', String(obscureOnly)),
        api.settings.set('ignored_genres', JSON.stringify(ignoredGenreIds)),
      ]);
    } catch {}
    saving = false;
    onSettingsChanged();
  }

  function pct(v) {
    return Math.round((v || 0) * 100);
  }
</script>

{#if loadingRecs}
  <div class="rec-grid">
    {#each Array(6) as _}
      <div class="sk-card"></div>
    {/each}
  </div>
{:else}
  <div class="rec-header">
    <div class="section-label" style="margin-top:0">obscure gems tailored to your taste, region, and friend circle</div>
    <button class="settings-toggle" on:click={() => { showSettings = !showSettings; }} title="recommendation settings">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="3"/><path d="M12 1v2m0 18v2M4.22 4.22l1.42 1.42m12.72 12.72 1.42 1.42M1 12h2m18 0h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
      </svg>
    </button>
  </div>

  {#if showSettings}
    <div class="rec-settings">
      <label class="setting-row">
        <input type="checkbox" checked={obscureOnly} on:change={toggleObscureOnly} />
        <span>only show obscure genres (rarity &ge; 60%)</span>
      </label>

      {#if allGenres.length > 0}
        <div class="ignore-section">
          <span class="ignore-label">ignore genres:</span>
          <div class="ignore-pills">
            {#each allGenres.slice(0, 40) as g}
              <button
                class="ignore-pill"
                class:active={!isIgnored(g.id)}
                style="--c: {g.color}"
                on:click={() => toggleIgnore(g.id)}
              >
                {g.name}
              </button>
            {/each}
          </div>
        </div>
      {/if}

      {#if saving}
        <span class="saving-hint">saving...</span>
      {/if}
    </div>
  {/if}

  {#if recs.length > 0}
    <div class="rec-grid">
      {#each recs as g}
        <button class="rec-card" style="border-color: {g.color}; {bg(g)}" on:click={() => onPickRec(g)}>
          <div class="rec-head">
            <span class="rec-name" style="color: {g.color}">{g.name}</span>
            <span class="rec-score" style="border-color: {g.color}88">{pct(g.score)}%</span>
          </div>

          {#if g.countries?.length}
            <div class="rec-countries">{g.countries.slice(0, 3).join(' · ')}</div>
          {/if}

          <p class="rec-reason">{g.reason}</p>

          <div class="rec-bars">
            <div class="rec-bar" title="taste similarity — {pct(g.taste_similarity)}%">
              <span class="rec-bar-label">taste</span>
              <div class="rec-bar-bg"><div class="rec-bar-fill" style="width:{pct(g.taste_similarity)}%; background:{g.color}"></div></div>
            </div>
            <div class="rec-bar" title="how undiscovered this still is — {pct(g.obscurity)}%">
              <span class="rec-bar-label">rarity</span>
              <div class="rec-bar-bg"><div class="rec-bar-fill" style="width:{pct(g.obscurity)}%; background:{g.color}"></div></div>
            </div>
            <div class="rec-bar" title="regional novelty — {pct(g.regional_novelty)}%">
              <span class="rec-bar-label">region</span>
              <div class="rec-bar-bg"><div class="rec-bar-fill" style="width:{pct(g.regional_novelty)}%; background:{g.color}"></div></div>
            </div>
            {#if g.friend_novelty > 0}
              <div class="rec-bar" title="new within your friend circle — {pct(g.friend_novelty)}%">
                <span class="rec-bar-label">friends</span>
                <div class="rec-bar-bg"><div class="rec-bar-fill" style="width:{pct(g.friend_novelty)}%; background:{g.color}"></div></div>
              </div>
            {/if}
          </div>
        </button>
      {/each}
    </div>
  {:else if !showSettings}
    <div class="empty-state">
      <p>No personalized recommendations yet.</p>
      <p class="hint">
        Analyze your taste on the <a href="/tastes">my taste</a> page, then check back once the genre
        graph has been built (this refreshes automatically on a schedule).
      </p>
    </div>
  {/if}

  {#if deepCuts.length > 0}
    <div class="section-label">deep cuts — genres not yet explored by anyone</div>
    <div class="rec-grid">
      {#each deepCuts as g}
        <button class="rec-card deep-cut" style="border-color: {g.color}44; {bg(g)}" on:click={() => onPickRec(g)}>
          <div class="rec-head">
            <span class="rec-name" style="color: {g.color}">{g.name}</span>
            <span class="rec-score" style="border-color: {g.color}44">{pct(g.obscurity)}% rare</span>
          </div>

          {#if g.countries?.length}
            <div class="rec-countries">{g.countries.slice(0, 3).join(' · ')}</div>
          {/if}

          <p class="rec-reason">{g.reason}</p>
        </button>
      {/each}
    </div>
  {/if}
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

  .rec-header { display: flex; align-items: center; gap: 0.5rem; }
  .section-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 1px; color: #555; margin-bottom: 0.6rem; margin-top: 0.5rem; }

  .settings-toggle {
    background: none; border: 1px solid #333; border-radius: 6px; color: #666;
    padding: 0.2rem 0.35rem; cursor: pointer; display: flex; align-items: center;
    transition: all 0.12s; margin-left: auto;
  }
  .settings-toggle:hover { color: #aaa; border-color: #555; }

  .rec-settings {
    background: #161616; border: 1px solid #2a2a2a; border-radius: 8px;
    padding: 0.7rem 0.85rem; margin-bottom: 0.7rem; animation: fade-up 0.12s ease-out;
  }
  .setting-row { display: flex; align-items: center; gap: 0.5rem; font-size: 0.75rem; color: #999; cursor: pointer; }
  .setting-row input[type="checkbox"] { accent-color: #818cf8; }

  .ignore-section { margin-top: 0.6rem; }
  .ignore-label { font-size: 0.65rem; text-transform: uppercase; letter-spacing: 0.5px; color: #666; display: block; margin-bottom: 0.35rem; }
  .ignore-pills { display: flex; flex-wrap: wrap; gap: 0.3rem; }
  .ignore-pill {
    font-size: 0.6rem; padding: 0.15rem 0.45rem; border-radius: 8px; cursor: pointer;
    border: 1px solid var(--c, #444); background: transparent; color: var(--c, #999);
    transition: all 0.1s; text-transform: capitalize;
  }
  .ignore-pill.active { background: var(--c, #444); color: #111; }
  .ignore-pill:hover { opacity: 0.8; }

  .saving-hint { font-size: 0.6rem; color: #555; margin-top: 0.3rem; display: block; }

  .rec-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 0.7rem; animation: fade-up 0.16s ease-out; }

  .sk-card { height: 168px; border-radius: 10px; background: linear-gradient(90deg, #1a1a1a 0%, #2a2a2a 50%, #1a1a1a 100%); background-size: 200px 100%; animation: sk-shimmer 1.5s ease-in-out infinite; }

  .rec-card {
    background: #111;
    border: 1px solid #333;
    border-radius: 10px;
    padding: 0.85rem;
    cursor: pointer;
    transition: all 0.12s;
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
    text-align: left;
    overflow: hidden;
    min-width: 0;
  }
  .rec-card:hover { transform: translateY(-2px); box-shadow: 0 4px 16px rgba(0,0,0,0.4); }
  .rec-card.deep-cut { background: #0e0e0e; border-style: dashed; }

  .rec-head { display: flex; align-items: center; justify-content: space-between; gap: 0.4rem; min-width: 0; }
  .rec-name { font-weight: 700; font-size: 0.9rem; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .rec-score { font-size: 0.65rem; color: #ccc; border: 1px solid #444; border-radius: 10px; padding: 0.1rem 0.45rem; white-space: nowrap; }

  .rec-countries { font-size: 0.65rem; text-transform: uppercase; letter-spacing: 0.4px; color: #777; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; }

  .rec-reason { font-size: 0.75rem; color: #999; margin: 0; line-height: 1.3; min-width: 0; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }

  .rec-bars { display: flex; flex-direction: column; gap: 0.25rem; margin-top: auto; min-width: 0; }
  .rec-bar { display: flex; align-items: center; gap: 0.4rem; min-width: 0; overflow: hidden; }
  .rec-bar-label { font-size: 0.6rem; color: #666; width: 34px; flex-shrink: 0; text-transform: uppercase; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .rec-bar-bg { flex: 1; min-width: 0; height: 4px; background: #222; border-radius: 2px; overflow: hidden; }
  .rec-bar-fill { height: 100%; border-radius: 2px; }

  .empty-state { text-align: center; padding: 2rem; color: #888; }
  .empty-state .hint { font-size: 0.8rem; margin-top: 0.3rem; }
  .empty-state .hint a { color: #818cf8; }
</style>
