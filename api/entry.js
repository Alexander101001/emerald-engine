import orchestrator from '../src/core/orchestrator.js';
import subManager from '../src/modules/subscription.js';
import router from '../src/modules/aiRouter.js';
import scrapeTrends from '../src/modules/trendScraper.js';
import generatePage from '../src/modules/pageGenerator.js';
import { analyzeSEO, generateMetaTags, injectSEO } from '../src/modules/seoSuite.js';
import stripe from '../src/monetization/stripe.js';
import affiliateManager from '../src/monetization/affiliate.js';
import logger from '../src/utils/logger.js';
import config from '../src/config.js';

let booted = false;

export default async function handler(req, res) {
  if (!booted) {
    await orchestrator.bootstrap();
    booted = true;
  }

  const { method, url } = req;
  const path = new URL(url, `http://${req.headers.host}`).pathname;

  res.setHeader('Content-Type', 'application/json');

  try {
    if (method === 'POST' && path === '/api/generate') {
      let body = '';
      req.on('data', c => body += c);
      await new Promise(r => req.on('end', r));
      const { topic, apiKey } = JSON.parse(body || '{}');

      if (apiKey) {
        const user = subManager.validateApiKey(apiKey);
        if (!user) return res.status(401).json({ error: 'invalid_api_key' });
        const access = await subManager.checkAccess(user.userId, 'page');
        if (!access.allowed) return res.status(429).json({ error: access.reason });
      }

      const page = generatePage({ title: topic || 'Emerald Page' });
      subManager.trackUsage(apiKey ? subManager.validateApiKey(apiKey).userId : 'anonymous', 'page');

      const metaTags = generateMetaTags({ title: page.title, description: page.title, canonical: `https://emerald.app/${page.slug}` });
      page.html = injectSEO(page.html, metaTags);

      return res.status(200).json(page);
    }

    if (method === 'GET' && path === '/api/trends') {
      const trends = await scrapeTrends(config.TREND_SOURCES);
      return res.status(200).json({ trends: trends.slice(0, 10) });
    }

    if (method === 'POST' && path === '/api/ai') {
      let body = '';
      req.on('data', c => body += c);
      await new Promise(r => req.on('end', r));
      const { prompt } = JSON.parse(body || '{}');
      const result = await router.generate(prompt || 'Give a startup tip', { allowFallbackMock: true });
      return res.status(200).json(result);
    }

    if (method === 'GET' && path === '/api/health') {
      const affiliateStats = affiliateManager.getStats();
      return res.status(200).json({
        status: 'ok',
        platform: config.DEPLOY_PLATFORM,
        uptime: process.uptime(),
        users: subManager.getAllUsers().length,
        affiliate: affiliateStats,
      });
    }

    if (method === 'GET' && path === '/api/pricing') {
      return res.status(200).json({ tiers: stripe.getLocalPrices(), publishableKey: stripe.getPublishableKey() });
    }

    if (method === 'POST' && path === '/api/subscribe') {
      let body = '';
      req.on('data', c => body += c);
      await new Promise(r => req.on('end', r));
      const { userId, priceId } = JSON.parse(body || '{}');
      if (!userId || !priceId) return res.status(400).json({ error: 'userId and priceId required' });
      const session = await subManager.upgradeToStripe(userId, priceId);
      return res.status(200).json(session);
    }

    if (method === 'POST' && path === '/api/stripe-webhook') {
      let body = '';
      req.on('data', c => body += c);
      await new Promise(r => req.on('end', r));
      const sig = req.headers['stripe-signature'];
      const event = await stripe.handleWebhook(body, sig);
      await subManager.handleStripeWebhook(event);
      return res.status(200).json({ received: true });
    }

    if (method === 'GET' && path === '/api/affiliate/stats') {
      return res.status(200).json(affiliateManager.getStats());
    }

    if (method === 'POST' && path === '/api/affiliate/click') {
      let body = '';
      req.on('data', c => body += c);
      await new Promise(r => req.on('end', r));
      const { linkId, userId } = JSON.parse(body || '{}');
      affiliateManager.trackClick(linkId, userId);
      return res.status(200).json({ tracked: true });
    }

    if (method === 'GET' && path.startsWith('/api/seo')) {
      const urlParam = new URL(url, `http://${req.headers.host}`).searchParams.get('url');
      if (!urlParam) return res.status(400).json({ error: 'url query param required' });
      const resp = await fetch(urlParam);
      const html = await resp.text();
      const report = analyzeSEO(html, urlParam);
      return res.status(200).json(report);
    }

    return res.status(404).json({ error: 'not_found' });
  } catch (e) {
    logger.error('api: handler error', e.message);
    return res.status(500).json({ error: e.message });
  }
}
