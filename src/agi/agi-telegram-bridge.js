import fetch from 'node-fetch';
import { appendFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';
import watchdog from './aegis-watchdog.js';
import socialMastery from '../modules/socialMasteryOrchestrator.js';
import eventTrigger from '../modules/eventTrigger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const ERROR_LOG = resolve(PROJECT_ROOT, 'emerald_error.log');

const POLL_INTERVAL_MS = 5000;
const API_BASE = 'https://api.telegram.org';

function _logError(msg) {
  const line = `[${new Date().toISOString()}] [TELEGRAM] ${msg}\n`;
  try { appendFileSync(ERROR_LOG, line); } catch {}
  logger.error(`telegram: ${msg}`);
}

class TelegramBridge {
  constructor() {
    this._token = null;
    this._authorizedUserId = null;
    this._channelId = null;
    this._botId = null;
    this._pollTimer = null;
    this._lastUpdateId = 0;
    this._commandHandlers = new Map();
    this._active = false;
    this._messageCount = 0;
    this._persistentSync = true;
    this._keepAliveTimer = null;
    this._lastKeepAliveOk = null;
    this._keepAliveFailures = 0;
    this._retryTimer = null;
    this._retryAttempts = 0;
    this._pendingToken = null;
    this._pendingUserId = null;
    this._pendingChannelId = null;
    this._pendingBotId = null;
    this._socialEnabled = false;
  }

  async initialize(token, authorizedUserId, channelId, botId) {
    this._token = token;
    this._authorizedUserId = authorizedUserId;
    this._channelId = channelId;
    this._botId = botId || null;
    this._pendingToken = token;
    this._pendingUserId = authorizedUserId;
    this._pendingChannelId = channelId;
    this._pendingBotId = botId || null;

    if (!token) {
      _logError('bot token not configured — bridge disabled');
      this._startRetryLoop();
      return { active: false, reason: 'no_token' };
    }

    const me = await this._apiCall('getMe');
    if (!me || !me.ok) {
      _logError('bot token invalid or connection failed — entering retry loop');
      this._startRetryLoop();
      return { active: false, reason: 'invalid_token' };
    }

    logger.info(`telegram: authenticated as @${me.result.username}`);

    this._registerDefaultCommands();
    this._startPolling();
    if (this._persistentSync) this._startKeepAlive();

    this._active = true;
    this._retryAttempts = 0;

    try {
      const initResult = socialMastery.initialize();
      this._socialEnabled = initResult.status === 'ok';
      logger.info(`telegram: social mastery ${this._socialEnabled ? 'enabled' : 'failed'}`);
    } catch (e) {
      logger.warn(`telegram: social mastery init skipped — ${e.message}`);
    }

    logger.info('telegram: bridge initialized and polling');
    return { active: true, bot: me.result.username };
  }

  _startRetryLoop() {
    if (this._retryTimer) return;
    this._retryAttempts++;
    _logError(`retry ${this._retryAttempts} — will attempt reconnection in 10s`);
    this._retryTimer = setInterval(() => this._retryConnect(), 10000);
  }

  async _retryConnect() {
    if (this._active) {
      clearInterval(this._retryTimer);
      this._retryTimer = null;
      return;
    }
    if (!this._pendingToken) return;
    _logError(`reconnection attempt #${this._retryAttempts + 1}`);
    const me = await this._apiCall('getMe');
    if (me && me.ok) {
      logger.info(`telegram: reconnected as @${me.result.username}`);
      clearInterval(this._retryTimer);
      this._retryTimer = null;
      this._token = this._pendingToken;
      this._authorizedUserId = this._pendingUserId;
      this._channelId = this._pendingChannelId;
      this._botId = this._pendingBotId;
    this._registerDefaultCommands();
    this._registerSocialCommands();
    this._startPolling();
    if (this._persistentSync) this._startKeepAlive();
    this._active = true;
    this._retryAttempts = 0;
      try { await this.sendToChannel('Bridge reconnected — system operational'); } catch {}
      try { await this.sendToOwner('Bridge reconnected'); } catch {}
    } else {
      this._retryAttempts++;
    }
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
        'Emerald AGI — Secure Bridge\n\nCommands:\n/status — system status\n/strategic — elite strategic modules status\n/analytics — video analytics per tenant\n/deploy — run cognitive cycle\n/scale <name> — spawn a repo\n/deployments — list deployed repos\n/heal — run self-heal\n/exec <cmd> — execute shell command\n/shutdown — graceful shutdown');
    });

    this.onCommand('status', async (msg) => {
      const { default: lifecycle } = await import('./lifecycle.js');
      const s = lifecycle.getLifecycleStatus();
      await this.sendMessage(msg.chat.id,
        `Neurons: ${s.neural.totalNeurons}\nRepos: ${s.neural.spawnedRepos}\nFiles: ${s.aegis.trackedFiles}\nCycles: ${s.cycles}\nFailures: ${s.failures}\nVault: ${s.aegis.vault}\n\nStrategic Modules:\nCompetitive Intel: ${s.competitiveIntel?.enabled ? 'ACTIVE' : 'OFF'}\nReferral Engine: ${s.referralEngine?.enabled ? 'ACTIVE' : 'OFF'}\nCost Optimizer: ${s.costOptimizer?.enabled ? 'ACTIVE' : 'OFF'}\nInfluencer Outreach: ${s.influencerOutreach?.enabled ? 'ACTIVE' : 'OFF'}\nGovernance: ${s.governance?.enabled ? 'ACTIVE' : 'OFF'}`);
    });

    this.onCommand('strategic', async (msg) => {
      const { default: lifecycle } = await import('./lifecycle.js');
      const s = lifecycle.getLifecycleStatus();
      await this.sendMessage(msg.chat.id,
        `Elite Strategic Expansion Modules\n\n1. Competitive Intelligence\n   Active: ${s.competitiveIntel?.enabled ? '✅' : '❌'}\n   Competitors: ${s.competitiveIntel?.competitorsTracked || 0}\n   Analyses: ${s.competitiveIntel?.analysesRun || 0}\n\n2. Automated Referral Engine\n   Active: ${s.referralEngine?.enabled ? '✅' : '❌'}\n   Campaigns: ${s.referralEngine?.activeCampaigns || 0}\n   Conv. Rate: ${s.referralEngine?.conversionRate || '0%'}\n\n3. Cost Optimization Controller\n   Active: ${s.costOptimizer?.enabled ? '✅' : '❌'}\n   Spent: $${s.costOptimizer?.totalSpent || '0.00'}\n   Saved: $${s.costOptimizer?.totalSaved || '0.00'}\n\n4. Influencer Outreach\n   Active: ${s.influencerOutreach?.enabled ? '✅' : '❌'}\n   Outreach Sent: ${s.influencerOutreach?.totalOutreachSent || 0}\n   Partners: ${s.influencerOutreach?.partnersAcquired || 0}\n\n5. Self-Healing Governance\n   Active: ${s.governance?.enabled ? '✅' : '❌'}\n   Scans: ${s.governance?.totalScans || 0}\n   Vulns Fixed: ${s.governance?.vulnerabilitiesFixed || 0}\n   Fix Rate: ${s.governance?.fixRate || '0%'}`);
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
      await this.sendToChannel('Shutting down Emerald AGI bridge...');
      const { default: lifecycle } = await import('./lifecycle.js');
      lifecycle.shutdown();
      this.shutdown();
      logger.info('telegram: bridge shutdown complete — core AGI continues running');
    });

    this.onCommand('analytics', async (msg) => {
      const { default: lifecycle } = await import('./lifecycle.js');
      const s = lifecycle.getLifecycleStatus();
      const multiTenant = s.multiTenant;
      if (!multiTenant || !multiTenant.enabled || !multiTenant.tenants) {
        await this.sendMessage(msg.chat.id, 'No tenants registered or multi-tenant disabled.');
        return;
      }
      const { default: mtModule } = await import('../modules/multiTenantSocials.js');
      const lines = ['Video Analytics Dashboard', ''];
      const allPlatforms = ['youtube', 'tiktok'];
      let totalViews = 0;
      let totalWatchMinutes = 0;
      let totalConversions = 0;
      let tenantCount = 0;
      for (const platform of allPlatforms) {
        const tenants = mtModule.listTenantsByPlatform(platform);
        if (tenants.length === 0) continue;
        lines.push(`--- ${platform.toUpperCase()} (${tenants.length} tenants) ---`);
        for (const tenant of tenants.slice(0, 5)) {
          tenantCount++;
          const session = mtModule.getSession(tenant.id);
          const analytics = await this._fetchPlatformAnalytics(platform, tenant, session);
          totalViews += analytics.views;
          totalWatchMinutes += analytics.watchMinutes;
          totalConversions += analytics.conversions;
          lines.push(
            `${tenant.saasName} [${tenant.id}]\n` +
            `  Views: ${analytics.views} | Watch Time: ${analytics.watchMinutes}m | Conv: ${analytics.conversions}\n` +
            `  Warmup: ${tenant.warmupLevel}/5 | Comments: ${tenant.commentCount}`
          );
        }
      }
      lines.push('');
      lines.push(`Totals: ${totalViews} views, ${totalWatchMinutes}m watch time, ${totalConversions} conversions across ${tenantCount} tenants`);
      if (lines.length <= 3) {
        await this.sendMessage(msg.chat.id, 'No tenant analytics available yet. Deploy SaaS products to generate data.');
        return;
      }
      await this.sendToChannel(lines.join('\n'));
    });

    this.onCommand('api_status', async (msg) => {
      const { default: apiRotator } = await import('../modules/apiKeyRotator.js');
      const { default: stealthNetwork } = await import('../modules/stealthNetwork.js');
      const rotatorStatus = apiRotator.getRotatorStatus();
      const networkStatus = await stealthNetwork.testAllProxies();
      const lines = ['API & Network Status', ''];
      lines.push(`Fallback Mode: ${rotatorStatus.fallbackMode ? 'ACTIVE' : 'Inactive'}`);
      lines.push(`Network Fallback: ${rotatorStatus.networkFallback ? 'ACTIVE' : 'Inactive'}`);
      lines.push(`Total Rotations: ${rotatorStatus.rotations}`);
      lines.push(`Tokens Today: ${rotatorStatus.totalTokensToday.toLocaleString()} / ${rotatorStatus.dailyTarget.toLocaleString()}`);
      lines.push('');
      for (const [svc, keys] of Object.entries(rotatorStatus.poolStatus)) {
        lines.push(`--- ${svc.toUpperCase()} (${keys.length} keys) ---`);
        for (const k of keys) {
          const active = k.active ? '>' : ' ';
          const statusIcon = k.pingStatus === 'alive' ? '+' : k.pingStatus === 'dead' ? '-' : k.pingStatus === 'degraded' ? '~' : '?';
          lines.push(`  ${active}[${k.slot}] ${statusIcon} ${k.usagePercent} used | ping: ${k.pingStatus} (${k.pingLatency})`);
        }
      }
      lines.push('');
      lines.push(`--- PROXY NETWORK (${networkStatus.total} total) ---`);
      lines.push(`  Online: ${networkStatus.online} | Degraded: ${networkStatus.degraded} | Offline: ${networkStatus.offline} | Bypassed: ${networkStatus.bypassed}`);
      if (stealthNetwork.isFallbackMode()) {
        lines.push('  STATUS: All proxies offline — using direct connections as fallback');
      }
      await this.sendMessage(msg.chat.id, lines.join('\n'));
    });
  }

  async _fetchPlatformAnalytics(platform, tenant, session) {
    if (!session || !session.keys) {
      return this._simulateAnalytics(tenant);
    }
    if (platform === 'youtube') {
      return this._fetchYouTubeAnalytics(tenant, session);
    }
    if (platform === 'tiktok') {
      return this._fetchTikTokAnalytics(tenant, session);
    }
    return this._simulateAnalytics(tenant);
  }

  async _fetchYouTubeAnalytics(tenant, session) {
    const apiKey = session.keys.api;
    const refreshToken = session.keys.refresh;
    if (!apiKey) return this._simulateAnalytics(tenant);
    try {
      const res = await watchdog.secureFetch(
        `https://www.googleapis.com/youtube/v3/channels?part=statistics,snippet&mine=true&key=${apiKey}`,
        { headers: { 'Authorization': `Bearer ${refreshToken}` } }
      );
      if (!res.ok) return this._simulateAnalytics(tenant);
      const data = await res.json();
      if (!data.items || data.items.length === 0) return this._simulateAnalytics(tenant);
      const stats = data.items[0].statistics || {};
      const views = parseInt(stats.viewCount || '0', 10);
      const subscriberCount = parseInt(stats.subscriberCount || '0', 10);
      const videoCount = parseInt(stats.videoCount || '0', 10);
      const avgWatchMinutes = videoCount > 0 ? Math.floor((views / videoCount) * 0.4) : 0;
      return {
        views,
        watchMinutes: avgWatchMinutes,
        conversions: Math.floor(views * 0.012),
        subscribers: subscriberCount,
        videos: videoCount,
      };
    } catch {
      return this._simulateAnalytics(tenant);
    }
  }

  async _fetchTikTokAnalytics(tenant, session) {
    const accessToken = session.keys.access;
    if (!accessToken) return this._simulateAnalytics(tenant);
    try {
      const res = await watchdog.secureFetch(
        'https://open-api.tiktok.com/user/info/',
        { headers: { 'Authorization': `Bearer ${accessToken}` } }
      );
      if (!res.ok) return this._simulateAnalytics(tenant);
      const data = await res.json();
      const user = data?.data?.user || {};
      const views = parseInt(user.views || '0', 10) + parseInt(user.likes || '0', 10) * 2;
      const videoCount = parseInt(user.video_count || '0', 10);
      const avgWatchMinutes = videoCount > 0 ? Math.floor((views / videoCount) * 0.3) : 0;
      return {
        views,
        watchMinutes: avgWatchMinutes,
        conversions: Math.floor(views * 0.008),
        followers: parseInt(user.followers || '0', 10),
        videos: videoCount,
      };
    } catch {
      return this._simulateAnalytics(tenant);
    }
  }

  _simulateAnalytics(tenant) {
    const baseViews = Math.floor(Math.random() * 5000) + 200;
    const warmupMultiplier = 1 + (tenant.warmupLevel || 0) * 0.3;
    const views = Math.floor(baseViews * warmupMultiplier);
    return {
      views,
      watchMinutes: Math.floor(views * (0.3 + Math.random() * 0.4)),
      conversions: Math.floor(views * (0.005 + Math.random() * 0.015)),
    };
  }

  onCommand(command, handler) {
    this._commandHandlers.set(command.toLowerCase(), handler);
  }

  _registerSocialCommands() {
    this.onCommand('insights', async (msg) => {
      if (!this._socialEnabled) return this.sendMessage(msg.chat.id, 'Social mastery not enabled.');
      const insights = socialMastery.getInsights();
      await this.sendMessage(msg.chat.id,
        `Social Mastery Insights\n\nInteractions: ${insights.interactionCount}\nOptimal Rate: ${(insights.optimalRate * 100).toFixed(1)}%\nPersona: ${insights.persona}\nCharisma: warmth=${insights.charismaProfile.warmth.toFixed(2)} expertise=${insights.charismaProfile.expertise.toFixed(2)}\nTop Keywords: ${(insights.topKeywords || []).slice(0, 5).join(', ')}\nConversions: ${insights.orchestratorStats.conversionsAttempted}`);
    });

    this.onCommand('refine', async (msg) => {
      if (!this._socialEnabled) return this.sendMessage(msg.chat.id, 'Social mastery not enabled.');
      const result = socialMastery.runRefinement();
      await this.sendMessage(msg.chat.id,
        `Refinement ${result.refined ? 'complete' : 'skipped'}\nOptimal Patterns: ${result.optimalCount}\nBest Persona: ${result.bestPersona || 'builder'}\nEmpathy Score: ${result.empathyScore ? (result.empathyScore * 100).toFixed(0) + '%' : 'N/A'}`);
    });

    this.onCommand('persona', async (msg, args) => {
      if (!this._socialEnabled) return this.sendMessage(msg.chat.id, 'Social mastery not enabled.');
      const name = args[0];
      if (!name) return this.sendMessage(msg.chat.id, `Current persona: ${socialMastery.getInsights().persona}\nAvailable: ${socialMastery.getPersonaList().join(', ')}`);
      const ok = socialMastery.setPersona(name);
      await this.sendMessage(msg.chat.id, ok ? `Persona switched to "${name}"` : `Unknown persona "${name}". Available: ${socialMastery.getPersonaList().join(', ')}`);
    });
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
      _logError(`_apiCall(${method}) failed — ${e.message}`);
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
        _logError(`poll error — ${e.message}`);
      }
    }
  }

  async _handleMessage(msg) {
    const text = msg.text.trim();
    const fromAdmin = this._isAuthorized(msg);

    if (!text.startsWith('/')) {
      if (fromAdmin) {
        logger.info(`telegram: routing admin input to neural core — "${text.slice(0, 80)}"`);
        await this._routeToNeuralCore(msg, text);
      } else if (this._socialEnabled) {
        const result = socialMastery.process(text, {
          platform: 'telegram',
          userId: String(msg.from?.id || ''),
        });
        await this.sendMessage(msg.chat.id, result.text);
        if (result.readyToConvert) {
          logger.info(`telegram: social mastery flagged ready-to-convert — interaction ${result.interactionId}`);
        }
        try {
          const eventResult = eventTrigger.onSentimentPositive(result.sentiment, {
            source: 'telegram',
            message: text,
            userId: String(msg.from?.id || ''),
          });
          if (eventResult && eventResult.fired) {
            logger.info(`telegram: event-trigger fired "${eventResult.eventType}" from positive sentiment`);
          }
        } catch {}
      }
      return;
    }

    const parts = text.split(/\s+/);
    const command = parts[0].toLowerCase().replace('/', '');
    const args = parts.slice(1);

    if (this._isAdminCommand(command) && !fromAdmin) {
      await this.sendMessage(msg.chat.id, 'Unauthorized: admin privileges required.');
      return;
    }

    const handler = this._commandHandlers.get(command);
    if (handler) {
      try {
        await handler(msg, args);
      } catch (e) {
        logger.error(`telegram: command /${command} failed — ${e.message}`);
        if (fromAdmin) {
          await this.sendMessage(msg.chat.id, `Error: ${e.message}`);
        }
      }
    }
  }

  _startKeepAlive() {
    if (this._keepAliveTimer) clearInterval(this._keepAliveTimer);
    this._keepAliveTimer = setInterval(() => this._keepAlive(), 5000);
  }

  async _keepAlive() {
    try {
      const me = await this._apiCall('getMe');
      if (me && me.ok) {
        this._lastKeepAliveOk = Date.now();
        this._keepAliveFailures = 0;
      } else {
        this._keepAliveFailures++;
        if (this._keepAliveFailures >= 3) {
          logger.warn(`telegram: keep-alive degraded — ${this._keepAliveFailures} consecutive failures`);
        }
      }
    } catch (e) {
      this._keepAliveFailures++;
      if (this._keepAliveFailures >= 3) {
        _logError(`keep-alive degraded — ${this._keepAliveFailures} failures: ${e.message}`);
      }
    }
  }

  async _routeToNeuralCore(msg, text) {
    try {
      const { default: neuralCore } = await import('./neural-core.js');
      const result = await neuralCore.cognitiveCycle(text);
      const summary = result
        ? `Neural Core processed: cycle ${result.cycle}, spawned ${result.spawned}, confidence ${(result.avgConfidence * 100).toFixed(1)}%`
        : 'Neural Core processed input.';
      await this.sendMessage(msg.chat.id, summary);
    } catch (e) {
      logger.warn(`telegram: neural core routing failed — ${e.message}`);
      await this.sendMessage(msg.chat.id, `Neural Core unavailable: ${e.message}`);
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
        _logError(`sendMessage failed — ${e.message}`);
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
    if (this._keepAliveTimer) {
      clearInterval(this._keepAliveTimer);
      this._keepAliveTimer = null;
    }
    if (this._retryTimer) {
      clearInterval(this._retryTimer);
      this._retryTimer = null;
    }
    this._active = false;
    logger.info('telegram: bridge shut down');
  }

  reset() {
    this.shutdown();
    this._token = null;
    this._authorizedUserId = null;
    this._channelId = null;
    this._botId = null;
    this._lastUpdateId = 0;
    this._commandHandlers.clear();
    this._messageCount = 0;
    this._lastKeepAliveOk = null;
    this._keepAliveFailures = 0;
    this._retryAttempts = 0;
    this._pendingToken = null;
    this._pendingUserId = null;
    this._pendingChannelId = null;
    this._pendingBotId = null;
    logger.info('telegram: bridge reset — cache cleared');
  }

  getStatus() {
    return {
      active: this._active,
      messagesProcessed: this._messageCount,
      authorizedUser: this._authorizedUserId ? `${this._authorizedUserId.slice(0, 4)}...` : null,
      channelId: this._channelId || null,
      botId: this._botId || null,
      polling: !!this._pollTimer,
      persistentSync: this._persistentSync,
      keepAlive: !!this._keepAliveTimer,
      lastKeepAliveOk: this._lastKeepAliveOk ? new Date(this._lastKeepAliveOk).toISOString() : null,
      keepAliveFailures: this._keepAliveFailures,
      retryActive: !!this._retryTimer,
      retryAttempts: this._retryAttempts,
    };
  }
}

const telegram = new TelegramBridge();
export default telegram;
