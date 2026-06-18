import fetch from 'node-fetch';
import logger from '../utils/logger.js';
import watchdog from './aegis-watchdog.js';

const POLL_INTERVAL_MS = 5000;
const API_BASE = 'https://api.telegram.org';

class TelegramBridge {
  constructor() {
    this._token = null;
    this._authorizedUserId = null;
    this._channelId = null;
    this._pollTimer = null;
    this._lastUpdateId = 0;
    this._commandHandlers = new Map();
    this._active = false;
    this._messageCount = 0;
  }

  async initialize(token, authorizedUserId, channelId) {
    this._token = token;
    this._authorizedUserId = authorizedUserId;
    this._channelId = channelId;

    if (!token) {
      logger.warn('telegram: bot token not configured — bridge disabled');
      return { active: false, reason: 'no_token' };
    }

    const me = await this._apiCall('getMe');
    if (!me || !me.ok) {
      logger.warn('telegram: bot token invalid — bridge disabled');
      return { active: false, reason: 'invalid_token' };
    }

    logger.info(`telegram: authenticated as @${me.result.username}`);

    this._registerDefaultCommands();
    this._startPolling();

    this._active = true;
    logger.info('telegram: bridge initialized and polling');
    return { active: true, bot: me.result.username };
  }

  _adminCommands() {
    return new Set(['deploy', 'scale', 'heal', 'shutdown', 'exec']);
  }

  _isAdminCommand(command) {
    return this._adminCommands().has(command);
  }

  _isAuthorized(msg) {
    if (!this._authorizedUserId) return false;
    const userId = String(msg.from?.id || msg.chat?.id || '');
    return userId === String(this._authorizedUserId);
  }

  _registerDefaultCommands() {
    this.onCommand('start', async (msg) => {
      await this.sendMessage(msg.chat.id,
        'Emerald AGI — Secure Bridge\n\nCommands:\n/status — system status\n/deploy — run cognitive cycle\n/scale <name> — spawn a repo\n/deployments — list deployed repos\n/heal — run self-heal\n/exec <cmd> — execute shell command\n/shutdown — graceful shutdown');
    });

    this.onCommand('status', async (msg) => {
      const { default: lifecycle } = await import('./lifecycle.js');
      const s = lifecycle.getLifecycleStatus();
      await this.sendMessage(msg.chat.id,
        `Neurons: ${s.neural.totalNeurons}\nRepos: ${s.neural.spawnedRepos}\nFiles: ${s.aegis.trackedFiles}\nCycles: ${s.cycles}\nFailures: ${s.failures}\nVault: ${s.aegis.vault}`);
    });

    this.onCommand('deploy', async (msg) => {
      await this.sendMessage(msg.chat.id, 'Running cognitive cycle...');
      try {
        const { default: neuralCore } = await import('./neural-core.js');
        const result = await neuralCore.cognitiveCycle();
        await this.sendToChannel(
          `Cycle ${result.cycle} complete\nConfidence: ${result.avgConfidence}\nRisk: ${result.risk}\nSpawned: ${result.spawned}\nTotal repos: ${result.reposTotal}`);
      } catch (e) {
        await this.sendToChannel(`Deploy failed: ${e.message}`);
      }
    });

    this.onCommand('scale', async (msg, args) => {
      const name = args.join(' ');
      if (!name) return this.sendMessage(msg.chat.id, 'Usage: /scale <product name>');
      const { default: lifecycle } = await import('./lifecycle.js');
      const repo = await lifecycle.scaleUp({ productName: name, tagline: 'Telegram-spawned', keyFeatures: ['Auto-deployed'], monetization: 'freemium' });
      await this.sendMessage(msg.chat.id, repo ? `Spawned: ${repo.url}` : 'Scale-up failed');
    });

    this.onCommand('heal', async (msg) => {
      const { default: lifecycle } = await import('./lifecycle.js');
      const result = await lifecycle.selfHeal();
      await this.sendToChannel(
        `Self-heal complete\n  Files before: ${result.before}\n  Files after: ${result.after}\n  Repairs: ${result.repairs}\n  Cycles: ${result.cycles}`);
    });

    this.onCommand('exec', async (msg, args) => {
      const cmd = args.join(' ');
      if (!cmd) return this.sendMessage(msg.chat.id, 'Usage: /exec <command>');
      await this.sendMessage(msg.chat.id, `Executing: ${cmd.slice(0, 100)}`);
      try {
        const { execSync } = await import('child_process');
        const output = execSync(cmd, { timeout: 30000, encoding: 'utf-8', maxBuffer: 4096 });
        const safe = output.slice(0, 2000);
        await this.sendMessage(msg.chat.id, `Exit 0\n${safe || '(empty output)'}`);
      } catch (e) {
        const safeErr = (e.stdout || e.message || '').slice(0, 1500);
        await this.sendMessage(msg.chat.id, `Exit ${e.status || 1}\n${safeErr}`);
      }
    });

    this.onCommand('deployments', async (msg) => {
      const { default: lifecycle } = await import('./lifecycle.js');
      const s = lifecycle.getLifecycleStatus();
      await this.sendMessage(msg.chat.id,
        `Total repos: ${s.neural.spawnedRepos}\nCycles: ${s.cycles}\nVault loaded: ${s.vaultLoaded}`);
    });

    this.onCommand('shutdown', async (msg) => {
      await this.sendToChannel('Shutting down Emerald AGI...');
      const { default: lifecycle } = await import('./lifecycle.js');
      lifecycle.shutdown();
      process.exit(0);
    });
  }

  onCommand(command, handler) {
    this._commandHandlers.set(command.toLowerCase(), handler);
  }

  _startPolling() {
    if (this._pollTimer) clearInterval(this._pollTimer);
    this._pollTimer = setInterval(() => this._poll(), POLL_INTERVAL_MS);
  }

  async _apiCall(method) {
    try {
      const res = await watchdog.secureFetch(
        `${API_BASE}/bot${this._token}/${method}`,
        { method: 'GET' }
      );
      return res.json();
    } catch (e) {
      logger.warn(`telegram: _apiCall(${method}) failed — ${e.message}`);
      return null;
    }
  }

  async _poll() {
    try {
      const params = { offset: this._lastUpdateId + 1, timeout: 10 };
      const res = await watchdog.secureFetch(
        `${API_BASE}/bot${this._token}/getUpdates`,
        { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(params) }
      );
      const data = await res.json();
      if (!data.ok) return;

      for (const update of data.result || []) {
        this._lastUpdateId = update.update_id;
        const msg = update.message;
        if (!msg || !msg.text) continue;
        this._messageCount++;
        await this._handleMessage(msg);
      }
    } catch (e) {
      if (!e.message.includes('Watchdog blocked')) {
        logger.warn(`telegram: poll error — ${e.message}`);
      }
    }
  }

  async _handleMessage(msg) {
    const text = msg.text.trim();
    if (!text.startsWith('/')) return;

    const parts = text.split(/\s+/);
    const command = parts[0].toLowerCase().replace('/', '');
    const args = parts.slice(1);

    if (this._isAdminCommand(command) && !this._isAuthorized(msg)) {
      await this.sendMessage(msg.chat.id, 'Unauthorized: admin privileges required.');
      return;
    }

    const handler = this._commandHandlers.get(command);
    if (handler) {
      try {
        await handler(msg, args);
      } catch (e) {
        logger.error(`telegram: command /${command} failed — ${e.message}`);
        if (this._isAuthorized(msg)) {
          await this.sendMessage(msg.chat.id, `Error: ${e.message}`);
        }
      }
    }
  }

  async sendMessage(chatId, text) {
    if (!this._token) return null;
    const target = chatId || this._channelId || this._authorizedUserId;
    if (!target) return null;
    try {
      const res = await watchdog.secureFetch(
        `${API_BASE}/bot${this._token}/sendMessage`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ chat_id: target, text: text.slice(0, 4096) }),
        }
      );
      return res.json();
    } catch (e) {
      if (!e.message.includes('Watchdog blocked')) {
        logger.warn(`telegram: sendMessage failed — ${e.message}`);
      }
      return null;
    }
  }

  async sendToChannel(text) {
    return this.sendMessage(this._channelId, text);
  }

  async sendToOwner(text) {
    return this.sendMessage(this._authorizedUserId, text);
  }

  async sendDeploymentSuccess(platform, url, repoName) {
    const text = [
      `Deployment Successful — ${platform}`,
      ``,
      `Repo: ${repoName}`,
      `Platform: ${platform}`,
      `Live URL: ${url}`,
      ``,
      `Emerald AGI — Autonomous Deployment`,
    ].join('\n');
    return this.sendToChannel(text);
  }

  async verifyChannel(channelId) {
    const target = channelId || this._channelId;
    if (!target) return { ok: false, reason: 'no_channel' };
    try {
      const res = await watchdog.secureFetch(
        `${API_BASE}/bot${this._token}/sendMessage`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ chat_id: target, text: 'Channel verification ping', disable_notification: true }),
        }
      );
      const data = await res.json();
      return { ok: data.ok, chatId: data.result?.chat?.id || target };
    } catch (e) {
      return { ok: false, reason: e.message };
    }
  }

  shutdown() {
    if (this._pollTimer) {
      clearInterval(this._pollTimer);
      this._pollTimer = null;
    }
    this._active = false;
    logger.info('telegram: bridge shut down');
  }

  reset() {
    this.shutdown();
    this._token = null;
    this._authorizedUserId = null;
    this._channelId = null;
    this._lastUpdateId = 0;
    this._commandHandlers.clear();
    this._messageCount = 0;
    logger.info('telegram: bridge reset — cache cleared');
  }

  getStatus() {
    return {
      active: this._active,
      messagesProcessed: this._messageCount,
      authorizedUser: this._authorizedUserId ? `${this._authorizedUserId.slice(0, 4)}...` : null,
      channelId: this._channelId || null,
      polling: !!this._pollTimer,
    };
  }
}

const telegram = new TelegramBridge();
export default telegram;
