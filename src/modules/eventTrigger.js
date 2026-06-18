import logger from '../utils/logger.js';

const MARKETING_ACTIONS = {
  social_reply: { priority: 10, cooldown: 300000, handler: 'handleSocialReply' },
  content_publish: { priority: 8, cooldown: 600000, handler: 'publishContent' },
  outreach: { priority: 7, cooldown: 1800000, handler: 'sendOutreach' },
  conversion: { priority: 9, cooldown: 60000, handler: 'triggerConversion' },
  trend_jack: { priority: 6, cooldown: 900000, handler: 'jackTrend' },
};

class EventTrigger {
  constructor() {
    this._listeners = new Map();
    this._cooldowns = new Map();
    this._eventsFired = 0;
    this._eventsQueued = 0;
    this._lastFlush = Date.now();
    this._enabled = false;
    this._queue = [];
    this._flushTimers = new Map();
  }

  activate() {
    this._enabled = true;
    for (const [action, cfg] of Object.entries(MARKETING_ACTIONS)) {
      this._registerAction(action, cfg);
    }
    logger.info(`event-trigger: active — ${Object.keys(MARKETING_ACTIONS).length} marketing actions registered`);
    return { active: true, actions: Object.keys(MARKETING_ACTIONS) };
  }

  deactivate() {
    this._enabled = false;
    for (const timer of this._flushTimers.values()) clearTimeout(timer);
    this._flushTimers.clear();
    logger.info('event-trigger: deactivated');
  }

  fire(eventType, payload = {}) {
    if (!this._enabled) return false;
    const now = Date.now();
    const lastFired = this._cooldowns.get(eventType) || 0;
    const action = MARKETING_ACTIONS[eventType];
    if (action && now - lastFired < action.cooldown) {
      this._eventsQueued++;
      this._queue.push({ eventType, payload, timestamp: now });
      return { fired: false, queued: true, reason: 'cooldown' };
    }
    this._cooldowns.set(eventType, now);
    this._eventsFired++;
    logger.info(`event-trigger: fired "${eventType}" — ${this._eventsFired} total`);
    if (!this._flushTimers.has(eventType)) {
      this._flushTimers.set(eventType, setTimeout(() => {
        this._flushQueued(eventType);
        this._flushTimers.delete(eventType);
      }, action?.cooldown || 600000));
    }
    return { fired: true, eventType, payload };
  }

  onSentimentPositive(sentimentResult, context = {}) {
    if (!this._enabled || !sentimentResult) return null;
    const confidence = sentimentResult.confidence || 0;
    if (confidence < 0.5) return null;
    const immediate = sentimentResult.primary === 'interest' || sentimentResult.primary === 'gratitude';
    if (immediate && confidence > 0.6) {
      return this.fire('conversion', {
        sentiment: sentimentResult,
        source: context.source || 'telegram',
        userMessage: context.message || '',
        userId: context.userId,
      });
    }
    if (sentimentResult.primary === 'interest' && sentimentResult.readyToConvert) {
      return this.fire('outreach', {
        sentiment: sentimentResult,
        source: context.source || 'telegram',
        userMessage: context.message || '',
      });
    }
    return null;
  }

  onTrendSpike(trend, score) {
    if (!this._enabled || !trend || score < 7) return null;
    return this.fire('trend_jack', {
      trendName: trend.title || trend.name || 'unknown',
      category: trend.category || 'general',
      score,
      description: (trend.description || '').slice(0, 200),
    });
  }

  onContentReady(content, platform) {
    if (!this._enabled || !content) return null;
    return this.fire('content_publish', {
      content: content.slice(0, 500),
      platform: platform || 'multi',
    });
  }

  flush() {
    const now = Date.now();
    const batch = [];
    for (const [eventType, cfg] of Object.entries(MARKETING_ACTIONS)) {
      const lastFired = this._cooldowns.get(eventType) || 0;
      if (now - lastFired >= cfg.cooldown) {
        const queued = this._queue.filter(q => q.eventType === eventType);
        if (queued.length > 0) {
          batch.push({ eventType, count: queued.length, actions: queued });
          this._queue = this._queue.filter(q => q.eventType !== eventType);
        }
      }
    }
    if (batch.length > 0) {
      logger.info(`event-trigger: flushed ${batch.length} event types from queue`);
    }
    this._lastFlush = now;
    return batch;
  }

  getTriggerStatus() {
    return {
      enabled: this._enabled,
      eventsFired: this._eventsFired,
      eventsQueued: this._eventsQueued,
      queueDepth: this._queue.length,
      actions: Object.entries(MARKETING_ACTIONS).map(([k, v]) => ({
        type: k,
        priority: v.priority,
        cooldownMs: v.cooldown,
      })),
    };
  }

  _registerAction(action, config) {
    this._listeners.set(action, config);
  }

  _flushQueued(eventType) {
    const queued = this._queue.filter(q => q.eventType === eventType);
    if (queued.length === 0) return;
    this._queue = this._queue.filter(q => q.eventType !== eventType);
    logger.info(`event-trigger: flushed ${queued.length} queued "${eventType}" events`);
  }
}

const eventTrigger = new EventTrigger();
export default eventTrigger;
