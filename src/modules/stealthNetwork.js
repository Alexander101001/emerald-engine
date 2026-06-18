import fetch from 'node-fetch';
import logger from '../utils/logger.js';
import watchdog from '../agi/aegis-watchdog.js';

const PROXY_SOURCES = [
  { env: 'PROXY_1', host: null, port: null, protocol: 'socks5' },
  { env: 'PROXY_2', host: null, port: null, protocol: 'socks5' },
  { env: 'PROXY_3', host: null, port: null, protocol: 'socks5' },
  { env: 'PROXY_4', host: null, port: null, protocol: 'socks5' },
  { env: 'PROXY_5', host: null, port: null, protocol: 'socks5' },
];

const PROXY_TEST_TIMEOUT_MS = 5000;
const PROXY_TEST_URLS = [
  'https://api.ipify.org?format=json',
  'https://httpbin.org/ip',
  'https://ifconfig.me/all.json',
];

const OFFLINE_COOLDOWN_MS = 120000;

export class StealthNetwork {
  constructor() {
    this._proxies = [];
    this._lastTest = 0;
    this._testIntervalMs = 30000;
    this._fallbackMode = false;
    this._enabled = false;
    this._initProxies();
  }

  activate() {
    this._enabled = true;
    this._initProxies();
    this.testAllProxies();
    logger.info(`stealth-network: active — ${this._proxies.length} proxies monitored, 5s timeout`);
    return { active: true, proxyCount: this._proxies.length };
  }

  deactivate() {
    this._enabled = false;
    logger.info('stealth-network: deactivated');
  }

  async getProxy() {
    if (!this._enabled) return null;
    if (Date.now() - this._lastTest > this._testIntervalMs) {
      await this._testAllProxies();
    }
    const online = this._proxies.filter(p => p.status === 'online');
    if (online.length === 0) {
      return this._enterFallbackMode();
    }
    this._fallbackMode = false;
    const selected = online[Math.floor(Math.random() * online.length)];
    selected.lastUsed = Date.now();
    return selected.url;
  }

  async testProxy(source) {
    if (!source.url) return;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), PROXY_TEST_TIMEOUT_MS);
    source.lastTested = Date.now();
    try {
      const testUrl = PROXY_TEST_URLS[Math.floor(Math.random() * PROXY_TEST_URLS.length)];
      const res = await watchdog.secureFetch(testUrl, {
        signal: controller.signal,
        timeout: PROXY_TEST_TIMEOUT_MS,
      });
      clearTimeout(timeout);
      if (res.ok) {
        source.status = 'online';
        source.statusChangedAt = Date.now();
        source.consecutiveFailures = 0;
      } else {
        this._markOffline(source);
      }
    } catch {
      clearTimeout(timeout);
      source.consecutiveFailures = (source.consecutiveFailures || 0) + 1;
      if (source.consecutiveFailures >= 2) {
        this._markOffline(source);
      } else {
        source.status = 'degraded';
        source.statusChangedAt = Date.now();
      }
    }
  }

  async testAllProxies() {
    this._lastTest = Date.now();
    const results = await Promise.allSettled(
      this._proxies.map(p => this.testProxy(p))
    );
    const online = this._proxies.filter(p => p.status === 'online').length;
    const offline = this._proxies.filter(p => p.status === 'offline').length;
    const degraded = this._proxies.filter(p => p.status === 'degraded').length;
    const bypassed = this._proxies.filter(p => p.status === 'bypassed').length;
    if (this._proxies.length > 0) {
      logger.info(`stealth-network: proxy scan — ${online} online, ${degraded} degraded, ${offline} offline, ${bypassed} bypassed`);
    }
    this._recoverOfflineProxies();
    return { online, offline, degraded, bypassed, total: this._proxies.length };
  }

  getNetworkStatus() {
    const proxies = this._proxies.map(p => ({
      url: p.url ? p.url.replace(/\/\/.*@/, '//***:***@') : '(none)',
      status: p.status,
      lastTested: p.lastTested ? new Date(p.lastTested).toISOString() : null,
      statusChangedAt: p.statusChangedAt ? new Date(p.statusChangedAt).toISOString() : null,
      consecutiveFailures: p.consecutiveFailures || 0,
    }));
    return {
      enabled: this._enabled,
      fallbackMode: this._fallbackMode,
      proxyCount: this._proxies.length,
      online: proxies.filter(p => p.status === 'online').length,
      offline: proxies.filter(p => p.status === 'offline' || p.status === 'bypassed').length,
      degraded: proxies.filter(p => p.status === 'degraded').length,
      lastFullTest: this._lastTest ? new Date(this._lastTest).toISOString() : null,
      proxies,
    };
  }

  isFallbackMode() {
    return this._fallbackMode;
  }

  _initProxies() {
    this._proxies = PROXY_SOURCES.map(s => {
      const raw = process.env[s.env] || '';
      return {
        env: s.env,
        url: raw,
        host: raw ? raw.split('://')[1]?.split(':')[0] || null : null,
        port: raw ? parseInt(raw.split(':').pop(), 10) || null : null,
        protocol: s.protocol,
        status: raw ? 'untested' : 'bypassed',
        lastTested: 0,
        statusChangedAt: 0,
        consecutiveFailures: 0,
        lastUsed: 0,
      };
    }).filter(p => p.url);
    if (this._proxies.length === 0) {
      logger.info('stealth-network: no proxies configured — running in direct mode');
    }
  }

  _markOffline(source) {
    source.status = 'offline';
    source.statusChangedAt = Date.now();
    logger.warn(`stealth-network: proxy ${source.env} marked offline`);
  }

  _recoverOfflineProxies() {
    const now = Date.now();
    for (const p of this._proxies) {
      if (p.status === 'offline' && now - p.statusChangedAt > OFFLINE_COOLDOWN_MS) {
        p.status = 'untested';
        p.consecutiveFailures = 0;
        logger.info(`stealth-network: proxy ${p.env} re-enabled for testing after cooldown`);
      }
    }
  }

  _enterFallbackMode() {
    if (!this._fallbackMode) {
      logger.warn('stealth-network: all proxies offline — entering fallback direct mode');
      this._fallbackMode = true;
    }
    return null;
  }
}

const stealthNetwork = new StealthNetwork();
export default stealthNetwork;
