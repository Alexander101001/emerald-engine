import fetch from 'node-fetch';
import logger from '../utils/logger.js';
import stealthNetwork from './stealthNetwork.js';
import watchdog from '../agi/aegis-watchdog.js';

const DAILY_TOKEN_TARGET = 1000000;
const KEY_POOLS = {
  openai: [
    { key: process.env.OPENAI_API_KEY_1 || '', tier: 'free', dailyLimit: 200000, used: 0, resetAt: 0 },
    { key: process.env.OPENAI_API_KEY_2 || '', tier: 'free', dailyLimit: 200000, used: 0, resetAt: 0 },
    { key: process.env.OPENAI_API_KEY_3 || '', tier: 'free', dailyLimit: 200000, used: 0, resetAt: 0 },
    { key: process.env.OPENAI_API_KEY_4 || '', tier: 'free', dailyLimit: 200000, used: 0, resetAt: 0 },
    { key: process.env.OPENAI_API_KEY_5 || '', tier: 'free', dailyLimit: 200000, used: 0, resetAt: 0 },
  ],
  github: [
    { key: process.env.GITHUB_PAT_1 || process.env.GITHUB_PAT || '', tier: 'free', dailyLimit: 5000, used: 0, resetAt: 0 },
    { key: process.env.GITHUB_PAT_2 || '', tier: 'free', dailyLimit: 5000, used: 0, resetAt: 0 },
  ],
  supabase: [
    { key: process.env.SUPABASE_KEY_1 || '', tier: 'free', dailyLimit: 50000, used: 0, resetAt: 0 },
    { key: process.env.SUPABASE_KEY_2 || '', tier: 'free', dailyLimit: 50000, used: 0, resetAt: 0 },
  ],
};

export class ApiKeyRotator {
  constructor() {
    this._pools = JSON.parse(JSON.stringify(KEY_POOLS));
    this._active = {};
    this._rotations = 0;
    this._totalTokensToday = 0;
    this._enabled = false;
    this._dailyResetHour = Date.now();
    this._keyHealth = {};
    this._fallbackActive = false;
    for (const svc of Object.keys(this._pools)) {
      this._active[svc] = 0;
      this._keyHealth[svc] = this._pools[svc].map(() => ({ status: 'untested', lastPing: 0, latency: 0 }));
    }
  }

  activate() {
    this._enabled = true;
    const counts = {};
    for (const [svc, keys] of Object.entries(this._pools)) {
      const valid = keys.filter(k => k.key && k.key.length > 10);
      counts[svc] = valid.length;
    }
    logger.info(`api-rotator: active — ${JSON.stringify(counts)}`);
    this.checkPoolStatus().catch(() => {});
    setInterval(() => this._dailyReset(), 60000);
    return { active: true, keyCounts: counts, dailyTarget: DAILY_TOKEN_TARGET };
  }

  deactivate() {
    this._enabled = false;
    logger.info('api-rotator: deactivated');
  }

  getActiveKey(service) {
    if (!this._enabled) return process.env.OPENAI_API_KEY || process.env.GITHUB_PAT || '';
    const pool = this._pools[service];
    if (!pool || pool.length === 0) return this._fallbackKey(service);
    if (stealthNetwork.isFallbackMode()) {
      this._fallbackActive = true;
      return this._fallbackKey(service);
    }
    this._fallbackActive = false;
    const idx = this._active[service] || 0;
    const keyEntry = pool[idx];
    const health = this._keyHealth[service]?.[idx];
    if (health && health.status === 'dead') {
      const rotated = this._rotateToAvailable(service, pool);
      if (rotated !== null) {
        this._active[service] = rotated;
        this._rotations++;
        return pool[rotated].key;
      }
    }
    if (this._isOverLimit(keyEntry)) {
      const rotated = this._rotateToAvailable(service, pool);
      if (rotated !== null) {
        this._active[service] = rotated;
        this._rotations++;
        return pool[rotated].key;
      }
      logger.warn(`api-rotator: all keys over limit for ${service}`);
      return keyEntry.key;
    }
    return keyEntry.key;
  }

  recordTokens(service, count) {
    if (!this._enabled) return;
    const pool = this._pools[service];
    if (!pool) return;
    const idx = this._active[service] || 0;
    const entry = pool[idx];
    if (entry) {
      entry.used += count;
      this._totalTokensToday += count;
    }
  }

  handleRateLimit(service) {
    if (!this._enabled) return null;
    const pool = this._pools[service];
    if (!pool) return null;

    const currentIdx = this._active[service] || 0;
    const nextIdx = this._rotateToAvailable(service, pool);

    if (nextIdx !== null && nextIdx !== currentIdx) {
      this._active[service] = nextIdx;
      this._rotations++;
      logger.warn(`api-rotator: 429 on ${service} — rotated key ${currentIdx}->${nextIdx} (rotation #${this._rotations})`);
      return pool[nextIdx].key;
    }

    logger.warn(`api-rotator: 429 on ${service} — no alternative keys available`);
    return null;
  }

  async checkPoolStatus() {
    if (!this._enabled) return { error: 'not_active' };
    const results = [];
    for (const svc of Object.keys(this._pools)) {
      const endpoints = this._pingEndpoints(svc);
      for (let i = 0; i < this._pools[svc].length; i++) {
        const entry = this._pools[svc][i];
        if (!entry.key || entry.key.length < 10) {
          this._keyHealth[svc][i] = { status: 'unconfigured', lastPing: Date.now(), latency: 0 };
          continue;
        }
        const endpoint = endpoints[i % endpoints.length];
        const health = await this._pingEndpoint(svc, entry.key, endpoint);
        this._keyHealth[svc][i] = health;
        results.push({ service: svc, slot: i, status: health.status, latency: health.latency });
      }
    }
    const dead = results.filter(r => r.status === 'dead').length;
    const alive = results.filter(r => r.status === 'alive').length;
    const degraded = results.filter(r => r.status === 'degraded').length;
    logger.info(`api-rotator: pool check — ${alive} alive, ${degraded} degraded, ${dead} dead, ${results.length} total`);
    return { results, alive, dead, degraded };
  }

  getRotatorStatus() {
    const pools = {};
    for (const [svc, keys] of Object.entries(this._pools)) {
      pools[svc] = keys.map((k, i) => {
        const health = this._keyHealth[svc]?.[i] || { status: 'untested', lastPing: 0, latency: 0 };
        return {
          slot: i,
          active: this._active[svc] === i,
          used: k.used,
          limit: k.dailyLimit,
          usagePercent: k.dailyLimit > 0 ? ((k.used / k.dailyLimit) * 100).toFixed(1) + '%' : '0%',
          pingStatus: health.status,
          pingLatency: health.latency > 0 ? health.latency + 'ms' : '-',
          lastPing: health.lastPing > 0 ? new Date(health.lastPing).toISOString() : '-',
        };
      });
    }
    return {
      enabled: this._enabled,
      rotations: this._rotations,
      totalTokensToday: this._totalTokensToday,
      dailyTarget: DAILY_TOKEN_TARGET,
      fallbackMode: this._fallbackActive,
      poolStatus: pools,
      networkFallback: stealthNetwork.isFallbackMode(),
    };
  }

  _pingEndpoints(service) {
    if (service === 'openai') return [
      { url: 'https://api.openai.com/v1/models', headers: { 'Authorization': 'Bearer {KEY}' }, method: 'GET' },
    ];
    if (service === 'github') return [
      { url: 'https://api.github.com/user', headers: { 'Authorization': 'Bearer {KEY}', 'User-Agent': 'Emerald-AGI' }, method: 'GET' },
    ];
    if (service === 'supabase') return [
      { url: 'https://api.supabase.com/v1/projects', headers: { 'apiKey': '{KEY}', 'Authorization': 'Bearer {KEY}' }, method: 'GET' },
    ];
    return [];
  }

  async _pingEndpoint(service, key, endpoint) {
    const start = Date.now();
    const url = endpoint.url;
    const headers = {};
    for (const [k, v] of Object.entries(endpoint.headers || {})) {
      headers[k] = v.replace('{KEY}', key);
    }
    try {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 8000);
      const res = await watchdog.secureFetch(url, {
        method: endpoint.method || 'GET',
        headers,
        signal: controller.signal,
      });
      clearTimeout(timeout);
      const latency = Date.now() - start;
      if (res.ok) {
        return { status: 'alive', lastPing: Date.now(), latency };
      }
      if (res.status === 401 || res.status === 403) {
        return { status: 'dead', lastPing: Date.now(), latency };
      }
      return { status: 'degraded', lastPing: Date.now(), latency };
    } catch {
      const latency = Date.now() - start;
      return { status: 'dead', lastPing: Date.now(), latency };
    }
  }

  _isOverLimit(entry) {
    if (!entry || entry.dailyLimit === 0) return false;
    const now = Date.now();
    if (now - entry.resetAt > 86400000) {
      entry.used = 0;
      entry.resetAt = now;
      return false;
    }
    return entry.used >= entry.dailyLimit;
  }

  _rotateToAvailable(service, pool) {
    const currentIdx = this._active[service] || 0;
    for (let i = 0; i < pool.length; i++) {
      const idx = (currentIdx + 1 + i) % pool.length;
      if (idx === currentIdx) continue;
      if (pool[idx].key && !this._isOverLimit(pool[idx])) {
        return idx;
      }
    }
    return null;
  }

  _dailyReset() {
    const now = Date.now();
    if (now - this._dailyResetHour >= 86400000) {
      for (const pool of Object.values(this._pools)) {
        for (const entry of pool) {
          entry.used = 0;
          entry.resetAt = now;
        }
      }
      this._totalTokensToday = 0;
      this._dailyResetHour = now;
      logger.info('api-rotator: daily token counters reset');
    }
  }

  _fallbackKey(service) {
    if (service === 'openai') return process.env.OPENAI_API_KEY || '';
    if (service === 'github') return process.env.GITHUB_PAT || '';
    return '';
  }
}

const apiRotator = new ApiKeyRotator();
export default apiRotator;
