import logger from '../utils/logger.js';

const REFERRAL_TIERS = [
  { name: 'bronze', reward: '1 month free', threshold: 3, badge: '🥉' },
  { name: 'silver', reward: '3 months free', threshold: 10, badge: '🥈' },
  { name: 'gold', reward: '1 year free', threshold: 25, badge: '🥇' },
  { name: 'platinum', reward: 'Lifetime free access', threshold: 100, badge: '💎' },
];

const REFERRAL_TEMPLATES = {
  email: {
    subject: '{{name}} thinks you\'ll love {{product}}!',
    body: `Hey there,\n\n{{referee_name}} has been using {{product}} and thought you'd love it too.\n\nHere's your exclusive invite: {{referral_link}}\n\nYou both get {{reward}} when you sign up!\n\nCheers,\nThe {{product}} Team`,
  },
  social: {
    twitter: 'I\'ve been using {{product}} and it\'s incredible! Get started and we both get {{reward}} 🎉 {{referral_link}}',
    linkedin: 'I\'ve been really impressed with {{product}}. If you\'re looking for {{benefit}}, check it out — we both get {{reward}} when you join! {{referral_link}}',
  },
  embed: {
    widget: `<div class="referral-widget">
  <p>Love {{product}}? Share the love!</p>
  <a href="{{referral_link}}" class="referral-btn">Invite a friend →</a>
  <small>You both get {{reward}}</small>
</div>`,
    popup: `<div class="referral-popup">
  <h3>You've earned {{reward}}!</h3>
  <p>Share your referral link and earn more:</p>
  <input type="text" value="{{referral_link}}" readonly onclick="this.select()">
  <div class="share-buttons">
    <a href="https://twitter.com/intent/tweet?text={{encoded_text}}">Twitter</a>
    <a href="https://www.linkedin.com/sharing/share-offsite/?url={{encoded_link}}">LinkedIn</a>
  </div>
</div>`,
  },
};

class AutomatedReferralEngine {
  constructor() {
    this._campaigns = [];
    this._enabled = false;
    this._totalReferralsGenerated = 0;
    this._conversionRate = 0;
    this._tiers = [...REFERRAL_TIERS];
  }

  activate() {
    this._enabled = true;
    logger.info(`referral-engine: active — ${this._tiers.length} tiers configured`);
    return { active: true, tiers: this._tiers.length };
  }

  deactivate() {
    this._enabled = false;
    logger.info('referral-engine: deactivated');
  }

  generateCampaign(product) {
    if (!this._enabled) return { error: 'not_active' };

    const campaignId = `ref-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`;
    const baseLink = `https://${product.productName?.toLowerCase().replace(/\s+/g, '-') || 'product'}.app/refer?ref={{referee_code}}`;

    const campaign = {
      id: campaignId,
      productName: product.productName || 'Product',
      createdAt: new Date().toISOString(),
      referralLink: baseLink,
      rewardStructure: this._tiers.map(t => ({
        tier: t.name,
        referralsNeeded: t.threshold,
        reward: t.reward,
        badge: t.badge,
      })),
      templates: {
        email: this._fillTemplate(REFERRAL_TEMPLATES.email, product, baseLink),
        twitter: this._fillTemplate({ body: REFERRAL_TEMPLATES.social.twitter }, product, baseLink).body,
        linkedin: this._fillTemplate({ body: REFERRAL_TEMPLATES.social.linkedin }, product, baseLink).body,
        widget: this._fillTemplate({ body: REFERRAL_TEMPLATES.embed.widget }, product, baseLink).body,
        popup: this._fillTemplate({ body: REFERRAL_TEMPLATES.embed.popup }, product, baseLink).body,
      },
      active: true,
      stats: { generated: 0, converted: 0, revenueAttributed: 0 },
    };

    this._campaigns.push(campaign);
    logger.info(`referral-engine: campaign "${campaignId}" created for "${product.productName || 'Product'}"`);
    return campaign;
  }

  getReferralCode(productName, userId) {
    const hash = (productName + userId).split('').reduce((a, c) => a + c.charCodeAt(0), 0);
    const code = (hash % 90000 + 10000).toString(36).toUpperCase();
    return code;
  }

  getCampaignTemplates(product) {
    if (!this._enabled) return {};
    return {
      emailHtml: `<div style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:20px">
        <h2>You've been invited to ${product.productName || 'Product'}!</h2>
        <p>{{friend_name}} thinks you'll love it. Click below to get started — you both get 1 month free.</p>
        <a href="{{referral_link}}" style="display:inline-block;padding:12px 24px;background:#10b981;color:#fff;text-decoration:none;border-radius:6px">Claim Your Reward →</a>
      </div>`,
      widgetJs: `(function() {
        var link = '{{referral_link}}';
        var widget = document.createElement('div');
        widget.innerHTML = '<a href="' + link + '" style="position:fixed;bottom:20px;right:20px;background:#10b981;color:#fff;padding:12px 20px;border-radius:30px;text-decoration:none;font-family:sans-serif;box-shadow:0 4px 12px rgba(0,0,0,0.15);z-index:9999">🎁 Refer a friend →</a>';
        document.body.appendChild(widget);
      })();`,
    };
  }

  updateStats(conversionData) {
    const campaign = this._campaigns.find(c => c.id === conversionData.campaignId);
    if (!campaign) return false;

    campaign.stats.converted++;
    campaign.stats.revenueAttributed += conversionData.revenue || 0;
    this._totalReferralsGenerated++;

    const totalConverted = this._campaigns.reduce((a, c) => a + c.stats.converted, 0);
    const totalGenerated = this._campaigns.reduce((a, c) => a + c.stats.generated, 0);
    this._conversionRate = totalGenerated > 0 ? totalConverted / totalGenerated : 0;

    return true;
  }

  getActiveCampaigns() {
    return this._campaigns.filter(c => c.active);
  }

  getPerformanceReport() {
    return {
      enabled: this._enabled,
      totalCampaigns: this._campaigns.length,
      activeCampaigns: this._campaigns.filter(c => c.active).length,
      totalReferralsGenerated: this._totalReferralsGenerated,
      totalConverted: this._campaigns.reduce((a, c) => a + c.stats.converted, 0),
      conversionRate: (this._conversionRate * 100).toFixed(1) + '%',
      revenueAttributed: this._campaigns.reduce((a, c) => a + c.stats.revenueAttributed, 0),
      tiers: this._tiers,
    };
  }

  _fillTemplate(template, product, link) {
    const reward = '1 month free';
    const benefit = 'a better way to work';
    const result = {};
    for (const [key, val] of Object.entries(template)) {
      result[key] = val
        .replace(/\{\{product\}\}/g, product.productName || 'Product')
        .replace(/\{\{referral_link\}\}/g, link)
        .replace(/\{\{reward\}\}/g, reward)
        .replace(/\{\{benefit\}\}/g, benefit)
        .replace(/\{\{name\}\}/g, 'Your friend')
        .replace(/\{\{referee_name\}\}/g, 'a friend');
    }
    return result;
  }
}

const referralEngine = new AutomatedReferralEngine();
export default referralEngine;
