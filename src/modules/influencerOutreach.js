import logger from '../utils/logger.js';

const INFLUENCER_NICHES = {
  tech: [
    { name: 'TechTutorialsDaily', niche: 'coding', followers: 250000, engagement: 0.04, platform: 'youtube', rate: 500 },
    { name: 'StartupInsider', niche: 'startups', followers: 180000, engagement: 0.05, platform: 'twitter', rate: 300 },
    { name: 'DevLifePro', niche: 'devtools', followers: 120000, engagement: 0.06, platform: 'youtube', rate: 400 },
    { name: 'CodeWithClarity', niche: 'programming', followers: 90000, engagement: 0.07, platform: 'youtube', rate: 250 },
    { name: 'TheSaaSReview', niche: 'saas', followers: 75000, engagement: 0.045, platform: 'blog', rate: 200 },
  ],
  productivity: [
    { name: 'ProductivityPro', niche: 'productivity', followers: 300000, engagement: 0.035, platform: 'youtube', rate: 600 },
    { name: 'WorkflowWizard', niche: 'automation', followers: 150000, engagement: 0.05, platform: 'twitter', rate: 350 },
    { name: 'TimeMastery', niche: 'productivity', followers: 100000, engagement: 0.06, platform: 'instagram', rate: 300 },
  ],
  business: [
    { name: 'FounderFuel', niche: 'startups', followers: 200000, engagement: 0.04, platform: 'podcast', rate: 400 },
    { name: 'GrowthHackerHQ', niche: 'growth', followers: 160000, engagement: 0.055, platform: 'twitter', rate: 350 },
    { name: 'SmallBizCoach', niche: 'small-business', followers: 85000, engagement: 0.07, platform: 'youtube', rate: 200 },
  ],
  design: [
    { name: 'DesignDaily', niche: 'design', followers: 220000, engagement: 0.045, platform: 'instagram', rate: 450 },
    { name: 'UXUnlocked', niche: 'ux', followers: 110000, engagement: 0.055, platform: 'youtube', rate: 300 },
    { name: 'CreativeCodeLab', niche: 'web-design', followers: 70000, engagement: 0.08, platform: 'tiktok', rate: 200 },
  ],
  marketing: [
    { name: 'MarketingMaven', niche: 'marketing', followers: 280000, engagement: 0.04, platform: 'youtube', rate: 550 },
    { name: 'SEODaily', niche: 'seo', followers: 130000, engagement: 0.05, platform: 'twitter', rate: 300 },
    { name: 'ContentKing', niche: 'content-marketing', followers: 95000, engagement: 0.06, platform: 'blog', rate: 250 },
  ],
};

const OUTREACH_TEMPLATES = {
  youtube: {
    subject: 'Partner with {{product}} — sponsored content opportunity',
    body: `Hi {{name}},\n\nI'm a huge fan of your {{niche}} content on YouTube. Your recent video on {{topic}} was spot on.\n\nWe're building {{product}} — {{tagline}}. We think your audience would love what we're doing.\n\nWe'd love to sponsor a video or get your honest review. We offer:\n- {{compensation}} per video\n- Custom discount code for your audience ({{discount}}% off)\n- Early access to new features\n- Lifetime free {{plan}} plan for you\n\nWould you be open to a quick chat about this?\n\nBest,\n{{founder}}\nFounder, {{product}}`,
  },
  twitter: {
    subject: 'Let\'s collaborate — {{product}} x {{name}}',
    body: `Hey {{name}}, love your {{niche}} content! We're building {{product}} ({{tagline}}) and would love to set up an affiliate partnership.\n\nWe offer {{commission}}% commission on all referrals, plus a free {{plan}} plan for you to use and review.\n\nInterested? Let me know and I'll send over the details!\n\n— {{founder}}`,
  },
  blog: {
    subject: 'Review opportunity: {{product}}',
    body: `Hi {{name}},\n\nI've been reading your {{niche}} blog and I think {{product}} would be a great fit for your audience.\n\n{{product}} is {{tagline}}. We're looking for honest reviews and would love for you to be one of the first to try it.\n\nWhat we offer:\n- Free lifetime {{plan}} plan\n- {{compensation}} per review/post\n- Early access to all new features\n- Affiliate program ({{commission}}% recurring)\n\nWould you be interested?\n\nCheers,\n{{founder}}`,
  },
};

const COMMISSION_RATES = [10, 15, 20, 25, 30];
const COMPENSATION_RANGES = [200, 350, 500, 750, 1000];

class InfluencerOutreachModule {
  constructor() {
    this._outreachHistory = [];
    this._activePartnerships = [];
    this._enabled = false;
    this._totalOutreachSent = 0;
    this._partnersAcquired = 0;
    this._revenueAttributed = 0;
  }

  activate() {
    this._enabled = true;
    const totalInfluencers = Object.values(INFLUENCER_NICHES).flat().length;
    logger.info(`influencer-outreach: active — ${totalInfluencers} influencers in database across ${Object.keys(INFLUENCER_NICHES).length} niches`);
    return { active: true, influencers: totalInfluencers, niches: Object.keys(INFLUENCER_NICHES) };
  }

  deactivate() {
    this._enabled = false;
    logger.info('influencer-outreach: deactivated');
  }

  findInfluencers(product, niche) {
    if (!this._enabled) return { error: 'not_active' };

    const targetNiche = niche || product.category || 'tech';
    const candidates = INFLUENCER_NICHES[targetNiche] || INFLUENCER_NICHES.tech;
    const nicheInfluencers = INFLUENCER_NICHES[targetNiche] || [];

    const allInfluencers = nicheInfluencers.length > 0
      ? nicheInfluencers
      : Object.values(INFLUENCER_NICHES).flat();

    const sorted = allInfluencers
      .filter(inf => !this._activePartnerships.some(p => p.name === inf.name))
      .sort((a, b) => (b.engagement * b.followers) - (a.engagement * a.followers));

    const top5 = sorted.slice(0, 5);
    logger.info(`influencer-outreach: found ${top5.length} candidates for "${product.productName || 'Product'}" in ${targetNiche}`);
    return top5;
  }

  generateOutreach(influencer, product) {
    if (!this._enabled) return { error: 'not_active' };

    const commission = COMMISSION_RATES[Math.floor(Math.random() * COMMISSION_RATES.length)];
    const compensation = COMPENSATION_RANGES[Math.floor(Math.random() * COMPENSATION_RANGES.length)];
    const discount = Math.floor(Math.random() * 30) + 10;
    const plan = 'Pro';

    const template = OUTREACH_TEMPLATES[influencer.platform] || OUTREACH_TEMPLATES.blog;
    const proposal = {
      to: influencer.name,
      platform: influencer.platform,
      niche: influencer.niche,
      subject: template.subject
        .replace(/\{\{product\}\}/g, product.productName || 'Product')
        .replace(/\{\{name\}\}/g, influencer.name),
      body: template.body
        .replace(/\{\{name\}\}/g, influencer.name)
        .replace(/\{\{product\}\}/g, product.productName || 'Product')
        .replace(/\{\{tagline\}\}/g, product.tagline || 'a new way to work')
        .replace(/\{\{niche\}\}/g, influencer.niche)
        .replace(/\{\{topic\}\}/g, influencer.niche + ' tips')
        .replace(/\{\{compensation\}\}/g, '$' + compensation)
        .replace(/\{\{discount\}\}/g, discount)
        .replace(/\{\{commission\}\}/g, commission)
        .replace(/\{\{plan\}\}/g, plan)
        .replace(/\{\{founder\}\}/g, 'Alex'),
      commission: commission + '%',
      compensation: '$' + compensation,
      discount: discount + '%',
    };

    const record = {
      id: `out-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
      timestamp: new Date().toISOString(),
      influencer: influencer.name,
      platform: influencer.platform,
      product: product.productName || 'Product',
      proposal,
      status: 'sent',
    };

    this._outreachHistory.push(record);
    this._totalOutreachSent++;
    logger.info(`influencer-outreach: proposal sent to ${influencer.name} (${influencer.platform}) for "${product.productName || 'Product'}"`);
    return record;
  }

  recordPartnership(outreachId) {
    const outreach = this._outreachHistory.find(o => o.id === outreachId);
    if (!outreach) return false;

    outreach.status = 'accepted';
    const partnership = {
      ...outreach,
      partnershipStarted: new Date().toISOString(),
      revenueGenerated: 0,
      active: true,
    };
    this._activePartnerships.push(partnership);
    this._partnersAcquired++;
    logger.info(`influencer-outreach: partnership established with ${outreach.influencer}`);
    return partnership;
  }

  recordRevenue(partnerName, amount) {
    const partner = this._activePartnerships.find(p => p.influencer === partnerName);
    if (!partner) return false;
    partner.revenueGenerated += amount;
    this._revenueAttributed += amount;
    return true;
  }

  getOutreachReport() {
    return {
      enabled: this._enabled,
      totalOutreachSent: this._totalOutreachSent,
      partnersAcquired: this._partnersAcquired,
      activePartnerships: this._activePartnerships.length,
      revenueAttributed: this._revenueAttributed,
      conversionRate: this._totalOutreachSent > 0
        ? ((this._partnersAcquired / this._totalOutreachSent) * 100).toFixed(1) + '%'
        : '0%',
      recentOutreach: this._outreachHistory.slice(-5).map(o => ({
        influencer: o.influencer,
        platform: o.platform,
        product: o.product,
        status: o.status,
        date: o.timestamp,
      })),
      topPartners: this._activePartnerships
        .sort((a, b) => b.revenueGenerated - a.revenueGenerated)
        .slice(0, 5)
        .map(p => ({ name: p.influencer, revenue: p.revenueGenerated })),
    };
  }
}

const influencerOutreach = new InfluencerOutreachModule();
export default influencerOutreach;
