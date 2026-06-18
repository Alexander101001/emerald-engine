import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const EVOLUTION_DIR = resolve(PROJECT_ROOT, '.evolution');
const LOG_FILE = resolve(EVOLUTION_DIR, 'social_evolution.json');
const OPTIMAL_PATTERNS_FILE = resolve(EVOLUTION_DIR, 'optimal_patterns.json');

class SocialEvolutionLog {
  constructor() {
    this._interactions = [];
    this._optimalPatterns = {
      greetings: [],
      empathy: [],
      conversions: [],
      tone: {},
      timing: {},
      keywords: [],
    };
    this._loaded = false;
  }

  load() {
    if (!existsSync(EVOLUTION_DIR)) {
      mkdirSync(EVOLUTION_DIR, { recursive: true, mode: 0o700 });
    }

    if (existsSync(LOG_FILE)) {
      try {
        const raw = readFileSync(LOG_FILE, 'utf-8');
        this._interactions = JSON.parse(raw);
      } catch (e) {
        logger.warn(`evolution: failed to load log — ${e.message}`);
        this._interactions = [];
      }
    }

    if (existsSync(OPTIMAL_PATTERNS_FILE)) {
      try {
        const raw = readFileSync(OPTIMAL_PATTERNS_FILE, 'utf-8');
        this._optimalPatterns = JSON.parse(raw);
      } catch {
        this._optimalPatterns = this._defaultPatterns();
      }
    } else {
      this._optimalPatterns = this._defaultPatterns();
    }

    this._loaded = true;
    logger.info(`evolution: loaded ${this._interactions.length} interactions, ${Object.keys(this._optimalPatterns.tone).length} tone patterns`);
  }

  _defaultPatterns() {
    return {
      greetings: ['Hey!', 'Thanks for reaching out.', 'Great to hear from you.'],
      empathy: ['I understand.', 'You make a great point.', 'That\'s totally fair.'],
      conversions: ['Want to see how it works?', 'Give it a try — 14 days free.', 'I can set you up with a demo.'],
      tone: { warmth: 0.7, expertise: 0.75, enthusiasm: 0.7, directness: 0.65 },
      timing: { responseDelay_ms: 3000, followUpHours: 24 },
      keywords: ['value', 'solve', 'help', 'try', 'demo', 'free', 'easy', 'fast'],
    };
  }

  recordInteraction(interaction) {
    const record = {
      id: `ev_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
      timestamp: new Date().toISOString(),
      ...interaction,
    };
    this._interactions.push(record);
    this._persist();
    return record;
  }

  tagOptimal(interactionId, conversionType) {
    const interaction = this._interactions.find(i => i.id === interactionId);
    if (!interaction) return false;

    interaction.optimal = true;
    interaction.optimalTaggedAt = new Date().toISOString();
    interaction.conversionType = conversionType || 'signup';

    this._extractPattern(interaction);
    this._persist();
    logger.info(`evolution: tagged interaction ${interactionId} as OPTIMAL (${conversionType})`);
    return true;
  }

  _extractPattern(interaction) {
    if (!interaction.responseText) return;

    const response = interaction.responseText.toLowerCase();
    const keywords = response.match(/\b\w{4,}\b/g) || [];

    for (const kw of keywords) {
      if (!this._optimalPatterns.keywords.includes(kw)) {
        this._optimalPatterns.keywords.push(kw);
      }
    }

    if (this._optimalPatterns.keywords.length > 100) {
      this._optimalPatterns.keywords = this._optimalPatterns.keywords.slice(-100);
    }

    if (response.includes('try') || response.includes('demo') || response.includes('free')) {
      this._optimalPatterns.conversions.push(interaction.responseText.slice(0, 200));
      if (this._optimalPatterns.conversions.length > 50) {
        this._optimalPatterns.conversions = this._optimalPatterns.conversions.slice(-50);
      }
    }

    if (interaction.sentiment) {
      const toneKey = `${interaction.sentiment}_optimal`;
      this._optimalPatterns.tone[toneKey] = (this._optimalPatterns.tone[toneKey] || 0) + 1;
    }
  }

  getOptimalPatterns() {
    return { ...this._optimalPatterns };
  }

  getOptimalResponses(sentiment) {
    const optimal = this._interactions.filter(i => i.optimal === true);
    if (sentiment) {
      return optimal.filter(i => i.sentiment === sentiment);
    }
    return optimal;
  }

  getStats() {
    const total = this._interactions.length;
    const optimal = this._interactions.filter(i => i.optimal).length;
    const bySentiment = {};
    for (const i of this._interactions) {
      if (i.sentiment) {
        bySentiment[i.sentiment] = (bySentiment[i.sentiment] || 0) + 1;
      }
    }
    return { totalInteractions: total, optimalCount: optimal, optimalRate: total > 0 ? optimal / total : 0, bySentiment };
  }

  getInteractionsSince(since) {
    const sinceDate = new Date(since).getTime();
    return this._interactions.filter(i => new Date(i.timestamp).getTime() > sinceDate);
  }

  _persist() {
    try {
      if (!existsSync(EVOLUTION_DIR)) {
        mkdirSync(EVOLUTION_DIR, { recursive: true, mode: 0o700 });
      }
      writeFileSync(LOG_FILE, JSON.stringify(this._interactions, null, 2), { mode: 0o600 });
      writeFileSync(OPTIMAL_PATTERNS_FILE, JSON.stringify(this._optimalPatterns, null, 2), { mode: 0o600 });
    } catch (e) {
      logger.error(`evolution: persist failed — ${e.message}`);
    }
  }
}

const evolutionLog = new SocialEvolutionLog();
export default evolutionLog;
