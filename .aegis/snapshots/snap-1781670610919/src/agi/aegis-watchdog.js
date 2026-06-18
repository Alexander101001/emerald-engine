import fetch from 'node-fetch';
import { createHash } from 'crypto';
import logger from '../utils/logger.js';

const ALLOWED_DOMAINS = [
  'api.github.com',
  'api.telegram.org',
  'api.openai.com',
  'api.together.xyz',
  'openrouter.ai',
  'news.ycombinator.com',
  'reddit.com',
  'www.reddit.com',
  'producthunt.com',
  'www.producthunt.com',
];

const BLOCKED_PATTERNS = [
  /(?:git|telegram|openrouter|openai|together)_(?:pat|token|key|secret)/i,
  /ghp_[a-zA-Z0-9]{36,}/,
  /gho_[a-zA-Z0-9]{36,}/,
  /sk-[a-zA-Z0-9]{20,}/,
];

class AegisWatchdog {
  constructor() {
    this._active = false;
    this._violations = [];
    this._blockedCount = 0;
    this._passedCount = 0;
    this._orchestrator = null;
    this._keyHashes = new Set();
  }

  activate(orchestrator) {
    this._orchestrator = orchestrator;
    this._keyHashes.clear();

    const keys = orchestrator.listKeys();
    for (const keyName of keys) {
      const val = orchestrator.get(keyName);
      if (val && !val.startsWith('[INSERT')) {
        this._keyHashes.add(createHash('sha256').update(val).digest('hex'));
      }
    }

    this._active = true;
    logger.info(`watchdog: activated — ${this._keyHashes.size} key fingerprints registered`);
    return { active: true, fingerprints: this._keyHashes.size };
  }

  async secureFetch(url, options = {}) {
    if (!this._active) {
      return fetch(url, options);
    }

    const violation = this._inspect(url, options);
    if (violation) {
      this._blockedCount++;
      this._violations.push({ ts: Date.now(), type: violation, url: this._sanitizeUrl(url) });
      logger.warn(`watchdog: BLOCKED — ${violation} to ${this._sanitizeUrl(url)}`);
      throw new Error(`Watchdog blocked: ${violation}`);
    }

    const domain = this._extractDomain(url);
    if (!ALLOWED_DOMAINS.some(d => domain.endsWith(d))) {
      this._blockedCount++;
      this._violations.push({ ts: Date.now(), type: 'unauthorized_domain', url: this._sanitizeUrl(url) });
      logger.warn(`watchdog: BLOCKED — unauthorized domain ${domain}`);
      throw new Error(`Watchdog blocked: unauthorized domain ${domain}`);
    }

    this._passedCount++;
    return fetch(url, options);
  }

  _inspect(url, options) {
    const urlStr = typeof url === 'string' ? url : url.href || '';
    const headers = options?.headers || {};
    const body = options?.body || '';

    const combined = urlStr + '|' + JSON.stringify(headers) + '|' + (typeof body === 'string' ? body : '');

    for (const pattern of BLOCKED_PATTERNS) {
      if (pattern.test(combined)) {
        return 'blocked_pattern_match';
      }
    }

    const hash = createHash('sha256').update(combined).digest('hex');
    for (const keyHash of this._keyHashes) {
      if (combined.includes(keyHash)) {
        return 'key_leak_detected';
      }
    }

    if (this._orchestrator) {
      for (const keyName of this._orchestrator.listKeys()) {
        const val = this._orchestrator.get(keyName);
        if (val && !val.startsWith('[INSERT') && val.length > 8) {
          if (combined.includes(val)) {
            return `key_leak_${keyName}`;
          }
        }
      }
    }

    return null;
  }

  _extractDomain(url) {
    try {
      const u = typeof url === 'string' ? new URL(url) : url;
      return u.hostname.replace(/^www\./, '');
    } catch {
      return 'unknown';
    }
  }

  _sanitizeUrl(url) {
    try {
      const u = typeof url === 'string' ? new URL(url) : url;
      const sanitized = new URL(u.origin + u.pathname);
      return sanitized.href;
    } catch {
      return '(invalid)';
    }
  }

  deactivate() {
    this._active = false;
    this._keyHashes.clear();
    this._orchestrator = null;
    logger.info('watchdog: deactivated');
  }

  getStatus() {
    return {
      active: this._active,
      blocked: this._blockedCount,
      passed: this._passedCount,
      violations: this._violations.length,
      recentViolations: this._violations.slice(-5).map(v => ({
        type: v.type, url: v.url, ts: new Date(v.ts).toISOString(),
      })),
    };
  }
}

const watchdog = new AegisWatchdog();
export default watchdog;
