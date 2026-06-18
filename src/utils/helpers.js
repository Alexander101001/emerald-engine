import crypto from 'crypto';

export function slugify(text) {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60) || 'page';
}

export function uid(len = 12) {
  return crypto.randomBytes(len).toString('hex').slice(0, len);
}

export function sanitizeHtml(text) {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

export function debounce(fn, ms) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), ms);
  };
}

export function cache(ttl = 3600000) {
  const store = new Map();
  return {
    get(key) {
      const entry = store.get(key);
      if (!entry) return null;
      if (Date.now() - entry.ts > ttl) { store.delete(key); return null; }
      return entry.val;
    },
    set(key, val) { store.set(key, { val, ts: Date.now() }); },
    clear() { store.clear(); },
  };
}
