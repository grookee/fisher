import { writable, derived } from 'svelte/store';

const stored = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null;
export const token = writable(stored || '');

export const isAuthenticated = derived(token, ($token) => !!$token);

token.subscribe((val) => {
  if (typeof localStorage !== 'undefined') {
    if (val) localStorage.setItem('token', val);
    else localStorage.removeItem('token');
  }
});

export const user = writable(null);

export async function fetchUser() {
  const { api } = await import('../api/client');
  try {
    const u = await api.auth.me();
    user.set(u);
  } catch {
    token.set('');
    user.set(null);
  }
}
