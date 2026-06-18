import fetch from 'node-fetch';
import logger from '../utils/logger.js';

const COMPETITOR_MARKETS = [
  {
    name: 'Asana',
    category: 'project-management',
    strengths: ['workflow automation', 'integrations', 'enterprise'],
    weaknesses: ['pricing', 'complexity', 'mobile'],
    positioning: 'simpler, cheaper, faster for small teams',
  },
  {
    name: 'Notion',
    category: 'productivity',
    strengths: ['flexibility', 'all-in-one', 'templates'],
    weaknesses: ['performance', 'offline', 'database limits'],
    positioning: 'purpose-built workflows without the overhead',
  },
  {
    name: 'Slack',
    category: 'communication',
    strengths: ['ubiquity', 'integrations', 'channels'],
    weaknesses: ['distraction', 'cost', 'search'],
    positioning: 'focused communication that respects your attention',
  },
  {
    name: 'GitHub',
    category: 'devtools',
    strengths: ['ecosystem', 'actions', 'community'],
    weaknesses: ['complexity', 'pricing tiers', 'CI/CD limits'],
    positioning: 'lean DevOps for lean teams',
  },
  {
    name: 'Stripe',
    category: 'fintech',
    strengths: ['developer experience', 'global', 'documentation'],
    weaknesses: ['pricing', 'support', 'payout delays'],
    positioning: 'transparent pricing, instant payouts, white-glove support',
  },
  {
    name: 'Calendly',
    category: 'scheduling',
    strengths: ['simplicity', 'integrations', ' ubiquity'],
    weaknesses: ['features', 'customization', 'pricing'],
    positioning: 'smart scheduling with AI-driven preferences',
  },
  {
    name: 'Figma',
    category: 'design',
    strengths: ['collaboration', 'browser-based', 'plugins'],
    weaknesses: ['performance', 'vector tools', 'prototyping limits'],
    positioning: 'design tools for non-designers who ship fast',
  },
  {
    name: 'Linear',
    category: 'project-management',
    strengths: ['speed', 'UX', 'keyboard shortcuts'],
    weaknesses: ['features', 'integrations', 'enterprise'],
    positioning: 'issue tracking for teams that move at startup speed',
  },
  {
    name: 'Vercel',
    category: 'hosting',
    strengths: ['developer experience', 'edge', 'analytics'],
    weaknesses: ['pricing', 'cold starts', 'lock-in'],
    positioning: 'open-source friendly, predictable pricing, global by default',
  },
  {
    name: 'Canva',
    category: 'design',
    strengths: ['ease of use', 'templates', 'brand kits'],
    weaknesses: ['advanced features', 'export limits', ' collaboration'],
    positioning: 'professional design assets with AI generation built in',
  },
];

const MARKET_MAP = new Map(COMPETITOR_MARKETS.map(c => [c.category, c]));

class CompetitiveIntelligenceNetwork {
  constructor() {
    this._competitors = [...COMPETITOR_MARKETS];
    this._scanHistory = [];
    this._activeWatchlist = [];
    this._enabled = false;
  }

  activate() {
    this._enabled = true;
    logger.info(`competitive-intel: active — tracking ${this._competitors.length} competitors across ${new Set(this._competitors.map(c => c.category)).size} categories`);
    return { active: true, competitors: this._competitors.length, categories: new Set(this._competitors.map(c => c.category)).size };
  }

  deactivate() {
    this._enabled = false;
    logger.info('competitive-intel: deactivated');
  }

  analyzeCategory(category) {
    if (!this._enabled) return { error: 'not_active' };

    const competitor = MARKET_MAP.get(category);
    if (!competitor) return this._genericAnalysis(category);

    return {
      category,
      competitor: competitor.name,
      competitorStrengths: competitor.strengths,
      competitorWeaknesses: competitor.weaknesses,
      recommendedPositioning: competitor.positioning,
      advantageSuggestions: this._generateAdvantages(competitor),
      marketGaps: this._identifyGaps(competitor),
    };
  }

  analyzeProduct(product) {
    if (!this._enabled) return { error: 'not_active' };
    const category = product.category || 'default';
    const analysis = this.analyzeCategory(category);
    const entry = { timestamp: new Date().toISOString(), productName: product.productName, category, analysis };
    this._scanHistory.push(entry);
    if (this._scanHistory.length > 1000) this._scanHistory.shift();
    return entry;
  }

  getCompetitiveAdvice(product) {
    if (!this._enabled) return '';
    const analysis = this.analyzeProduct(product);
    if (analysis.error) return '';

    const parts = [];
    const a = analysis.analysis || {};

    if (a.marketGaps && a.marketGaps.length > 0) {
      parts.push(`Market Gap: ${a.marketGaps[0]}`);
    }
    if (a.advantageSuggestions && a.advantageSuggestions.length > 0) {
      parts.push(`Advantage: ${a.advantageSuggestions[0]}`);
    }
    if (a.recommendedPositioning) {
      parts.push(`Position: ${a.recommendedPositioning}`);
    }

    return parts.length > 0
      ? `[Competitive Intel] ${product.productName} — ${parts.join(' | ')}`
      : '';
  }

  scanMarket() {
    if (!this._enabled) return { error: 'not_active' };

    const scan = {
      timestamp: new Date().toISOString(),
      trackedCompetitors: this._competitors.length,
      categories: [...new Set(this._competitors.map(c => c.category))],
      recentAnalyses: this._scanHistory.length,
    };
    this._activeWatchlist.push(scan);
    logger.info(`competitive-intel: market scan complete — ${this._scanHistory.length} products analyzed`);
    return scan;
  }

  getIntelReport() {
    return {
      enabled: this._enabled,
      competitorsTracked: this._competitors.length,
      analysesRun: this._scanHistory.length,
      categories: [...new Set(this._competitors.map(c => c.category))],
      lastScan: this._activeWatchlist.length > 0 ? this._activeWatchlist[this._activeWatchlist.length - 1] : null,
    };
  }

  _generateAdvantages(competitor) {
    return competitor.weaknesses.map(w => `Unlike ${competitor.name}, we prioritize ${w} as a core strength`);
  }

  _identifyGaps(competitor) {
    return [
      `SMBs underserved by ${competitor.name}'s enterprise pricing`,
      `Mobile-first experience lacking in ${competitor.name}'s offering`,
      `Integration simplicity that ${competitor.name} over-engineers`,
    ];
  }

  _genericAnalysis(category) {
    return {
      category,
      competitor: 'Market incumbents',
      competitorStrengths: ['brand recognition', 'market share', 'funding'],
      competitorWeaknesses: ['slow innovation', 'high pricing', 'poor support'],
      recommendedPositioning: `faster, cheaper, and more focused than incumbents in ${category}`,
      advantageSuggestions: [
        `Focus on underserved sub-segments within ${category}`,
        `Offer transparent pricing with no hidden fees`,
        `Provide superior onboarding and support experience`,
      ],
      marketGaps: [
        `Existing solutions in ${category} ignore small teams`,
        `Automation gap: most ${category} tools require manual work`,
      ],
    };
  }
}

const competitiveIntel = new CompetitiveIntelligenceNetwork();
export default competitiveIntel;
