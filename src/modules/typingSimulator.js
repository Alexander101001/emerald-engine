import { randomBytes } from 'crypto';
import logger from '../utils/logger.js';

const CHARS_PER_SECOND = { min: 2, max: 7 };
const WORD_LENGTH_VARIANCE = { min: 3, max: 12 };
const SENTENCE_LENGTH_VARIANCE = { min: 5, max: 25 };
const PARAGRAPH_BREAK_PROBABILITY = 0.35;
const BACKSPACE_PROBABILITY = 0.08;
const PAUSE_AFTER_WORD_MS = { min: 80, max: 350 };
const PAUSE_AFTER_SENTENCE_MS = { min: 400, max: 1200 };
const PAUSE_AFTER_PARAGRAPH_MS = { min: 1200, max: 3000 };

export class TypingSimulator {
  constructor() {
    this._enabled = false;
  }

  activate() {
    this._enabled = true;
    logger.info('typing-simulator: active — natural composition delays enabled');
    return { active: true };
  }

  deactivate() {
    this._enabled = false;
    logger.info('typing-simulator: deactivated');
  }

  async simulateTyping(text, options = {}) {
    if (!this._enabled) return text;
    const totalDelay = this._calculateTotalDelay(text, options);
    const jittered = totalDelay + Math.floor(Math.random() * totalDelay * 0.3);
    await this._sleep(jittered);
    const varied = this._applyStructuralVariance(text, options);
    return varied;
  }

  async simulateComposition(text, onChar) {
    if (!this._enabled) return text;
    const chars = text.split('');
    for (let i = 0; i < chars.length; i++) {
      const cps = CHARS_PER_SECOND.min + Math.random() * (CHARS_PER_SECOND.max - CHARS_PER_SECOND.min);
      const charDelay = Math.floor(1000 / cps);
      const jitter = Math.floor(Math.random() * charDelay * 0.6);
      await this._sleep(charDelay + jitter);
      if (onChar) onChar(chars.slice(0, i + 1).join(''));
    }
    return text;
  }

  async simulateTypingWithMistakes(text) {
    if (!this._enabled) return text;
    let buffer = '';
    const chars = text.split('');
    for (let i = 0; i < chars.length; i++) {
      const cps = CHARS_PER_SECOND.min + Math.random() * (CHARS_PER_SECOND.max - CHARS_PER_SECOND.min);
      const charDelay = Math.floor(1000 / cps);
      await this._sleep(charDelay + Math.floor(Math.random() * 150));
      if (Math.random() < BACKSPACE_PROBABILITY && buffer.length > 2) {
        const backspaceCount = Math.floor(Math.random() * 3) + 1;
        buffer = buffer.slice(0, -backspaceCount);
        await this._sleep(200 + Math.floor(Math.random() * 300));
        for (let b = 0; b < backspaceCount; b++) {
          buffer += chars[Math.max(0, i - backspaceCount + b + 1)];
          await this._sleep(50 + Math.floor(Math.random() * 100));
        }
        continue;
      }
      buffer += chars[i];
    }
    return buffer;
  }

  getTimingStats(text) {
    const words = text.split(/\s+/).length;
    const chars = text.length;
    const sentences = text.split(/[.!?]+/).length;
    const paragraphs = text.split(/\n\s*\n/).length;
    const estimatedSeconds = this._calculateTotalDelay(text, {}) / 1000;
    return { words, chars, sentences, paragraphs, estimatedSeconds };
  }

  _calculateTotalDelay(text, options) {
    const words = text.split(/\s+/);
    let totalMs = 0;
    for (const word of words) {
      const cps = CHARS_PER_SECOND.min + Math.random() * (CHARS_PER_SECOND.max - CHARS_PER_SECOND.min);
      totalMs += (word.length / cps) * 1000;
      totalMs += PAUSE_AFTER_WORD_MS.min + Math.random() * (PAUSE_AFTER_WORD_MS.max - PAUSE_AFTER_WORD_MS.min);
      if (word.endsWith('.') || word.endsWith('!') || word.endsWith('?')) {
        totalMs += PAUSE_AFTER_SENTENCE_MS.min + Math.random() * (PAUSE_AFTER_SENTENCE_MS.max - PAUSE_AFTER_SENTENCE_MS.min);
      }
    }
    const paragraphCount = text.split(/\n\s*\n/).length - 1;
    totalMs += paragraphCount * (PAUSE_AFTER_PARAGRAPH_MS.min + Math.random() * (PAUSE_AFTER_PARAGRAPH_MS.max - PAUSE_AFTER_PARAGRAPH_MS.min));
    return totalMs;
  }

  _applyStructuralVariance(text, options) {
    const sentences = text.match(/[^.!?]*[.!?]/g) || [text];
    const varied = [];
    for (const sentence of sentences) {
      if (Math.random() < PARAGRAPH_BREAK_PROBABILITY && varied.length > 0) {
        varied.push('\n\n');
      }
      const trimmed = sentence.trim();
      if (Math.random() < 0.15 && trimmed.length > 30) {
        const splitPoint = Math.floor(trimmed.length * (0.3 + Math.random() * 0.4));
        varied.push(trimmed.slice(0, splitPoint));
        varied.push(' ');
        varied.push(trimmed.slice(splitPoint));
      } else {
        varied.push(trimmed);
      }
      varied.push(' ');
    }
    return varied.join('').trim();
  }

  _sleep(ms) {
    return new Promise(r => setTimeout(r, ms));
  }
}

const typingSimulator = new TypingSimulator();
export default typingSimulator;
