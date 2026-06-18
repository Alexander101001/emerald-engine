import fetch from 'node-fetch';
import logger from '../utils/logger.js';

const MEDIUM_API = 'https://api.medium.com/v1';
const DEVTO_API = 'https://dev.to/api';
const LINKEDIN_API = 'https://api.linkedin.com/v2';

export class MultiChannelPublisher {
  constructor() {
    this._published = [];
    this._failures = [];
  }

  getMediumToken() {
    return process.env.MEDIUM_API_KEY || '';
  }

  getDevtoToken() {
    return process.env.DEVTO_API_KEY || '';
  }

  getLinkedinToken() {
    return process.env.LINKEDIN_ACCESS_TOKEN || '';
  }

  async publishToMedium(article, product) {
    const token = this.getMediumToken();
    if (!token) {
      logger.info('publisher: MEDIUM_API_KEY not set — simulating Medium post');
      return this._simulate('medium', article.title, product);
    }

    try {
      const userRes = await fetch(`${MEDIUM_API}/me`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!userRes.ok) throw new Error(`Medium auth failed: ${userRes.status}`);
      const userData = await userRes.json();
      const authorId = userData.data.id;

      const res = await fetch(`${MEDIUM_API}/users/${authorId}/posts`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          title: article.title,
          contentFormat: 'markdown',
          content: article.mediumContent,
          tags: article.tags.slice(0, 5),
          publishStatus: 'public',
          canonicalUrl: product.url || '',
        }),
      });

      if (!res.ok) throw new Error(`Medium publish failed: ${res.status} ${await res.text()}`);
      const data = await res.json();
      const result = { platform: 'medium', title: article.title, url: data.data.url, success: true };
      this._published.push(result);
      logger.info(`publisher: published to Medium — ${data.data.url}`);
      return result;
    } catch (e) {
      logger.warn(`publisher: Medium publish failed — ${e.message}`);
      const result = { platform: 'medium', title: article.title, success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async publishToDevto(article, product) {
    const token = this.getDevtoToken();
    if (!token) {
      logger.info('publisher: DEVTO_API_KEY not set — simulating Dev.to post');
      return this._simulate('dev.to', article.title, product);
    }

    try {
      const res = await fetch(`${DEVTO_API}/articles`, {
        method: 'POST',
        headers: {
          'api-key': token,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          article: {
            title: article.title,
            published: true,
            body_markdown: article.devtoContent,
            tags: article.tags.slice(0, 4).map(t => t.toLowerCase().replace(/[^a-z0-9-]/g, '')),
            canonical_url: product.url || '',
            description: article.description,
          },
        }),
      });

      if (!res.ok) throw new Error(`Dev.to publish failed: ${res.status} ${await res.text()}`);
      const data = await res.json();
      const result = { platform: 'dev.to', title: article.title, url: data.url, success: true };
      this._published.push(result);
      logger.info(`publisher: published to Dev.to — ${data.url}`);
      return result;
    } catch (e) {
      logger.warn(`publisher: Dev.to publish failed — ${e.message}`);
      const result = { platform: 'dev.to', title: article.title, success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async publishToLinkedin(article, product) {
    const token = this.getLinkedinToken();
    if (!token) {
      logger.info('publisher: LINKEDIN_ACCESS_TOKEN not set — simulating LinkedIn post');
      return this._simulate('linkedin', article.title, product);
    }

    try {
      const res = await fetch(`${LINKEDIN_API}/ugcPosts`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'X-Restli-Protocol-Version': '2.0.0',
        },
        body: JSON.stringify({
          author: `urn:li:person:${process.env.LINKEDIN_PERSON_ID || '{person_id}'}`,
          lifecycleState: 'PUBLISHED',
          specificContent: {
            'com.linkedin.ugc.ShareContent': {
              shareCommentary: {
                text: article.linkedinContent.slice(0, 3000),
              },
              shareMediaCategory: 'ARTICLE',
            },
          },
          visibility: {
            'com.linkedin.ugc.MemberNetworkVisibility': 'PUBLIC',
          },
        }),
      });

      if (!res.ok) throw new Error(`LinkedIn publish failed: ${res.status} ${await res.text()}`);
      const data = await res.json();
      const result = { platform: 'linkedin', title: article.title, url: `https://linkedin.com/feed/update/${data.id}`, success: true };
      this._published.push(result);
      logger.info(`publisher: published to LinkedIn — ${data.id}`);
      return result;
    } catch (e) {
      logger.warn(`publisher: LinkedIn publish failed — ${e.message}`);
      const result = { platform: 'linkedin', title: article.title, success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async publishAll(articles, product) {
    const results = [];
    for (const article of articles) {
      const medium = await this.publishToMedium(article, product);
      const devto = await this.publishToDevto(article, product);
      const linkedin = await this.publishToLinkedin(article, product);
      results.push(medium, devto, linkedin);
    }
    return results;
  }

  _simulate(platform, title, product) {
    const slug = title.toLowerCase().replace(/[^a-z0-9]+/g, '-').slice(0, 40);
    const url = `https://${platform}.com/@${slugify(product.productName || 'emerald')}/${slug}-${Date.now()}`;
    const result = { platform, title, url, success: true, simulated: true };
    this._published.push(result);
    logger.info(`publisher: SIMULATED ${platform} post — ${url}`);
    return result;
  }

  getReport() {
    return {
      totalPublished: this._published.length,
      totalFailures: this._failures.length,
      published: this._published.slice(-20),
      failures: this._failures.slice(-10),
    };
  }

  reset() {
    this._published = [];
    this._failures = [];
  }
}

function slugify(text) {
  return text.toLowerCase().replace(/[^\w\s-]/g, '').replace(/[\s_]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 30) || 'page';
}

const publisher = new MultiChannelPublisher();
export default publisher;
