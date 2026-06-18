import fetch from 'node-fetch';
import logger from '../utils/logger.js';

const REDDIT_API = 'https://oauth.reddit.com/api';
const QUORA_API = 'https://api.quora.com';
const PRODUCTHUNT_API = 'https://api.producthunt.com/v2/api/graphql';

export class TrafficDriver {
  constructor() {
    this._posts = [];
    this._failures = [];
  }

  async postToReddit(article, product) {
    const clientId = process.env.REDDIT_CLIENT_ID || '';
    const clientSecret = process.env.REDDIT_CLIENT_SECRET || '';
    const username = process.env.REDDIT_USERNAME || '';
    const password = process.env.REDDIT_PASSWORD || '';

    if (!clientId || !clientSecret) {
      logger.info('traffic: REDDIT credentials not set — simulating Reddit post');
      return this._simulate('reddit', article.title, product);
    }

    try {
      const authRes = await fetch('https://www.reddit.com/api/v1/access_token', {
        method: 'POST',
        headers: {
          Authorization: `Basic ${Buffer.from(`${clientId}:${clientSecret}`).toString('base64')}`,
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({
          grant_type: 'password',
          username,
          password,
        }),
      });
      if (!authRes.ok) throw new Error(`Reddit auth failed: ${authRes.status}`);
      const authData = await authRes.json();
      const token = authData.access_token;

      const subreddit = product.category === 'devtools' ? 'webdev' : product.category === 'marketing' ? 'startups' : 'SaaS';
      const res = await fetch(`${REDDIT_API}/submit`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'User-Agent': 'EmeraldAGI/1.0',
        },
        body: JSON.stringify({
          kind: 'link',
          sr: subreddit,
          title: article.title.slice(0, 300),
          url: product.url || '',
          resubmit: true,
        }),
      });
      if (!res.ok) throw new Error(`Reddit submit failed: ${res.status} ${await res.text()}`);
      const data = await res.json();
      const result = { platform: 'reddit', subreddit, title: article.title, success: true };
      this._posts.push(result);
      logger.info(`traffic: posted to r/${subreddit} — "${article.title.slice(0, 60)}"`);
      return result;
    } catch (e) {
      logger.warn(`traffic: Reddit post failed — ${e.message}`);
      const result = { platform: 'reddit', title: article.title, success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async postToQuora(article, product) {
    const token = process.env.QUORA_ACCESS_TOKEN || '';
    if (!token) {
      logger.info('traffic: QUORA_ACCESS_TOKEN not set — simulating Quora post');
      return this._simulate('quora', article.title, product);
    }

    try {
      const questionText = `What are the best ${product.category || 'SaaS'} solutions in ${new Date().getFullYear()}?`;
      const res = await fetch(`${QUORA_API}/questions`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          question: questionText,
          content: `${article.description}\n\nI've been using ${product.productName} and it's been great. Here's my detailed review: ${product.url || ''}`,
          topics: [product.category || 'SaaS', 'Technology', 'Software'],
        }),
      });
      if (!res.ok) throw new Error(`Quora post failed: ${res.status} ${await res.text()}`);
      const data = await res.json();
      const result = { platform: 'quora', question: questionText, success: true };
      this._posts.push(result);
      logger.info(`traffic: posted to Quora — "${questionText.slice(0, 60)}"`);
      return result;
    } catch (e) {
      logger.warn(`traffic: Quora post failed — ${e.message}`);
      const result = { platform: 'quora', title: article.title, success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async postToProductHunt(article, product) {
    const token = process.env.PRODUCTHUNT_TOKEN || '';
    if (!token) {
      logger.info('traffic: PRODUCTHUNT_TOKEN not set — simulating ProductHunt post');
      return this._simulate('producthunt', product.productName, product);
    }

    try {
      const query = `
        mutation {
          createPost(input: {
            name: "${product.productName}",
            tagline: "${(product.tagline || 'Built with Emerald AGI').replace(/"/g, '\\"')}",
            url: "${product.url || ''}",
            description: "${article.description.replace(/"/g, '\\"').slice(0, 260)}",
            topics: ["tech", "saas", "productivity"]
          }) {
            post { id name url }
          }
        }
      `;
      const res = await fetch(PRODUCTHUNT_API, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ query }),
      });
      if (!res.ok) throw new Error(`ProductHunt post failed: ${res.status} ${await res.text()}`);
      const data = await res.json();
      const result = { platform: 'producthunt', name: product.productName, success: true };
      this._posts.push(result);
      logger.info(`traffic: posted to ProductHunt — "${product.productName}"`);
      return result;
    } catch (e) {
      logger.warn(`traffic: ProductHunt post failed — ${e.message}`);
      const result = { platform: 'producthunt', name: product.productName, success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async driveAllTraffic(articles, product) {
    const results = [];
    for (const article of articles) {
      results.push(await this.postToReddit(article, product));
      results.push(await this.postToQuora(article, product));
    }
    results.push(await this.postToProductHunt(articles[0], product));
    return results;
  }

  _simulate(platform, title, product) {
    const slug = title.toLowerCase().replace(/[^a-z0-9]+/g, '-').slice(0, 30);
    const url = `https://${platform}.com/${slugify(product.productName || 'emerald')}/${slug}`;
    const result = { platform, title: title.slice(0, 60), url, success: true, simulated: true };
    this._posts.push(result);
    logger.info(`traffic: SIMULATED ${platform} post — ${url}`);
    return result;
  }

  getReport() {
    return {
      totalDrives: this._posts.length,
      totalFailures: this._failures.length,
      posts: this._posts.slice(-20),
      failures: this._failures.slice(-10),
    };
  }
}

function slugify(text) {
  return text.toLowerCase().replace(/[^\w\s-]/g, '').replace(/[\s_]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 30) || 'page';
}

const traffic = new TrafficDriver();
export default traffic;
