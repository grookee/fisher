<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { isAuthenticated } from '$lib/stores/auth';

  let genres = [];
  let loading = true;

  onMount(async () => {
    try {
      genres = await api.tastes.genres();
    } catch {}
    loading = false;
  });
</script>

<div class="hero">
  <h1>explore the <span class="accent">universe</span> of music</h1>
  <p class="subtitle">
    navigate genres, discover sounds, and share your taste with friends.
    connected to Spotify and Apple Music.
  </p>
  <div class="cta">
    {#if !$isAuthenticated}
      <a href="/auth" class="btn-primary btn-lg">get started</a>
    {:else}
      <a href="/genres" class="btn-primary btn-lg">start exploring</a>
    {/if}
  </div>
</div>

{#if !loading}
  <div class="genre-cloud">
    <h2>genre map</h2>
    <div class="cloud">
      {#each genres as genre}
        <a
          href="/genres?focus={genre.id}"
          class="genre-tag"
          style="background: {genre.color}22; border-color: {genre.color}; color: {genre.color}"
        >
          {genre.name}
        </a>
      {/each}
    </div>
  </div>
{/if}

<style>
  .hero {
    text-align: center;
    padding: 4rem 2rem 3rem;
  }
  .hero h1 {
    font-size: 3rem;
    font-weight: 800;
    margin: 0;
    line-height: 1.1;
  }
  .accent {
    color: #818cf8;
  }
  .subtitle {
    color: #888;
    font-size: 1.1rem;
    max-width: 500px;
    margin: 1rem auto 2rem;
    line-height: 1.6;
  }
  .btn-lg {
    padding: 0.8rem 2rem;
    font-size: 1rem;
  }
  .btn-primary {
    background: #818cf8;
    color: #000;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-weight: 600;
    display: inline-block;
  }
  .btn-primary:hover {
    background: #a5b4fc;
  }
  .genre-cloud {
    padding: 2rem 0;
  }
  .genre-cloud h2 {
    font-size: 1.2rem;
    margin-bottom: 1rem;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }
  .cloud {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .genre-tag {
    padding: 0.5rem 1.1rem;
    border-radius: 20px;
    border: 1px solid;
    font-size: 0.9rem;
    transition: all 0.15s;
  }
  .genre-tag:hover {
    transform: scale(1.05);
    filter: brightness(1.3);
  }
</style>
