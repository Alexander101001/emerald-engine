import axios from 'axios';

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
      });
      return response.data.response;
    } catch (error) {
      console.error('Ollama connection error:', error.message);
      return null;
    }
  }
}

export default new OllamaConnector();
