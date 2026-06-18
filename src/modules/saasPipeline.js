import { slugify, uid } from '../utils/helpers.js';
import logger from '../utils/logger.js';
import typingSimulator from './typingSimulator.js';
import localeEngine from './localeEngine.js';

const HIGH_DOMAIN_SITES = [
  { name: 'reddit', subreddits: ['r/SaaS', 'r/startups', 'r/Entrepreneur', 'r/WebDev', 'r/programming', 'r/technology', 'r/business', 'r/coding', 'r/devops', 'r/Productivity'] },
  { name: 'quora', topics: ['SaaS', 'Entrepreneurship', 'Startups', 'Programming', 'Web Development', 'Technology', 'Business Strategy', 'Marketing', 'Growth Hacking', 'Product Management'] },
  { name: 'medium', tags: ['saas', 'startup', 'technology', 'programming', 'productivity', 'business', 'entrepreneurship', 'webdev', 'devops', 'marketing'] },
  { name: 'devto', tags: ['saas', 'startup', 'webdev', 'programming', 'productivity', 'devops', 'javascript', 'python', 'react', 'tutorial'] },
];

const PUBLISH_FORMATS = {
  reddit: {
    type: 'self',
    titleMax: 300,
    bodyMax: 40000,
    format: (title, body) => ({ title: title.slice(0, 300), selftext: body.slice(0, 40000) }),
  },
  quora: {
    type: 'answer',
    bodyMax: 15000,
    format: (title, body) => ({ question: title, answer: body.slice(0, 15000) }),
  },
  medium: {
    type: 'article',
    format: (title, body, tags) => ({ title, contentFormat: 'markdown', content: body, tags: tags.slice(0, 5), publishStatus: 'public' }),
  },
  devto: {
    type: 'article',
    format: (title, body, tags) => ({ title, published: true, body_markdown: body, tags: tags.slice(0, 4), description: title }),
  },
};

const REGION_ACTIVITY_PEAKS = {
  US: { morning: [8, 11], afternoon: [14, 17], evening: [19, 22] },
  GB: { morning: [7, 10], afternoon: [13, 16], evening: [18, 21] },
  AU: { morning: [6, 9], afternoon: [12, 15], evening: [17, 20] },
  IN: { morning: [7, 10], afternoon: [13, 16], evening: [18, 21] },
  SG: { morning: [8, 11], afternoon: [14, 17], evening: [19, 22] },
  DE: { morning: [7, 10], afternoon: [13, 16], evening: [18, 21] },
  FR: { morning: [8, 11], afternoon: [14, 17], evening: [19, 22] },
  JP: { morning: [6, 9], afternoon: [12, 15], evening: [17, 20] },
  BR: { morning: [8, 11], afternoon: [14, 17], evening: [19, 22] },
};

export class SaaSPublishingEngine {
  constructor() {
    this._enabled = false;
    this._intervalIds = [];
    this._tenantPublishProfiles = new Map();
    this._isSleeping = false;
  }

  registerTenantProfile(tenantId, regionCode) {
    this._tenantPublishProfiles.set(tenantId, {
      region: regionCode || 'US',
      lastPublish: 0,
      publishCount: 0,
    });
  }

  activate() {
    this._enabled = true;
    typingSimulator.activate();
    logger.info('saas-pipeline-engine: active — jittered omnichannel publishing with regional sleep gating');
    return { active: true };
  }

  deactivate() {
    this._enabled = false;
    typingSimulator.deactivate();
    for (const id of this._intervalIds) clearTimeout(id);
    this._intervalIds = [];
    logger.info('saas-pipeline-engine: deactivated');
  }

  async publishContent(product, platform, regionCode) {
    if (!this._enabled) return { error: 'engine_not_active' };
    if (this._isSleepTime(regionCode)) {
      logger.info(`saas-pipeline: sleep gating active for ${regionCode} — deferring publish`);
      return { error: 'sleep_gate_active' };
    }
    const article = generateArticleForPlatform(product, platform);
    if (!article) return { error: 'no_article_generated' };
    const regionText = localeEngine.localizeComment(article.title + ' ' + article.body, regionCode || 'US');
    const humanized = await typingSimulator.simulateTypingWithMistakes(regionText);
    const formatted = PUBLISH_FORMATS[platform]?.format(article.title, humanized, article.tags) || {};
    logger.info(`saas-pipeline: published to ${platform} — "${article.title.slice(0, 50)}" (${regionCode || 'US'})`);
    const profile = this._getOrCreateProfile(product.productName, regionCode);
    profile.lastPublish = Date.now();
    profile.publishCount++;
    return { platform, title: article.title, formatted, humanized: true, region: regionCode || 'US' };
  }

  _schedulePublishingCycle(fn, minMinutes, maxMinutes) {
    const schedule = () => {
      const delay = (Math.floor(Math.random() * (maxMinutes - minMinutes + 1)) + minMinutes) * 60 * 1000;
      const id = setTimeout(async () => {
        const anyActive = Array.from(this._tenantPublishProfiles.values()).some(p => !this._isSleepTime(p.region));
        if (!anyActive || this._isSleeping) {
          schedule();
          return;
        }
        try {
          await fn();
        } catch (e) {
          logger.warn(`saas-pipeline: publish cycle error — ${e.message}`);
        }
        schedule();
      }, delay);
      this._intervalIds.push(id);
    };
    schedule();
  }

  _getOrCreateProfile(tenantId, regionCode) {
    if (!this._tenantPublishProfiles.has(tenantId)) {
      this._tenantPublishProfiles.set(tenantId, { region: regionCode || 'US', lastPublish: 0, publishCount: 0 });
    }
    return this._tenantPublishProfiles.get(tenantId);
  }

  _isSleepTime(regionCode) {
    const peaks = REGION_ACTIVITY_PEAKS[regionCode] || REGION_ACTIVITY_PEAKS.US;
    const now = new Date();
    const hour = now.getHours();
    const dayOfWeek = now.getDay();
    if (dayOfWeek === 0 || dayOfWeek === 6) {
      return hour >= 0 && hour < 9;
    }
    const allWindows = [peaks.morning, peaks.afternoon, peaks.evening].flat();
    const inPeak = allWindows.some(([start, end]) => hour >= start && hour < end);
    if (!inPeak) return true;
    if (localeEngine.isHoliday(regionCode, now)) return true;
    return false;
  }

  getStatus() {
    return {
      enabled: this._enabled,
      tenantProfiles: this._tenantPublishProfiles.size,
      scheduledCycles: this._intervalIds.length,
    };
  }
}

export function generateStripeTemplate(productName) {
  const slug = slugify(productName);
  return {
    path: 'src/monetization/stripe-checkout.js',
    content: `const STRIPE_PUBLISHABLE_KEY = process.env.NEXT_PUBLIC_STRIPE_KEY || 'pk_test_xxxxxxxxxxxx';
const PRICES = [
  { id: 'price_starter', lookup: 'starter', name: 'Starter', amount: 499, interval: 'month', features: ['Core features', 'Email support'] },
  { id: 'price_pro', lookup: 'pro', name: 'Pro', amount: 999, interval: 'month', features: ['All features', 'Priority support', 'API access'] },
  { id: 'price_enterprise', lookup: 'enterprise', name: 'Enterprise', amount: 2999, interval: 'month', features: ['Custom features', 'Dedicated support', 'SLA', 'White-label'] },
];
export async function redirectToCheckout(priceId, userId) {
  try {
    const stripe = await loadStripe(STRIPE_PUBLISHABLE_KEY);
    const res = await fetch('/api/create-checkout-session', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ priceId, userId, successUrl: window.location.origin + '/success', cancelUrl: window.location.origin + '/pricing' }),
    });
    const session = await res.json();
    const result = await stripe.redirectToCheckout({ sessionId: session.id });
    if (result.error) throw new Error(result.error.message);
  } catch (e) {
    window.location.href = '/pricing?error=' + encodeURIComponent(e.message);
  }
}
export function renderPricingTable() {
  return PRICES.map(tier => \`
    <div class="pricing-tier">
      <h3>\${tier.name}</h3>
      <div class="price">$\${(tier.amount / 100).toFixed(2)}<span>/\${tier.interval}</span></div>
      <ul>\${tier.features.map(f => '<li>' + f + '</li>').join('')}</ul>
      <button onclick="redirectToCheckout('\${tier.id}', 'guest')">\${tier.amount === 0 ? 'Get Started' : 'Subscribe'}</button>
    </div>
  \`).join('');
}
export const STRIPE_ENV_TEMPLATE = \`STRIPE_SECRET_KEY=sk_test_your_secret_key
STRIPE_PUBLISHABLE_KEY=pk_test_your_publishable_key
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret\`;`,
  };
}

export function generateLandingPage(trend) {
  const title = trend?.title || 'SaaS Landing';
  const slug = slugify(title);
  const tagline = trend?.description ? trend.description.slice(0, 100) : 'Built with Emerald AGI';
  const year = new Date().getFullYear();
  const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>${title} — SaaS Solution</title>
  <meta name="description" content="${tagline}" />
  <meta property="og:title" content="${title}" />
  <meta property="og:description" content="${tagline}" />
  <meta property="og:type" content="website" />
  <meta name="twitter:card" content="summary_large_image" />
  <link rel="canonical" href="https://${slug}.vercel.app" />
  <script type="application/ld+json">
  {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    "name": "${title}",
    "applicationCategory": "BusinessApplication",
    "description": "${tagline}",
    "offers": {
      "@type": "AggregateOffer",
      "lowPrice": "4.99",
      "highPrice": "29.99",
      "priceCurrency": "USD"
    }
  }
  </script>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;line-height:1.6;color:#1a1a2e;background:#f8f9fa}
    .hero{text-align:center;padding:5rem 2rem;background:linear-gradient(135deg,#667eea,#764ba2);color:#fff}
    .hero h1{font-size:2.8rem;margin-bottom:0.5rem}
    .hero p{font-size:1.2rem;opacity:0.9;max-width:600px;margin:1rem auto}
    .btn{display:inline-block;padding:0.8rem 2rem;background:#fff;color:#667eea;border-radius:6px;text-decoration:none;font-weight:600;margin-top:1rem}
    .features{display:grid;grid-template-columns:repeat(auto-fit,minmax(250px,1fr));gap:2rem;padding:4rem 2rem;max-width:1100px;margin:0 auto}
    .feature-card{background:#fff;padding:2rem;border-radius:10px;box-shadow:0 4px 12px rgba(0,0,0,0.08);text-align:center}
    .pricing{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:2rem;padding:4rem 2rem;max-width:1100px;margin:0 auto}
    .pricing-tier{background:#fff;padding:2rem;border-radius:10px;box-shadow:0 4px 12px rgba(0,0,0,0.08);text-align:center}
    .pricing-tier .price{font-size:2rem;font-weight:700;color:#1a1a2e;margin:1rem 0}
    .cta{text-align:center;padding:4rem 2rem;background:#1a1a2e;color:#fff}
    .cta h2{font-size:2rem;margin-bottom:1rem}
    .cta .btn{background:#667eea;color:#fff}
    footer{text-align:center;padding:2rem;color:#999;font-size:0.85rem}
    @media(max-width:600px){.hero h1{font-size:1.8rem}}
  </style>
</head>
<body>
  <section class="hero">
    <h1>${title}</h1>
    <p>${tagline}</p>
    <a href="/pricing" class="btn">Get Started Free</a>
  </section>
  <section class="features">
    <div class="feature-card"><h3>Lightning Fast</h3><p>Built for speed with modern architecture</p></div>
    <div class="feature-card"><h3>Secure</h3><p>Enterprise-grade security out of the box</p></div>
    <div class="feature-card"><h3>Scalable</h3><p>Grows with your business seamlessly</p></div>
  </section>
  <section class="pricing">
    <div class="pricing-tier"><h3>Starter</h3><div class="price">$4.99<span>/month</span></div><a href="/api/create-checkout-session?price=price_starter" class="btn">Subscribe</a></div>
    <div class="pricing-tier"><h3>Pro</h3><div class="price">$9.99<span>/month</span></div><a href="/api/create-checkout-session?price=price_pro" class="btn">Subscribe</a></div>
    <div class="pricing-tier"><h3>Enterprise</h3><div class="price">$29.99<span>/month</span></div><a href="/api/create-checkout-session?price=price_enterprise" class="btn">Contact Sales</a></div>
  </section>
  <section class="cta"><h2>Ready to Get Started?</h2><p>Join thousands of teams using ${title}</p><a href="/pricing" class="btn">Start Free Trial</a></section>
  <footer><p>&copy; ${year} ${title} — <a href="/privacy">Privacy Policy</a> | <a href="/terms">Terms of Service</a></p></footer>
</body>
</html>`;
  return html;
}

export function generateCheckoutAPI() {
  return {
    path: 'api/create-checkout-session.js',
    content: `import Stripe from 'stripe';
const stripe = new Stripe(process.env.STRIPE_SECRET_KEY, { apiVersion: '2023-10-16' });
export default async function handler(req, res) {
  if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' });
  const { priceId, userId, successUrl, cancelUrl } = req.body;
  if (!priceId) return res.status(400).json({ error: 'priceId is required' });
  try {
    const session = await stripe.checkout.sessions.create({
      mode: 'subscription', payment_method_types: ['card'],
      line_items: [{ price: priceId, quantity: 1 }],
      client_reference_id: userId || 'anonymous',
      success_url: successUrl || req.headers.origin + '/success?session_id={CHECKOUT_SESSION_ID}',
      cancel_url: cancelUrl || req.headers.origin + '/pricing',
      metadata: { userId: userId || '' },
    });
    return res.status(200).json({ id: session.id, url: session.url });
  } catch (e) {
    return res.status(500).json({ error: e.message });
  }
}`,
  };
}

export function generateSEOConfig(productName, tagline) {
  return {
    path: 'src/config/seo.js',
    content: `export default {
  siteName: '${productName}',
  tagline: '${tagline || 'SaaS built with Emerald'}',
  language: 'en', locale: 'en_US',
  twitterHandle: '@emerald_agi',
  defaultImage: '/og-image.png', themeColor: '#667eea',
  structuredData: {
    '@context': 'https://schema.org', '@type': 'SoftwareApplication',
    name: '${productName}', applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    offers: { '@type': 'AggregateOffer', lowPrice: '4.99', highPrice: '29.99', priceCurrency: 'USD' },
  },
};`,
  };
}

export function generatePrivacyPage(productName) {
  const year = new Date().getFullYear();
  const slug = slugify(productName);
  return {
    path: 'public/privacy/index.html',
    content: `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" /><title>Privacy Policy — ${productName}</title><meta name="robots" content="noindex" /></head><body style="max-width:800px;margin:0 auto;padding:2rem;font-family:sans-serif;line-height:1.6"><h1>Privacy Policy</h1><p>Last updated: ${new Date().toISOString().split('T')[0]}</p><h2>Data Collection</h2><p>We collect only the data necessary to provide our service: email address, payment information (processed securely by Stripe), and usage analytics.</p><h2>Data Usage</h2><p>Your data is used solely to operate and improve our service. We never sell your personal information to third parties.</p><h2>Contact</h2><p>Email: privacy@${slug}.com</p><p>&copy; ${year} ${productName}</p></body></html>`,
  };
}

export function generateTermsPage(productName) {
  const year = new Date().getFullYear();
  return {
    path: 'public/terms/index.html',
    content: `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" /><title>Terms of Service — ${productName}</title><meta name="robots" content="noindex" /></head><body style="max-width:800px;margin:0 auto;padding:2rem;font-family:sans-serif;line-height:1.6"><h1>Terms of Service</h1><p>Last updated: ${new Date().toISOString().split('T')[0]}</p><h2>Use of Service</h2><p>By using ${productName}, you agree to these terms. You must be at least 18 years old to use this service.</p><h2>Subscriptions</h2><p>Subscriptions auto-renew monthly. You can cancel at any time from your account dashboard.</p><h2>Limitation of Liability</h2><p>${productName} is provided "as is" without warranty of any kind.</p><p>&copy; ${year} ${productName}</p></body></html>`,
  };
}

export function generatePublishingPlan(product, count = 10) {
  const slug = slugify(product.productName || 'product');
  const topics = [
    { title: `Why ${product.productName} is the ${product.category || 'SaaS'} Tool You Need in ${new Date().getFullYear()}`, body: `A comprehensive look at how ${product.productName} solves real problems for teams. Unlike traditional solutions, ${product.productName} focuses on delivering value from day one with zero configuration overhead.` },
    { title: `How to ${product.category ? 'Optimize Your ' + product.category : 'Accelerate Your Workflow'} with ${product.productName}`, body: `Step-by-step guide to getting the most out of ${product.productName}. Learn the strategies that power users employ to maximize efficiency.` },
    { title: `${product.productName} vs Competitors: An Honest Comparison for ${product.category || 'Teams'}`, body: `An unbiased comparison of ${product.productName} against other solutions in the ${product.category || 'market'} space. See where each excels and which fits your needs.` },
    { title: `10 ${product.category || 'Productivity'} Tips for ${new Date().getFullYear()}`, body: `Discover the top ${product.category || 'productivity'} strategies for ${new Date().getFullYear()}. ${product.productName} makes implementing these tips effortless.` },
    { title: `The Future of ${product.category || 'SaaS'}: Trends Shaping ${new Date().getFullYear()}`, body: `An analysis of emerging trends in ${product.category || 'the SaaS industry'} and how ${product.productName} is positioned at the forefront of innovation.` },
    { title: `Building a ${product.category || 'Business'} with ${product.productName}: A Founder's Story`, body: `Real insights from founders who use ${product.productName} to streamline their operations and scale faster than ever before.` },
    { title: `Case Study: How Team Achieved 3x Growth with ${product.productName}`, body: `A detailed case study examining how a team leveraged ${product.productName} to achieve remarkable growth metrics and operational efficiency.` },
    { title: `${product.productName} Integrations: Connecting Your ${product.category || 'Tech'} Stack`, body: `Explore the integration ecosystem of ${product.productName} and learn how to connect it with your existing tools for maximum productivity.` },
    { title: `Security Best Practices for ${product.category || 'SaaS'} in ${new Date().getFullYear()}`, body: `An overview of security best practices for ${product.category || 'SaaS'} platforms, highlighting how ${product.productName} implements enterprise-grade protection.` },
    { title: `Getting Started with ${product.productName}: The Complete Guide`, body: `Everything you need to know to get started with ${product.productName}. From setup to advanced features, this guide covers it all.` },
  ].slice(0, count);

  return HIGH_DOMAIN_SITES.map(site => {
    const siteTopics = topics.map((t, i) => ({
      ...t,
      tags: site.name === 'medium' ? site.tags : site.name === 'devto' ? site.tags : site.name === 'reddit' ? [site.subreddits[i % site.subreddits.length]] : [site.topics[i % site.topics.length]],
      target: site.name === 'reddit' ? site.subreddits[i % site.subreddits.length] : site.name === 'quora' ? site.topics[i % site.topics.length] : null,
      platform: site.name,
      format: PUBLISH_FORMATS[site.name],
    }));
    return { platform: site.name, articles: siteTopics };
  });
}

export function generateArticleForPlatform(product, platform, index = 0) {
  const plans = generatePublishingPlan(product, 1);
  const sitePlan = plans.find(p => p.platform === platform) || plans[0];
  if (!sitePlan || !sitePlan.articles || sitePlan.articles.length === 0) return null;
  return sitePlan.articles[index % sitePlan.articles.length];
}

export function getContentMatrix() {
  return HIGH_DOMAIN_SITES.map(s => ({
    platform: s.name,
    targets: s.subreddits || s.topics || s.tags,
    targetCount: (s.subreddits || s.topics || s.tags).length,
  }));
}

const saasPipelineEngine = new SaaSPublishingEngine();
export default saasPipelineEngine;
