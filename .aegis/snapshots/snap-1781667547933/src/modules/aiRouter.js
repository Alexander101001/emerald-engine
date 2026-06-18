import fetch from 'node-fetch';
import logger from '../utils/logger.js';
import config from '../config.js';

const PROVIDERS = [
  {
    name: 'openai',
    base: 'https://api.openai.com/v1/chat/completions',
    key: () => config.OPENAI_API_KEY,
    buildBody: (prompt) => ({
      model: 'gpt-3.5-turbo',
      messages: [{ role: 'user', content: prompt }],
      max_tokens: +config.MAX_GENERATION_TOKENS,
    }),
    parse: (json) => json.choices?.[0]?.message?.content || '',
  },
  {
    name: 'openrouter',
    base: 'https://openrouter.ai/api/v1/chat/completions',
    key: () => config.OPENROUTER_API_KEY || '',
    buildBody: (prompt) => ({
      model: 'openai/gpt-3.5-turbo',
      messages: [{ role: 'user', content: prompt }],
      max_tokens: +config.MAX_GENERATION_TOKENS,
    }),
    parse: (json) => json.choices?.[0]?.message?.content || '',
  },
  {
    name: 'together',
    base: 'https://api.together.xyz/v1/chat/completions',
    key: () => config.TOGETHER_API_KEY || '',
    buildBody: (prompt) => ({
      model: 'mistralai/Mixtral-8x7B-Instruct-v0.1',
      messages: [{ role: 'user', content: prompt }],
      max_tokens: +config.MAX_GENERATION_TOKENS,
    }),
    parse: (json) => json.choices?.[0]?.message?.content || '',
  },
];

export class FailoverRouter {
  constructor() {
    this._failures = new Map();
    this._circuitBreaker = new Map();
    this._fallbackProviders = PROVIDERS.filter(p => p.key());
  }

  async generate(prompt, options = {}) {
    if (this._fallbackProviders.length === 0 && !options.allowFallbackMock) {
      return this._mockGenerate(prompt);
    }

    const candidates = this._fallbackProviders.length > 0
      ? this._fallbackProviders
      : PROVIDERS;

    const errors = [];
    for (const provider of candidates) {
      if (this._isOpen(provider.name)) {
        try {
          const result = await this._call(provider, prompt);
          this._recordSuccess(provider.name);
          return { provider: provider.name, content: result };
        } catch (e) {
          this._recordFailure(provider.name);
          errors.push({ provider: provider.name, error: e.message });
          logger.warn(`aiRouter: ${provider.name} failed - ${e.message}`);
        }
      }
    }

    logger.error('aiRouter: all providers failed', errors);
    if (options.allowFallbackMock) {
      return { provider: 'mock', content: this._mockGenerate(prompt) };
    }
    throw new Error(`AI providers exhausted: ${errors.map(e => e.provider).join(', ')}`);
  }

  async _call(provider, prompt) {
    const key = provider.key();
    if (!key) throw new Error('No API key configured');
    const res = await fetch(provider.base, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${key}`,
      },
      body: JSON.stringify(provider.buildBody(prompt)),
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`${res.status}: ${text.slice(0, 200)}`);
    }
    const json = await res.json();
    return provider.parse(json);
  }

  _isOpen(name) {
    const until = this._circuitBreaker.get(name);
    if (!until) return true;
    if (Date.now() > until) { this._circuitBreaker.delete(name); return true; }
    return false;
  }

  _recordFailure(name) {
    const count = (this._failures.get(name) || 0) + 1;
    this._failures.set(name, count);
    if (count >= 3) {
      this._circuitBreaker.set(name, Date.now() + 60000);
      this._failures.delete(name);
      logger.warn(`aiRouter: circuit opened for ${name} (60s)`);
    }
  }

  _recordSuccess(name) {
    this._failures.delete(name);
  }

  _mockGenerate(prompt) {
    const keywords = prompt.toLowerCase();
    let topic = 'your-saas';
    if (keywords.includes('saas') || keywords.includes('app')) topic = 'smart-saas';
    else if (keywords.includes('tool') || keywords.includes('utility')) topic = 'handy-tool';
    else if (keywords.includes('service') || keywords.includes('platform')) topic = 'premium-service';

    return JSON.stringify({
      title: `${topic.charAt(0).toUpperCase() + topic.slice(1).replace(/-/g, ' ')}`,
      tagline: `The next-gen ${topic.replace(/-/g, ' ')} built for modern teams.`,
      features: ['Real-time sync', 'Zero configuration', 'Team collaboration', 'API-first design'],
      cta: 'Start Free Trial',
    });
  }
}

const router = new FailoverRouter();
export default router;
