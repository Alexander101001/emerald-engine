import aegis from './aegis.js';
import watchdog from './aegis-watchdog.js';
import logger from '../utils/logger.js';

const _vault = new Map();

function _zero(buf) {
  if (typeof buf === 'string') return 'x'.repeat(buf.length);
  return buf;
}

export class SecureOrchestrator {
  constructor() {
    this._loaded = false;
    this._loadedAt = null;
    this._accessCount = 0;
  }

  async loadVault(passphrase) {
    if (this._loaded) {
      logger.warn('orchestrator: vault already loaded — call wipe() first to reload');
      return { loaded: true, keyCount: _vault.size };
    }

    const integrity = aegis.verifyVaultIntegrity();
    if (!integrity.ok) {
      throw new Error(`Vault integrity check failed — ${integrity.reason || 'hash_mismatch'}`);
    }

    const keys = aegis.vaultDecrypt();
    _vault.clear();
    for (const [k, v] of Object.entries(keys)) {
      _vault.set(k, v);
    }

    this._loaded = true;
    this._loadedAt = Date.now();
    logger.info(`orchestrator: vault loaded — ${_vault.size} keys in volatile memory`);
    return { loaded: true, keyCount: _vault.size };
  }

  get(key) {
    if (!this._loaded) throw new Error('Orchestrator vault not loaded');
    this._accessCount++;
    return _vault.get(key) || null;
  }

  has(key) {
    return this._loaded && _vault.has(key);
  }

  listKeys() {
    if (!this._loaded) return [];
    return Array.from(_vault.keys());
  }

  getGitHubToken() {
    return this.get('GITHUB_TOKEN');
  }

  getTelegramConfig() {
    const raw = this.get('TELEGRAM_CONFIG');
    if (!raw || raw.startsWith('[INSERT')) return null;
    try { return JSON.parse(raw); } catch { return null; }
  }

  getTelegramBotToken() {
    const cfg = this.getTelegramConfig();
    return cfg ? cfg.bot_token || null : null;
  }

  getTelegramChannelId() {
    const cfg = this.getTelegramConfig();
    return cfg ? cfg.channel_id || null : null;
  }

  getTelegramUserId() {
    const cfg = this.getTelegramConfig();
    return cfg ? cfg.user_id || null : null;
  }

  getOpenRouterKey() {
    return this.get('OPENROUTER_API_KEY');
  }

  getOtherAIKeys() {
    const raw = this.get('OTHER_AI_API_KEYS');
    if (!raw || raw.startsWith('[INSERT')) return {};
    try { return JSON.parse(raw); } catch { return { raw }; }
  }

  getVercelToken() {
    return this.get('VERCEL_TOKEN');
  }

  getNetlifyToken() {
    return this.get('NETLIFY_TOKEN');
  }

  getExchangeApiKeys() {
    const blocked = ['BINANCE', 'EXCHANGE', 'COINBASE', 'CRYPTO', 'FTX', 'KRAKEN', 'HUOBI', 'BYBIT', 'OKX'];
    const found = [];
    for (const key of _vault.keys()) {
      const upper = key.toUpperCase();
      if (blocked.some(b => upper.includes(b))) {
        found.push(key);
      }
    }
    return found;
  }

  assertNoExchangeKeys() {
    const exchangeKeys = this.getExchangeApiKeys();
    if (exchangeKeys.length > 0) {
      logger.error(`orchestrator: BLOCKED — exchange API keys detected in vault: ${exchangeKeys.join(', ')}`);
      throw new Error(`Exchange API access denied: ${exchangeKeys.join(', ')}`);
    }
    return true;
  }

  wipe() {
    for (const [k, v] of _vault) {
      if (typeof v === 'string') {
        _vault.set(k, _zero(v));
      }
    }
    _vault.clear();
    this._loaded = false;
    this._loadedAt = null;
    logger.info('orchestrator: volatile memory wiped');
  }

  getStatus() {
    return {
      loaded: this._loaded,
      keyCount: _vault.size,
      accessCount: this._accessCount,
      loadedAt: this._loadedAt ? new Date(this._loadedAt).toISOString() : null,
      keyNames: this.listKeys(),
    };
  }
}

export const orchestrator = new SecureOrchestrator();

async function initOrchestrator() {
  try {
    if (!aegis.getStatus().active) {
      await aegis.activate(process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026');
    }
    await orchestrator.loadVault(process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026');

    const ghToken = orchestrator.getGitHubToken();
    if (ghToken && !ghToken.startsWith('[INSERT')) {
      process.env.GITHUB_PAT = ghToken;
    }
    const openRouter = orchestrator.getOpenRouterKey();
    if (openRouter && !openRouter.startsWith('[INSERT')) {
      process.env.OPENROUTER_API_KEY = openRouter;
    }

    const vercelToken = orchestrator.getVercelToken();
    if (vercelToken && !vercelToken.startsWith('[INSERT')) {
      process.env.VERCEL_TOKEN = vercelToken;
    }

    const netlifyToken = orchestrator.getNetlifyToken();
    if (netlifyToken && !netlifyToken.startsWith('[INSERT')) {
      process.env.NETLIFY_TOKEN = netlifyToken;
    }

    const telegramCfg = orchestrator.getTelegramConfig();
    if (telegramCfg) {
      if (telegramCfg.bot_token) process.env.TELEGRAM_BOT_TOKEN = telegramCfg.bot_token;
      if (telegramCfg.channel_id) process.env.TELEGRAM_CHANNEL_ID = telegramCfg.channel_id;
      if (telegramCfg.user_id) process.env.TELEGRAM_USER_ID = telegramCfg.user_id;
    }

    await watchdog.activate(orchestrator);
    orchestrator.assertNoExchangeKeys();
    logger.info('orchestrator: secure init complete — keys in volatile memory, watchdog active');
    return orchestrator.getStatus();
  } catch (e) {
    logger.error(`orchestrator: init failed — ${e.message}`);
    throw e;
  }
}

export { initOrchestrator };
export default orchestrator;
