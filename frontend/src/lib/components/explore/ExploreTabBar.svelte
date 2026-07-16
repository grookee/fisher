<script>
  export let tab = 'genres';
  export let onSelect = () => {};
</script>

<div class="tab-bar">
  <button class="tab" class:active={tab === 'genres'} on:click={() => onSelect('genres')}>all genres</button>
  <button class="tab" class:active={tab === 'surprise'} on:click={() => onSelect('surprise')}>surprise me</button>
  <button class="tab" class:active={tab === 'timemachine'} on:click={() => onSelect('timemachine')}>time machine</button>
  <button class="tab" class:active={tab === 'missed'} on:click={() => onSelect('missed')}>missed gems</button>
  <button class="tab" class:active={tab === 'foryou'} on:click={() => onSelect('foryou')}>for you</button>
</div>

<style>
  .tab-bar { display: flex; gap: 0.25rem; margin-bottom: 1rem; border-bottom: 1px solid #222; }
  .tab { background: none; border: none; color: #666; padding: 0.5rem 1rem; cursor: pointer; font-size: 0.8rem; font-weight: 500; border-bottom: 2px solid transparent; margin-bottom: -1px; transition: color 0.15s, border-color 0.15s, transform 0.15s; }
  .tab:hover { color: #ccc; transform: translateY(-1px); }
  .tab.active { color: #818cf8; border-bottom-color: #818cf8; }

  @media (max-width: 940px) {
    .tab { padding: 0.45rem 0.65rem; font-size: 0.76rem; }
  }

  @media (max-width: 600px) {
    .tab-bar { overflow-x: auto; padding-bottom: 0.15rem; }
    .tab { white-space: nowrap; }
  }
</style>
