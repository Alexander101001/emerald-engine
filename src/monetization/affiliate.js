import { uid, slugify, cache } from '../utils/helpers.js';
import logger from '../utils/logger.js';

const CACHE_TTL = 86400000;
const linkCache = cache(CACHE_TTL);

const DEFAULT_NETWORKS = [
  {
    id: 'amazon',
    name: 'Amazon',
    baseUrl: 'https://amazon.com/dp/',
    tagParam: 'tag',
    defaultTag: 'emerald0a-20',
    commission: '1-10%',
    enabled: true,
  },
  {
    id: 'shareasale',
    name: 'ShareASale',
    baseUrl: 'https://shareasale.com/r.cfm',
    tagParam: 'afftrack',
    defaultTag: 'emerald',
    commission: '5-20%',
    enabled: true,
  },
  {
    id: 'cj',
    name: 'Commission Junction',
    baseUrl: 'https://cj.com/click',
    tagParam: 'sid',
    defaultTag: 'emerald',
    commission: '3-15%',
    enabled: true,
  },
  {
    id: 'digistore24',
    name: 'Digistore24',
    baseUrl: 'https://digistore24.com/redir',
    tagParam: 'ref',
    defaultTag: 'emerald_dev',
    commission: '20-60%',
    enabled: true,
  },
];

const CATEGORY_MAP = {
  hosting: { network: 'shareasale', keywords: ['hosting', 'server', 'vps', 'domain', 'cloud'] },
  books: { network: 'amazon', keywords: ['book', 'guide', 'course', 'tutorial', 'ebook'] },
  software: { network: 'cj', keywords: ['software', 'saas', 'app', 'tool', 'platform', 'api'] },
  education: { network: 'digistore24', keywords: ['course', 'training', 'certification', 'workshop'] },
  ecommerce: { network: 'amazon', keywords: ['shop', 'store', 'product', 'buy', 'deal'] },
};

export class AffiliateManager {
  constructor() {
    this.networks = [...DEFAULT_NETWORKS];
    this._clicks = new Map();
    this._conversions = new Map();
  }

  addNetwork(network) {
    this.networks.push(network);
  }

  getLink(productId, { network, tag, category, url } = {}) {
    const cacheKey = `${network || 'auto'}:${productId}:${tag || ''}:${category || ''}`;
    const cached = linkCache.get(cacheKey);
    if (cached) return cached;

    let net;
    if (network) {
      net = this.networks.find(n => n.id === network);
    } else if (category && CATEGORY_MAP[category]) {
      const netId = CATEGORY_MAP[category].network;
      net = this.networks.find(n => n.id === netId);
    }
    net = net || this.networks[0];

    const affTag = tag || net.defaultTag;
    const finalUrl = url || `${net.baseUrl}${productId}`;
    const separator = finalUrl.includes('?') ? '&' : '?';
    const trackedUrl = `${finalUrl}${separator}${net.tagParam}=${affTag}&ref=emerald-${uid(4)}`;

    const link = {
      url: trackedUrl,
      network: net.id,
      networkName: net.name,
      commission: net.commission,
      productId,
      tag: affTag,
      id: `aff-${uid(6)}`,
      createdAt: new Date().toISOString(),
    };

    linkCache.set(cacheKey, link);
    return link;
  }

  categorizeProduct(productTitle) {
    const lower = productTitle.toLowerCase();
    for (const [cat, config] of Object.entries(CATEGORY_MAP)) {
      if (config.keywords.some(kw => lower.includes(kw))) {
        return cat;
      }
    }
    return 'software';
  }

  trackClick(linkId, userId) {
    const entry = this._clicks.get(linkId) || { clicks: 0, users: new Set(), firstClick: null, lastClick: null };
    entry.clicks++;
    if (userId) entry.users.add(userId);
    entry.lastClick = new Date().toISOString();
    if (!entry.firstClick) entry.firstClick = entry.lastClick;
    this._clicks.set(linkId, entry);
  }

  trackConversion(linkId, amount, userId) {
    const conv = {
      linkId,
      amount,
      userId: userId || 'anonymous',
      timestamp: new Date().toISOString(),
      id: `conv-${uid(8)}`,
    };
    this._conversions.set(conv.id, conv);
    logger.info(`affiliate: conversion tracked ${conv.id} — $${(amount / 100).toFixed(2)}`);
    return conv;
  }

  getStats() {
    const totalClicks = Array.from(this._clicks.values()).reduce((s, e) => s + e.clicks, 0);
    const totalConversions = this._conversions.size;
    const totalRevenue = Array.from(this._conversions.values()).reduce((s, e) => s + e.amount, 0);
    return { totalClicks, totalConversions, totalRevenue, rate: totalClicks > 0 ? ((totalConversions / totalClicks) * 100).toFixed(2) + '%' : '0%' };
  }

  injectAffiliateLinks(html, pageTitle) {
    const category = this.categorizeProduct(pageTitle);

    const linkPlaceholders = html.match(/\[%affiliate:([^\]]+)%\]/g);
    if (!linkPlaceholders) return html;

    let result = html;
    for (const placeholder of linkPlaceholders) {
      const productId = placeholder.replace('[%affiliate:', '').replace('%]', '').trim();
      const link = this.getLink(productId, { category });
      result = result.replace(
        placeholder,
        `<a href="${link.url}" target="_blank" rel="sponsored noopener" class="affiliate-link" data-aff="${link.id}">
          ${link.networkName} — Check Price
        </a>`
      );
    }

    return result;
  }
}

const affiliateManager = new AffiliateManager();
export default affiliateManager;
