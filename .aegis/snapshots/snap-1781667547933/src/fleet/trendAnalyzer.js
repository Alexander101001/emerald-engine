import scrapeTrends from '../modules/trendScraper.js';
import router from '../modules/aiRouter.js';
import logger from '../utils/logger.js';
import config from '../config.js';
import { cache, slugify } from '../utils/helpers.js';

const analysisCache = cache(3600000);

const SAAS_CATEGORIES = [
  { id: 'analytics',      name: 'Analytics',        patterns: ['analytics', 'metrics', 'dashboard', 'tracking', 'insights'] },
  { id: 'automation',     name: 'Automation',       patterns: ['automation', 'workflow', 'pipeline', 'integration', 'zap'] },
  { id: 'content',        name: 'Content',          patterns: ['content', 'blog', 'writing', 'seo', 'copy', 'article'] },
  { id: 'communication',  name: 'Communication',    patterns: ['chat', 'messaging', 'notification', 'email', 'sms'] },
  { id: 'design',         name: 'Design',           patterns: ['design', 'ui', 'ux', 'mockup', 'prototype', 'wireframe'] },
  { id: 'developer',      name: 'Developer Tools',  patterns: ['api', 'cli', 'dev', 'code', 'deploy', 'monitoring'] },
  { id: 'ecommerce',      name: 'E-Commerce',       patterns: ['shop', 'store', 'cart', 'checkout', 'product', 'inventory'] },
  { id: 'education',      name: 'Education',        patterns: ['course', 'learn', 'tutorial', 'training', 'classroom'] },
  { id: 'finance',        name: 'Finance',          patterns: ['finance', 'invoice', 'billing', 'budget', 'expense', 'tax'] },
  { id: 'health',         name: 'Health & Wellness', patterns: ['health', 'fitness', 'wellness', 'medical', 'therapy'] },
  { id: 'hr',             name: 'HR & Recruiting',  patterns: ['hire', 'recruit', 'hr', 'resume', 'interview', 'onboard'] },
  { id: 'marketing',      name: 'Marketing',        patterns: ['marketing', 'lead', 'campaign', 'conversion', 'funnel'] },
  { id: 'productivity',   name: 'Productivity',     patterns: ['productivity', 'task', 'project', 'todo', 'calendar', 'plan'] },
  { id: 'sales',          name: 'Sales',            patterns: ['sales', 'crm', 'deal', 'pipeline', 'prospect', 'quote'] },
  { id: 'social',         name: 'Social Media',     patterns: ['social', 'twitter', 'linkedin', 'instagram', 'tiktok'] },
  { id: 'support',        name: 'Customer Support', patterns: ['support', 'ticket', 'helpdesk', 'faq', 'knowledge', 'service'] },
];

export function classifyTrend(trendTitle) {
  const lower = trendTitle.toLowerCase();
  for (const cat of SAAS_CATEGORIES) {
    if (cat.patterns.some(p => lower.includes(p))) {
      return cat.id;
    }
  }
  return 'general';
}

export function findGaps(existingCategories) {
  return SAAS_CATEGORIES.filter(c => !existingCategories.includes(c.id));
}

export async function analyzeTrendsForProducts() {
  const cached = analysisCache.get('analysis');
  if (cached) return cached;

  logger.info('trendAnalyzer: analyzing global trends for SaaS opportunities');

  const trends = await scrapeTrends(config.TREND_SOURCES);
  const classified = trends.map(t => ({
    ...t,
    category: classifyTrend(t.title),
  }));

  const categories = [...new Set(classified.map(c => c.category))];
  const gaps = findGaps(categories);

  const combinedTrends = classified.slice(0, 5);
  const combinationPrompt = `You are a SaaS product architect. Given these trending topics:
${combinedTrends.map((t, i) => `${i + 1}. "${t.title}" (${t.category})`).join('\n')}

Design a NEW micro-SaaS product that combines the best capabilities from these trends into a single, focused tool.
Return ONLY valid JSON with: { "productName": "...", "tagline": "...", "category": "...", "targetAudience": "...", "keyFeatures": ["...", "..."], "monetization": "freemium|free|paid", "estimatedDevDays": 1-3 }`;

  const aiResult = await router.generate(combinationPrompt, { allowFallbackMock: true });
  let blueprint;
  try {
    blueprint = typeof aiResult.content === 'string' ? JSON.parse(aiResult.content) : aiResult.content;
  } catch {
    blueprint = {
      productName: combinedTrends.map(t => t.title.split(' ').slice(0, 2).join(' ')).join(' ').slice(0, 40),
      tagline: 'AI-powered tool for modern teams',
      category: categories[0] || 'general',
      targetAudience: 'startups and SMBs',
      keyFeatures: ['Feature A', 'Feature B', 'Feature C'],
      monetization: 'freemium',
      estimatedDevDays: 2,
    };
  }

  const result = {
    blueprint,
    trends: classified.slice(0, 10),
    categories,
    gaps,
    timestamp: new Date().toISOString(),
  };

  analysisCache.set('analysis', result);
  logger.info(`trendAnalyzer: generated blueprint "${blueprint.productName}" in ${blueprint.category} (${blueprint.keyFeatures.length} features)`);
  return result;
}

export function estimateMarketFit(trend, category) {
  const cat = SAAS_CATEGORIES.find(c => c.id === category);
  if (!cat) return 50;
  const lower = trend.title.toLowerCase();
  const matchCount = cat.patterns.filter(p => lower.includes(p)).length;
  return Math.min(100, Math.round((matchCount / cat.patterns.length) * 100));
}

export default { analyzeTrendsForProducts, classifyTrend, findGaps, estimateMarketFit };
