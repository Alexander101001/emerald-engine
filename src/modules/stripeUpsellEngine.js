import { randomBytes, createHmac } from 'crypto';
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';
import strategist from './conversionStrategist.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const UPSELL_STATE_DIR = resolve(PROJECT_ROOT, '.data');
const UPSELL_STATE_PATH = resolve(UPSELL_STATE_DIR, 'upsell_state.json');

const COMPLEMENTARY_MAP = {
  productivity: ['automation', 'analytics', 'communication'],
  automation: ['productivity', 'analytics', 'crm'],
  analytics: ['productivity', 'automation', 'visualization'],
  marketing: ['analytics', 'automation', 'seo'],
  devtools: ['productivity', 'automation', 'analytics'],
  finance: ['analytics', 'automation', 'productivity'],
  ecommerce: ['marketing', 'analytics', 'automation'],
  communication: ['productivity', 'automation', 'analytics'],
  crm: ['marketing', 'analytics', 'communication'],
  seo: ['marketing', 'analytics', 'content'],
};

const PRODUCT_TEMPLATES = {
  productivity: { name: 'FlowForge', tagline: 'Automate your entire workflow in minutes' },
  automation: { name: 'AutoPilot', tagline: 'AI-driven process automation for teams' },
  analytics: { name: 'InsightStack', tagline: 'Real-time business intelligence without the complexity' },
  marketing: { name: 'GrowthEngine', tagline: 'Multi-channel marketing automation platform' },
  devtools: { name: 'DevKit', tagline: 'Ship 10x faster with zero-config tooling' },
  finance: { name: 'FinFlow', tagline: 'Smart financial management for growing businesses' },
  ecommerce: { name: 'StoreBoost', tagline: 'Supercharge your online store conversions' },
  communication: { name: 'CommHub', tagline: 'Unified team communication and collaboration' },
  crm: { name: 'RelationOS', tagline: 'Customer relationships, automated and personalized' },
  seo: { name: 'RankRise', tagline: 'SEO automation that puts you on page one' },
  visualization: { name: 'VizStudio', tagline: 'Turn data into stunning visuals automatically' },
  content: { name: 'ContentMill', tagline: 'AI content generation at enterprise scale' },
};

const RECOVERY_DISCOUNTS = [
  { percent: 30, label: '30% off for 3 months', duration: 'quarterly' },
  { percent: 50, label: '50% off next billing cycle', duration: 'one_time' },
  { percent: 25, label: '25% lifetime discount', duration: 'lifetime' },
  { percent: 40, label: '40% off annual plan upgrade', duration: 'annual' },
];

const RECOVERY_MESSAGES = [
  'We noticed you are thinking of leaving. Before you go, we would love to make it right. Here is a personal offer just for you.',
  'Hate to see you go! Your feedback matters to us. In the meantime, please accept this exclusive discount as a thank you for being part of our community.',
  'We value you as a customer. Here is a special offer to keep you on board — no strings attached.',
  'Your journey with us does not have to end here. We have prepared a personalized retention offer based on your usage patterns.',
];

export class StripeUpsellEngine {
  constructor() {
    this._enabled = false;
    this._conversions = [];
    this._upsellsSent = 0;
    this._upsellSuccess = 0;
    this._activeProducts = [];
    this._recoveryAttempts = [];
    this._recoverySuccess = 0;
    this._webhookSecret = process.env.STRIPE_WEBHOOK_SECRET || '';
  }

  activate() {
    this._enabled = true;
    if (!existsSync(UPSELL_STATE_DIR)) mkdirSync(UPSELL_STATE_DIR, { recursive: true, mode: 0o700 });
    this._loadState();
    logger.info('stripe-upsell: active — cross-SaaS upselling + churn retention ready');
    return { active: true };
  }

  deactivate() {
    this._enabled = false;
    this._saveState();
    logger.info('stripe-upsell: deactivated');
  }

  registerProduct(product) {
    this._activeProducts.push({
      ...product,
      registeredAt: Date.now(),
      conversions: 0,
    });
    logger.info(`stripe-upsell: product registered — "${product.productName}"`);
  }

  async handleWebhook(payload, signature) {
    if (!this._enabled) return { error: 'not_active' };
    if (this._webhookSecret && signature) {
      const expected = this._signPayload(payload);
      if (signature !== expected) {
        logger.warn('stripe-upsell: webhook signature mismatch');
        return { error: 'invalid_signature' };
      }
    }
    try {
      const event = typeof payload === 'string' ? JSON.parse(payload) : payload;
      if (event.type === 'payment_intent.succeeded' || event.type === 'checkout.session.completed') {
        const session = event.data?.object || event.data || {};
        const customerEmail = session.customer_email || session.customer_details?.email || 'unknown@user.com';
        const productName = session.metadata?.product_name || session.metadata?.saas_name || 'Unknown Product';
        const category = session.metadata?.category || 'productivity';
        const conversion = {
          email: customerEmail,
          productName,
          category,
          amount: session.amount_total || session.amount_received || 0,
          currency: session.currency || 'usd',
          timestamp: Date.now(),
          userId: session.client_reference_id || session.customer || `usr_${randomBytes(4).toString('hex')}`,
        };
        this._conversions.push(conversion);
        const upsell = await this._generateUpsell(conversion);
        if (upsell) {
          this._upsellsSent++;
          this._saveState();
          logger.info(`stripe-upsell: upsell sent to ${customerEmail} — "${upsell.productName}" for ${productName} user`);
        }
        return { handled: true, conversion, upsell };
      }
      if (event.type === 'customer.subscription.deleted' || event.type === 'invoice.payment_failed') {
        const subscription = event.data?.object || event.data || {};
        const customerEmail = subscription.customer_email || subscription.customer_details?.email || subscription.customer || 'unknown@user.com';
        const productName = subscription.metadata?.product_name || subscription.metadata?.saas_name || 'Unknown Product';
        const category = subscription.metadata?.category || 'productivity';
        const recoveryResult = await this.handleCancellation(customerEmail, productName, category);
        return { handled: true, event: event.type, recovery: recoveryResult };
      }
      return { handled: false, reason: 'unhandled_event_type' };
    } catch (e) {
      logger.warn(`stripe-upsell: webhook processing failed — ${e.message}`);
      return { error: e.message };
    }
  }

  async handleCancellation(userEmail, productName, category) {
    if (!this._enabled) return { error: 'not_active' };
    const discount = RECOVERY_DISCOUNTS[Math.floor(Math.random() * RECOVERY_DISCOUNTS.length)];
    const message = RECOVERY_MESSAGES[Math.floor(Math.random() * RECOVERY_MESSAGES.length)];
    strategist.setProduct({ productName, category, tagline: 'Customer retention offer' });
    const personaText = strategist.craftConversionResponse(
      `I am thinking about cancelling my ${productName} subscription.`,
      'doubt'
    );
    const recovery = {
      userEmail,
      productName,
      category,
      discount,
      message,
      personaText,
      offeredAt: Date.now(),
      followUpAt: Date.now() + 259200000,
      recoveryCode: `rcv_${randomBytes(4).toString('hex')}`,
    };
    this._recoveryAttempts.push(recovery);
    this._saveState();
    logger.info(`stripe-upsell: churn recovery for ${userEmail} — ${discount.percent}% off ${discount.duration}`);
    return recovery;
  }

  resolveRecovery(recoveryCode, accepted) {
    const attempt = this._recoveryAttempts.find(r => r.recoveryCode === recoveryCode);
    if (!attempt) return { error: 'recovery_not_found' };
    attempt.resolvedAt = Date.now();
    attempt.accepted = accepted;
    if (accepted) this._recoverySuccess++;
    this._saveState();
    logger.info(`stripe-upsell: recovery ${recoveryCode} ${accepted ? 'accepted' : 'declined'}`);
    return { success: true, accepted };
  }

  async generateManualUpsell(userEmail, currentProductCategory, currentProductName) {
    if (!this._enabled) return null;
    const complementaryCategories = COMPLEMENTARY_MAP[currentProductCategory] || ['productivity', 'automation', 'analytics'];
    const existingInPool = this._activeProducts
      .filter(p => complementaryCategories.some(c => (p.category || '').toLowerCase() === c))
      .map(p => ({ name: p.productName, category: p.category, tagline: p.tagline }));
    const candidates = existingInPool.length > 0 ? existingInPool : complementaryCategories.map(c => {
      const tpl = PRODUCT_TEMPLATES[c] || PRODUCT_TEMPLATES.productivity;
      return { name: tpl.name, category: c, tagline: tpl.tagline };
    });
    if (candidates.length === 0) return null;
    const target = candidates[Math.floor(Math.random() * candidates.length)];
    strategist.setProduct({ productName: target.name, category: target.category, tagline: target.tagline });
    const pitch = strategist.craftConversionResponse(
      `I just started using ${currentProductName} and I love it. What else do you recommend?`,
      'interest'
    );
    this._upsellsSent++;
    return {
      productName: target.name,
      category: target.category,
      tagline: target.tagline,
      pitch,
      forUser: userEmail,
    };
  }

  getUpsellReport() {
    return {
      enabled: this._enabled,
      totalConversions: this._conversions.length,
      upsellsSent: this._upsellsSent,
      upsellSuccess: this._upsellSuccess,
      activeProducts: this._activeProducts.length,
      recoveryAttempts: this._recoveryAttempts.length,
      recoverySuccess: this._recoverySuccess,
      recoveryRate: this._recoveryAttempts.length > 0
        ? `${((this._recoverySuccess / this._recoveryAttempts.length) * 100).toFixed(1)}%`
        : '0%',
      recentConversions: this._conversions.slice(-5).map(c => ({
        email: c.email.replace(/(.{2}).*(@.*)/, '$1***$2'),
        product: c.productName,
        amount: c.amount,
        timestamp: new Date(c.timestamp).toISOString(),
      })),
    };
  }

  _loadState() {
    try {
      if (!existsSync(UPSELL_STATE_PATH)) return;
      const raw = readFileSync(UPSELL_STATE_PATH, 'utf-8');
      const data = JSON.parse(raw);
      this._conversions = data.conversions || [];
      this._upsellsSent = data.upsellsSent || 0;
      this._upsellSuccess = data.upsellSuccess || 0;
      this._activeProducts = data.activeProducts || [];
      this._recoveryAttempts = data.recoveryAttempts || [];
      this._recoverySuccess = data.recoverySuccess || 0;
      logger.info(`stripe-upsell: loaded ${this._conversions.length} conversions, ${this._upsellsSent} upsells, ${this._recoveryAttempts.length} recovery attempts`);
    } catch {}
  }

  _saveState() {
    try {
      writeFileSync(UPSELL_STATE_PATH, JSON.stringify({
        conversions: this._conversions.slice(-100),
        upsellsSent: this._upsellsSent,
        upsellSuccess: this._upsellSuccess,
        activeProducts: this._activeProducts,
        recoveryAttempts: this._recoveryAttempts.slice(-50),
        recoverySuccess: this._recoverySuccess,
      }, null, 2), { mode: 0o600 });
    } catch {}
  }

  _signPayload(payload) {
    const sig = createHmac('sha256', this._webhookSecret)
      .update(typeof payload === 'string' ? payload : JSON.stringify(payload))
      .digest('hex');
    return sig;
  }

  async _generateUpsell(conversion) {
    return this.generateManualUpsell(conversion.email, conversion.category, conversion.productName);
  }
}

const upsellEngine = new StripeUpsellEngine();
export default upsellEngine;
