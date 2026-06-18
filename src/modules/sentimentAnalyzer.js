import logger from '../utils/logger.js';

const FRUSTRATION_INDICATORS = [
  'frustrat', 'annoy', 'angry', 'upset', 'terrible', 'horrible', 'awful',
  'worst', 'hate', 'useless', 'broken', 'buggy', 'disappoint', 'waste',
  'ridiculous', 'absurd', 'unacceptable', 'pathetic', 'furious', 'irritate',
  'fed up', 'sick of', 'tired of', 'sucks', 'crap', 'terrible experience',
  'not working', 'doesn\'t work', 'failed', 'error', 'crash', 'slow',
  'why does', 'still not', 'you guys', "can't believe", 'uninstall',
  'refund', 'cancel', 'terrible support', 'no help', 'waste of money',
];

const INTEREST_INDICATORS = [
  'interesting', 'cool', 'nice', 'great', 'love', 'awesome', 'amazing',
  'impressive', 'wow', 'fantastic', 'excellent', 'brilliant', 'genius',
  'how does', 'tell me more', 'sign up', 'try', 'demo', 'pricing',
  'features', 'integration', 'api', 'works with', 'compatible',
  'get started', 'free trial', 'download', 'install', 'setup',
  'curious about', 'looking for', 'need a', 'want to', 'interested in',
  'this looks', 'promising', 'potential', 'solves', 'solution',
  'recommend', 'suggest', 'use case', 'example', 'show me',
];

const DOUBT_INDICATORS = [
  'not sure', 'maybe', 'perhaps', 'if it works', 'concerned about',
  'worried', 'skeptical', 'hesitant', 'uncertain', 'doubt', 'unsure',
  'is it worth', 'too expensive', 'expensive', 'overpriced', 'costs too much',
  'does it really', 'can it actually', 'prove it', 'show me proof',
  'case study', 'testimonials', 'reviews', 'competitors', 'alternative',
  'better than', 'switching from', 'migration', 'data privacy', 'security',
  'what about', 'how is this different', 'another tool', 'yet another',
  'free alternative', 'open source', 'self-hosted', 'on-premise',
];

const CURIOSITY_INDICATORS = [
  'what is', 'how to', 'can you', 'does it', 'is there', 'where can',
  'when will', 'who is', 'which one', 'why would', 'explain', 'help me understand',
  'question', 'query', 'wondering', 'thoughts on', 'opinion',
  'what do you think', 'suggestion', 'idea', 'feedback', 'feature request',
];

function containsAny(text, indicators) {
  const lower = text.toLowerCase();
  for (const indicator of indicators) {
    if (lower.includes(indicator)) return true;
  }
  return false;
}

export function analyzeSentiment(text) {
  if (!text || typeof text !== 'string') {
    return { primary: 'neutral', confidence: 1, signals: [] };
  }

  const signals = [];
  let frustration = 0;
  let interest = 0;
  let doubt = 0;
  let curiosity = 0;

  const words = text.toLowerCase().split(/\s+/);

  for (const word of words) {
    if (FRUSTRATION_INDICATORS.some(i => word.includes(i))) {
      frustration++;
      signals.push({ type: 'frustration', trigger: word });
    }
    if (INTEREST_INDICATORS.some(i => word.includes(i))) {
      interest++;
      signals.push({ type: 'interest', trigger: word });
    }
    if (DOUBT_INDICATORS.some(i => word.includes(i))) {
      doubt++;
      signals.push({ type: 'doubt', trigger: word });
    }
    if (CURIOSITY_INDICATORS.some(i => word.includes(i))) {
      curiosity++;
      signals.push({ type: 'curiosity', trigger: word });
    }
  }

  if (containsAny(text, FRUSTRATION_INDICATORS) && frustration === 0) {
    frustration++;
    signals.push({ type: 'frustration', trigger: 'phrase_match' });
  }
  if (containsAny(text, INTEREST_INDICATORS) && interest === 0) {
    interest++;
    signals.push({ type: 'interest', trigger: 'phrase_match' });
  }
  if (containsAny(text, DOUBT_INDICATORS) && doubt === 0) {
    doubt++;
    signals.push({ type: 'doubt', trigger: 'phrase_match' });
  }
  if (containsAny(text, CURIOSITY_INDICATORS) && curiosity === 0) {
    curiosity++;
    signals.push({ type: 'curiosity', trigger: 'phrase_match' });
  }

  const hasQuestionMark = text.includes('?');
  if (hasQuestionMark) {
    curiosity += 2;
    signals.push({ type: 'curiosity', trigger: 'question_mark' });
  }

  const hasExclamation = text.includes('!');
  const capsRatio = text.replace(/[^A-Z]/g, '').length / Math.max(text.length, 1);
  if (hasExclamation || capsRatio > 0.3) {
    frustration += capsRatio > 0.4 ? 2 : 0;
    signals.push({ type: 'intensity', trigger: capsRatio > 0.4 ? 'high_caps' : 'exclamation' });
  }

  const scores = { frustration, interest, doubt, curiosity };
  const primary = Object.entries(scores).sort((a, b) => b[1] - a[1])[0];
  const totalSignals = frustration + interest + doubt + curiosity;
  const confidence = totalSignals > 0 ? Math.min(primary[1] / totalSignals + 0.3, 1) : 0.3;

  const result = {
    primary: primary[0],
    confidence: Math.round(confidence * 100) / 100,
    signals: signals.slice(0, 10),
    scores,
    requiresEmpathy: frustration > 0 || doubt > 0,
    readyToConvert: interest > 2 || (interest > 0 && curiosity > 0),
    needsReassurance: doubt > 0 || (frustration > 0 && curiosity > 0),
  };

  return result;
}

export function classifyUserIntent(text) {
  const sentiment = analyzeSentiment(text);

  if (sentiment.primary === 'frustration') return 'support';
  if (sentiment.primary === 'curiosity' && sentiment.scores.interest > 0) return 'sales';
  if (sentiment.primary === 'interest') return 'sales';
  if (sentiment.primary === 'doubt') return 'objection';
  if (sentiment.scores.curiosity >= 2) return 'inquiry';

  return 'general';
}

const analyzer = {
  analyzeSentiment,
  classifyUserIntent,
};

export default analyzer;
