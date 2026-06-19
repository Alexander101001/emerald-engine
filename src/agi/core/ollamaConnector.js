import axios from 'axios';

const FALLBACK_PROVIDERS = [
  {
    name: 'Groq',
    url: 'https://api.groq.com/openai/v1/chat/completions',
    model: 'mixtral-8x7b-32768',
    key: () => process.env.GROQ_API_KEY,
  },
  {
    name: 'Together',
    url: 'https://api.together.xyz/v1/chat/completions',
    model: 'mistralai/Mixtral-8x7B-Instruct-v0.1',
    key: () => process.env.TOGETHER_API_KEY,
  },
  {
    name: 'OpenRouter',
    url: 'https://openrouter.ai/api/v1/chat/completions',
    model: 'mistralai/mixtral-8x7b-instruct',
    key: () => process.env.OPENROUTER_API_KEY,
  },
];

class OllamaConnector {
  constructor(endpoint = process.env.OLLAMA_ENDPOINT || 'http://localhost:11434') {
    this.endpoint = endpoint;
  }

  async generateThought(prompt, model) {
    const m = model || process.env.OLLAMA_MODEL || 'qwen2.5-coder:1.5b-instruct';
    try {
      const response = await axios.post(`${this.endpoint}/api/generate`, {
        model: m,
        prompt: prompt,
        stream: false,
      }, { timeout: 30000 });
      if (response.data?.response) return response.data.response;
    } catch (err) {
      console.warn(`Ollama offline (${err.message}) — trying fallback AI...`);
    }
    return this._fallbackGenerate(prompt);
  }

  async _fallbackGenerate(prompt) {
    for (const provider of FALLBACK_PROVIDERS) {
      const apiKey = provider.key();
      if (!apiKey) continue;
      try {
        console.log(`[AI] Fallback to ${provider.name}`);
        const res = await axios.post(provider.url, {
          model: provider.model,
          messages: [{ role: 'user', content: prompt }],
          max_tokens: 500,
        }, {
          headers: {
            'Authorization': `Bearer ${apiKey}`,
            'Content-Type': 'application/json',
          },
          timeout: 15000,
        });
        const text = res.data?.choices?.[0]?.message?.content;
        if (text) return text;
      } catch (err) {
        console.warn(`[AI] ${provider.name} failed: ${err.message}`);
      }
    }
    console.warn('[AI] All AI providers unavailable');
    return null;
  }
}

export default new OllamaConnector();
