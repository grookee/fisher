const enabled = typeof import.meta !== 'undefined' && import.meta.env.VITE_DEBUG === 'true';

export function debug(...args) {
  if (!enabled) return;
  console.log('[DEBUG]', ...args);
}

export function debugf(format, ...args) {
  if (!enabled) return;
  console.log('[DEBUG]', format.replace(/%[sd]/g, m => {
    const v = args.shift();
    return v !== undefined ? String(v) : m;
  }));
}

export function isDebug() {
  return enabled;
}
