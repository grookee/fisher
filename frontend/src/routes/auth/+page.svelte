<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { api } from '$lib/api/client';
  import { token, isAuthenticated, fetchUser } from '$lib/stores/auth';

  let isLogin = true;
  let email = '';
  let username = '';
  let password = '';
  let error = '';
  let loading = false;
  let linkedService = '';

  const emailRegex = /^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$/;

  onMount(() => {
    if ($isAuthenticated) { goto('/tastes'); return; }
    linkedService = $page.url.searchParams.get('linked') || '';
  });

  function validate() {
    if (!email.trim()) { error = 'email is required'; return false; }
    if (!emailRegex.test(email.trim())) { error = 'please enter a valid email address'; return false; }
    if (!isLogin && !username.trim()) { error = 'username is required'; return false; }
    if (!isLogin && username.trim().length < 2) { error = 'username must be at least 2 characters'; return false; }
    if (!password) { error = 'password is required'; return false; }
    if (password.length < 6) { error = 'password must be at least 6 characters'; return false; }
    return true;
  }

  async function handleSubmit() {
    error = '';
    if (!validate()) return;
    loading = true;
    try {
      let res;
      if (isLogin) {
        res = await api.auth.login({ email: email.trim().toLowerCase(), password });
      } else {
        res = await api.auth.register({ email: email.trim().toLowerCase(), username: username.trim(), password });
      }
      token.set(res.token);
      await fetchUser();
      goto('/tastes');
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }
</script>

<div class="auth-page">
  <div class="auth-card">
    <h1>{isLogin ? 'sign in' : 'create account'}</h1>

    {#if linkedService}
      <div class="success">your {linkedService} account has been linked!</div>
    {/if}

    <form on:submit|preventDefault={handleSubmit}>
      <div class="field">
        <label for="email">email</label>
        <input id="email" type="email" bind:value={email} required placeholder="you@example.com" />
      </div>

      {#if !isLogin}
        <div class="field">
          <label for="username">username</label>
          <input id="username" type="text" bind:value={username} required placeholder="fisherman" />
        </div>
      {/if}

      <div class="field">
        <label for="password">password</label>
        <input id="password" type="password" bind:value={password} required placeholder="min 6 characters" />
      </div>

      {#if error}
        <div class="error">{error}</div>
      {/if}

      <button type="submit" class="btn-primary" disabled={loading}>
        {loading ? '...' : isLogin ? 'sign in' : 'create account'}
      </button>
    </form>

    <p class="toggle">
      {isLogin ? "don't have an account?" : 'already have an account?'}
      <button class="link" on:click={() => { isLogin = !isLogin; error = ''; }}>
        {isLogin ? 'sign up' : 'sign in'}
      </button>
    </p>
  </div>
</div>

<style>
  .auth-page {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 80vh;
  }
  .auth-card {
    background: #151515;
    padding: 2.5rem;
    border-radius: 12px;
    border: 1px solid #222;
    width: 100%;
    max-width: 400px;
  }
  .auth-card h1 {
    margin: 0 0 1.5rem;
    font-size: 1.5rem;
    text-align: center;
  }
  .field {
    margin-bottom: 1rem;
  }
  .field label {
    display: block;
    font-size: 0.85rem;
    color: #888;
    margin-bottom: 0.3rem;
  }
  .field input {
    width: 100%;
    padding: 0.6rem 0.8rem;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 6px;
    color: #fff;
    font-size: 0.95rem;
  }
  .field input:focus {
    outline: none;
    border-color: #818cf8;
  }
  .btn-primary {
    width: 100%;
    padding: 0.7rem;
    background: #818cf8;
    color: #000;
    border: none;
    border-radius: 6px;
    font-weight: 600;
    font-size: 0.95rem;
    cursor: pointer;
    margin-top: 0.5rem;
  }
  .btn-primary:hover {
    background: #a5b4fc;
  }
  .btn-primary:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .error {
    color: #f87171;
    font-size: 0.85rem;
    margin-bottom: 0.5rem;
  }
  .success {
    color: #4caf50;
    font-size: 0.85rem;
    margin-bottom: 0.5rem;
    text-align: center;
  }
  .toggle {
    text-align: center;
    font-size: 0.85rem;
    color: #888;
    margin-top: 1rem;
  }
  .link {
    background: none;
    border: none;
    color: #818cf8;
    cursor: pointer;
    font-size: 0.85rem;
    padding: 0;
  }
  .link:hover {
    color: #a5b4fc;
  }
</style>
