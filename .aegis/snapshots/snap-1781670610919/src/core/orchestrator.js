import scrapeTrends from '../modules/trendScraper.js';
import generatePage from '../modules/pageGenerator.js';
import { analyzeSEO, generateMetaTags, injectSEO } from '../modules/seoSuite.js';
import subManager from '../modules/subscription.js';
import router from '../modules/aiRouter.js';
import bot from '../modules/telegramBot.js';
import { slugify } from '../utils/helpers.js';
import { SelfImprover } from './selfImprove.js';
import logger from '../utils/logger.js';
import config from '../config.js';

export class Orchestrator {
  constructor() {
    this.improver = new SelfImprover();
    this._intervalIds = [];
  }

  async bootstrap() {
    logger.info(`orchestrator: booting Emerald v1.0 on ${config.DEPLOY_PLATFORM}`);

    this.improver.runLint();

    subManager.registerUser('admin', 'pro');

    bot.startPolling().catch(e => logger.warn(`orchestrator: bot polling deferred - ${e.message}`));

    logger.info('orchestrator: ready');
    return { status: 'ok', platform: config.DEPLOY_PLATFORM };
  }

  async fullPipeline(topic) {
    const prompt = topic || 'micro SaaS landing page';

    logger.info(`orchestrator: pipeline start - "${prompt}"`);

    const trends = await scrapeTrends(config.TREND_SOURCES).catch(() => []);
    const trendTopic = trends[0]?.title || prompt;

    const aiResult = await router.generate(
      `Generate a SaaS page JSON with "title", "tagline", "features" (array), and "cta" for: ${trendTopic}`,
      { allowFallbackMock: true }
    );

    let data;
    try { data = typeof aiResult.content === 'string' ? JSON.parse(aiResult.content) : aiResult.content; }
    catch { data = { title: trendTopic, tagline: 'Emerald generated', features: ['Feature 1', 'Feature 2'], cta: 'Get Started' }; }

    const page = generatePage({ title: data.title || trendTopic }, { adSlots: 3 });

    const metaTags = generateMetaTags({
      title: page.title,
      description: data.tagline || page.title,
      canonical: `https://${slugify(page.title)}.example.com`,
      keywords: `${data.title || ''}, SaaS, micro-saas, emerald`,
    });

    page.html = injectSEO(page.html, metaTags);

    const seoReport = analyzeSEO(page.html);

    logger.info(`orchestrator: pipeline complete - "${page.title}" (SEO: ${seoReport.score}/100)`);

    return { page, trends: trends.slice(0, 3), seo: seoReport, aiProvider: aiResult.provider };
  }

  schedule(fn, intervalMs) {
    const id = setInterval(fn, intervalMs);
    this._intervalIds.push(id);
    return id;
  }

  async shutdown() {
    for (const id of this._intervalIds) clearInterval(id);
    logger.info('orchestrator: shutdown');
  }
}

const orch = new Orchestrator();
export default orch;
