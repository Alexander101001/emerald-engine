import fetch from 'node-fetch';
import logger from '../utils/logger.js';

const TWITTER_API = 'https://api.twitter.com/2';

export class SocialMediaAutomation {
  constructor() {
    this._posts = [];
    this._failures = [];
  }

  async postTwitterThread(thread, product) {
    const token = process.env.TWITTER_BEARER_TOKEN || '';
    if (!token) {
      logger.info('social: TWITTER_BEARER_TOKEN not set — simulating thread');
      return this._simulate('twitter', thread.articleTitle, product);
    }

    try {
      const postedTweets = [];
      for (let i = 0; i < thread.tweets.length; i++) {
        const body = { text: thread.tweets[i] };
        if (postedTweets.length > 0) {
          body.reply = { in_reply_to_tweet_id: postedTweets[postedTweets.length - 1] };
        }
        const res = await fetch(`${TWITTER_API}/tweets`, {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(body),
        });
        if (!res.ok) throw new Error(`Tweet ${i + 1} failed: ${res.status} ${await res.text()}`);
        const data = await res.json();
        postedTweets.push(data.data.id);
      }
      const result = { platform: 'twitter', thread: thread.articleTitle, tweetCount: thread.tweets.length, success: true };
      this._posts.push(result);
      logger.info(`social: posted Twitter thread (${thread.tweets.length} tweets) for "${thread.articleTitle}"`);
      return result;
    } catch (e) {
      logger.warn(`social: Twitter thread failed — ${e.message}`);
      const result = { platform: 'twitter', thread: thread.articleTitle, success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async postFacebook(content, product) {
    const token = process.env.FACEBOOK_PAGE_TOKEN || '';
    const pageId = process.env.FACEBOOK_PAGE_ID || '';
    if (!token || !pageId) {
      logger.info('social: FACEBOOK_PAGE_TOKEN not set — simulating Facebook post');
      return this._simulate('facebook', content.content.slice(0, 80), product);
    }

    try {
      const res = await fetch(`https://graph.facebook.com/v18.0/${pageId}/feed`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message: content.content,
          link: product.url || '',
          access_token: token,
        }),
      });
      if (!res.ok) throw new Error(`Facebook post failed: ${res.status} ${await res.text()}`);
      const data = await res.json();
      const result = { platform: 'facebook', postId: data.id, success: true };
      this._posts.push(result);
      logger.info(`social: posted to Facebook — ${data.id}`);
      return result;
    } catch (e) {
      logger.warn(`social: Facebook post failed — ${e.message}`);
      const result = { platform: 'facebook', success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async postInstagram(content, product) {
    const token = process.env.INSTAGRAM_BUSINESS_TOKEN || '';
    const accountId = process.env.INSTAGRAM_ACCOUNT_ID || '';
    if (!token || !accountId) {
      logger.info('social: INSTAGRAM_BUSINESS_TOKEN not set — simulating Instagram post');
      return this._simulate('instagram', content.content.slice(0, 80), product);
    }

    try {
      const mediaRes = await fetch(`https://graph.facebook.com/v18.0/${accountId}/media`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          image_url: content.imageUrl || `${product.url || ''}/og-image.png`,
          caption: content.content.slice(0, 2200),
          access_token: token,
        }),
      });
      if (!mediaRes.ok) throw new Error(`Instagram media failed: ${mediaRes.status} ${await mediaRes.text()}`);
      const mediaData = await mediaRes.json();

      const publishRes = await fetch(`https://graph.facebook.com/v18.0/${accountId}/media_publish`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ creation_id: mediaData.id, access_token: token }),
      });
      if (!publishRes.ok) throw new Error(`Instagram publish failed: ${publishRes.status} ${await publishRes.text()}`);
      const pubData = await publishRes.json();
      const result = { platform: 'instagram', mediaId: pubData.id, success: true };
      this._posts.push(result);
      logger.info(`social: posted to Instagram — ${pubData.id}`);
      return result;
    } catch (e) {
      logger.warn(`social: Instagram post failed — ${e.message}`);
      const result = { platform: 'instagram', success: false, error: e.message, simulated: true };
      this._failures.push(result);
      return result;
    }
  }

  async postYouTubeScript(script, product) {
    const token = process.env.YOUTUBE_API_KEY || '';
    if (!token) {
      logger.info('social: YOUTUBE_API_KEY not set — simulating YouTube video metadata');
      return this._simulate('youtube', script.title, product);
    }
    const result = {
      platform: 'youtube',
      title: script.title,
      description: script.script.slice(0, 5000),
      tags: script.tags,
      thumbnail: script.thumbnailSuggestion,
      success: true,
      note: 'Video file must be uploaded separately via YouTube Data API v3',
    };
    this._posts.push(result);
    logger.info(`social: YouTube metadata generated for "${script.title}"`);
    return result;
  }

  async publishAllSocial(threads, facebookPosts, instagramPosts, videoScripts, product) {
    const results = [];
    for (const thread of threads) {
      results.push(await this.postTwitterThread(thread, product));
    }
    for (const post of facebookPosts) {
      results.push(await this.postFacebook(post, product));
    }
    for (const post of instagramPosts) {
      results.push(await this.postInstagram(post, product));
    }
    for (const script of videoScripts) {
      results.push(await this.postYouTubeScript(script, product));
    }
    return results;
  }

  _simulate(platform, title, product) {
    const slug = title.toLowerCase().replace(/[^a-z0-9]+/g, '-').slice(0, 30);
    const url = `https://${platform}.com/${slugify(product.productName || 'emerald')}/${slug}`;
    const result = { platform, title: title.slice(0, 60), url, success: true, simulated: true };
    this._posts.push(result);
    logger.info(`social: SIMULATED ${platform} post — ${url}`);
    return result;
  }

  getReport() {
    return {
      totalPosts: this._posts.length,
      totalFailures: this._failures.length,
      posts: this._posts.slice(-25),
      failures: this._failures.slice(-10),
    };
  }

  reset() {
    this._posts = [];
    this._failures = [];
  }
}

function slugify(text) {
  return text.toLowerCase().replace(/[^\w\s-]/g, '').replace(/[\s_]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 30) || 'page';
}

const social = new SocialMediaAutomation();
export default social;
