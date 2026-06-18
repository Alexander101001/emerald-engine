import fetch from 'node-fetch';
import logger from '../utils/logger.js';
import config from '../config.js';
import scrapeTrends from './trendScraper.js';
import generatePage from './pageGenerator.js';
import router from './aiRouter.js';
import { analyzeSEO } from './seoSuite.js';
import subManager from './subscription.js';

export class TelegramBot {
  constructor() {
    this.token = config.TELEGRAM_BOT_TOKEN;
    this.base = `https://api.telegram.org/bot${this.token}`;
    this._offset = 0;
    this._handlers = new Map();
    this._registerCommands();
  }

  _registerCommands() {
    this.on('/start', async (chatId) => {
      await this.send(chatId,
        `🤖 *Emerald Bot Online*\n\n` +
        `/trends - Fetch latest SaaS trends\n` +
        `/generate <topic> - Generate a landing page\n` +
        `/seo <url> - Analyze page SEO\n` +
        `/improve <prompt> - AI-powered suggestion\n` +
        `/status - System health\n` +
        `/register - Get your free API key`
      );
    });

    this.on('/trends', async (chatId) => {
      await this.send(chatId, '🔍 Fetching trends...');
      const trends = await scrapeTrends(config.TREND_SOURCES);
      const msg = trends.slice(0, 5).map((t, i) =>
        `${i + 1}. *${t.title}* (${t.source})\n[Link](${t.link})`
      ).join('\n\n');
      await this.send(chatId, `📈 *Top Trends*\n\n${msg}`, { parse_mode: 'Markdown' });
    });

    this.on('/generate', async (chatId, args) => {
      const topic = args.join(' ') || 'AI-powered SaaS';
      await this.send(chatId, `⚙️ Generating landing page for "${topic}"...`);
      const aiResult = await router.generate(`Create a SaaS landing page outline for: ${topic}`, { allowFallbackMock: true });
      let trendData;
      try { trendData = typeof aiResult.content === 'string' ? JSON.parse(aiResult.content) : aiResult.content; }
      catch { trendData = { title: topic, tagline: `The best ${topic} platform.` }; }
      const page = generatePage({ title: trendData.title || topic }, { adSlots: 3 });
      await this.send(chatId,
        `✅ *Page Generated!*\n\nTitle: ${page.title}\nSlug: ${page.slug}\nAds: ${page.adSlots.length} slots embedded\n\nHTML: \`${page.html.slice(0, 200)}...\``,
        { parse_mode: 'Markdown' }
      );
    });

    this.on('/seo', async (chatId, args) => {
      const url = args[0];
      if (!url) { await this.send(chatId, 'Usage: /seo <url>'); return; }
      await this.send(chatId, `🔎 Analyzing ${url}...`);
      const res = await fetch(url);
      const html = await res.text();
      const report = analyzeSEO(html, url);
      const msg =
        `📊 *SEO Report*\n\nScore: ${report.score}/100\n\n` +
        `*Issues:*\n${report.issues.map(i => `❌ ${i}`).join('\n') || 'None'}\n\n` +
        `*Suggestions:*\n${report.suggestions.map(s => `💡 ${s}`).join('\n') || 'None'}`;
      await this.send(chatId, msg, { parse_mode: 'Markdown' });
    });

    this.on('/improve', async (chatId, args) => {
      const prompt = args.join(' ');
      if (!prompt) { await this.send(chatId, 'Usage: /improve <question or prompt>'); return; }
      await this.send(chatId, `🧠 Thinking...`);
      const result = await router.generate(prompt, { allowFallbackMock: true });
      await this.send(chatId, `*Emerald AI:* ${result.content}`, { parse_mode: 'Markdown' });
    });

    this.on('/status', async (chatId) => {
      const registered = subManager._store.size;
      const trends = (await scrapeTrends(config.TREND_SOURCES)).length;
      await this.send(chatId,
        `🟢 *Emerald Status*\\n\\n` +
        `Engine: \`online\`\n` +
        `Trends cached: ${trends}\n` +
        `Users registered: ${registered}\n` +
        `AI provider: \`${router._fallbackProviders.length > 0 ? router._fallbackProviders[0].name : 'mock'}\`\n` +
        `Platform: ${config.DEPLOY_PLATFORM}`
      );
    });

    this.on('/register', async (chatId) => {
      const user = subManager.getUser(String(chatId));
      if (user) {
        await this.send(chatId, `You already have an API key: \`${user.apiKey}\``);
        return;
      }
      const record = subManager.registerUser(String(chatId));
      await this.send(chatId, `✅ Registered on *free* tier!\nAPI Key: \`${record.apiKey}\`\nLimit: ${record.limits.pages} pages/day`, { parse_mode: 'Markdown' });
    });

    this.on('/help', async (chatId) => {
      await this.send(chatId, 'Available commands:\n/start - Intro\n/trends - Top trends\n/generate <topic> - Generate page\n/seo <url> - SEO audit\n/improve <prompt> - Ask AI\n/status - System health\n/register - Get API key');
    });
  }

  on(command, handler) {
    this._handlers.set(command, handler);
  }

  async send(chatId, text, opts = {}) {
    const body = new URLSearchParams({ chat_id: chatId, text });
    if (opts.parse_mode) body.set('parse_mode', opts.parse_mode);
    try {
      const res = await fetch(`${this.base}/sendMessage`, { method: 'POST', body });
      if (!res.ok) logger.warn(`telegramBot: send failed ${await res.text()}`);
    } catch (e) {
      logger.error(`telegramBot: send error - ${e.message}`);
    }
  }

  async poll() {
    if (!this.token) { logger.warn('telegramBot: no token configured'); return; }
    try {
      const res = await fetch(`${this.base}/getUpdates`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ offset: this._offset, timeout: 30 }),
      });
      if (!res.ok) return;
      const data = await res.json();
      if (!data.ok || !data.result) return;

      for (const update of data.result) {
        this._offset = update.update_id + 1;
        const msg = update.message;
        if (!msg?.text) continue;
        const chatId = msg.chat.id;
        const [cmd, ...args] = msg.text.trim().split(/\s+/);
        const handler = this._handlers.get(cmd);
        if (handler) {
          handler(chatId, args).catch(e => logger.error(`telegramBot: handler error - ${e.message}`));
        }
      }
    } catch (e) {
      logger.error(`telegramBot: poll error - ${e.message}`);
    }
  }

  async startPolling(intervalMs = 2000) {
    if (!this.token) {
      logger.warn('telegramBot: no token — polling disabled');
      return;
    }
    logger.info('telegramBot: starting poll loop');
    const loop = async () => {
      await this.poll();
      setTimeout(loop, intervalMs);
    };
    loop();
  }
}

const bot = new TelegramBot();
export default bot;
