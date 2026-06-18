import config from '../config.js';
import logger from '../utils/logger.js';
import { uid } from '../utils/helpers.js';
import stripe from '../monetization/stripe.js';

const tiers = JSON.parse(JSON.stringify(config.SUBSCRIPTION_TIERS || []));

export class SubscriptionManager {
  constructor() {
    this._store = new Map();
    this._usage = new Map();
    this._stripeMap = new Map();
  }

  createTier(name, limits) {
    const existing = tiers.find(t => t.name === name);
    if (existing) Object.assign(existing, limits);
    else tiers.push({ name, ...limits });
  }

  registerUser(userId, tier = 'free') {
    const tierCfg = tiers.find(t => t.name === tier) || tiers[0];
    const apiKey = uid(24);
    const record = {
      userId,
      tier,
      apiKey,
      adFree: tier !== 'free',
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      createdAt: new Date().toISOString(),
      limits: { ...tierCfg },
    };
    this._store.set(userId, record);
    this._usage.set(userId, { pages: 0, apiCalls: 0, periodStart: Date.now() });
    logger.info(`subscription: registered ${userId} on ${tier}`);
    return { ...record, apiKey };
  }

  getUser(userId) {
    const u = this._store.get(userId);
    return u ? { ...u } : null;
  }

  async checkAccess(userId, resource) {
    const user = this._store.get(userId);
    if (!user) return { allowed: false, reason: 'not_registered' };
    const usage = this._usage.get(userId);
    const tierCfg = tiers.find(t => t.name === user.tier) || tiers[0];
    this._resetIfNeeded(usage, tierCfg);

    if (resource === 'page' && usage.pages >= tierCfg.pages) {
      return { allowed: false, reason: 'page_limit_reached' };
    }
    return { allowed: true, tier: user.tier, adFree: user.adFree };
  }

  trackUsage(userId, resource) {
    const usage = this._usage.get(userId);
    if (!usage) return;
    this._resetIfNeeded(usage);
    if (resource === 'page') usage.pages++;
    if (resource === 'api') usage.apiCalls++;
  }

  _resetIfNeeded(usage) {
    const dayMs = 86400000;
    if (Date.now() - usage.periodStart > dayMs) {
      usage.pages = 0;
      usage.apiCalls = 0;
      usage.periodStart = Date.now();
    }
  }

  validateApiKey(key) {
    for (const [_, record] of this._store) {
      if (record.apiKey === key) return { ...record };
    }
    return null;
  }

  async upgradeToStripe(userId, priceId) {
    const user = this._store.get(userId);
    if (!user) throw new Error('User not found');

    const session = await stripe.createCheckoutSession({
      priceId,
      userId,
      successUrl: `https://emerald.app/success?uid=${userId}`,
      cancelUrl: `https://emerald.app/pricing`,
    });

    this._stripeMap.set(session.id, { userId, priceId, tier: priceId.includes('pro') ? 'pro' : 'starter' });
    return session;
  }

  async handleStripeWebhook(event) {
    if (event.type === 'checkout.session.completed') {
      const session = event.data.object;
      const mapping = this._stripeMap.get(session.id);
      if (mapping) {
        await this.upgradeUser(mapping.userId, mapping.tier, session.customer, session.subscription);
        this._stripeMap.delete(session.id);
      }
    }
    if (event.type === 'customer.subscription.deleted') {
      for (const [uid, record] of this._store) {
        if (record.stripeSubscriptionId === event.data.object.id) {
          await this.downgradeUser(uid);
          break;
        }
      }
    }
  }

  async upgradeUser(userId, tier, stripeCustomerId, stripeSubscriptionId) {
    const user = this._store.get(userId);
    if (!user) return;
    user.tier = tier;
    user.adFree = true;
    user.stripeCustomerId = stripeCustomerId || user.stripeCustomerId;
    user.stripeSubscriptionId = stripeSubscriptionId || user.stripeSubscriptionId;
    const tierCfg = tiers.find(t => t.name === tier) || tiers[0];
    user.limits = { ...tierCfg };
    logger.info(`subscription: upgraded ${userId} to ${tier}`);
  }

  async downgradeUser(userId) {
    const user = this._store.get(userId);
    if (!user) return;
    user.tier = 'free';
    user.adFree = false;
    user.stripeSubscriptionId = null;
    const tierCfg = tiers.find(t => t.name === 'free') || tiers[0];
    user.limits = { ...tierCfg };
    logger.info(`subscription: downgraded ${userId} to free`);
  }

  getAllUsers() {
    return Array.from(this._store.values()).map(u => ({
      userId: u.userId, tier: u.tier, createdAt: u.createdAt, adFree: u.adFree,
      hasStripe: !!u.stripeCustomerId,
    }));
  }
}

const subManager = new SubscriptionManager();
export default subManager;
