import fetch from 'node-fetch';
import logger from '../utils/logger.js';

const TREND_SOURCES = [
  { name: 'youtube_trending', url: 'https://www.googleapis.com/youtube/v3/videos', region: 'US', category: '28', key: process.env.YOUTUBE_API_KEY || '' },
  { name: 'tiktok_trending', url: 'https://open-api.tiktok.com/trending/', key: process.env.TIKTOK_API_KEY || '' },
  { name: 'youtube_music', url: 'https://www.googleapis.com/youtube/v3/videos', region: 'US', category: '10', key: process.env.YOUTUBE_API_KEY || '' },
];

const VIRAL_HASHTAGS = new Map([
  ['saas', ['#saas', '#software', '#startup', '#entrepreneur', '#businesstips', '#growthhack', '#productivityhack', '#b2b', '#techstartup', '#indiehacker']],
  ['productivity', ['#productivity', '#timemanagement', '#workflow', '#efficiency', '#hustle', '#morningroutine', '#focus', '#goalsetting', '#remotework', '#deepwork']],
  ['devtools', ['#devtools', '#coding', '#programming', '#developer', '#webdev', '#opensource', '#github', '#deploy', '#devops', '#javascript']],
  ['design', ['#designtools', '#uxdesign', '#uidesign', '#creativity', '#designthinking', '#figma', '#visualdesign', '#branding', '#art', '#creativetools']],
  ['marketing', ['#marketing', '#digitalmarketing', '#contentmarketing', '#seo', '#socialmediamarketing', '#emailmarketing', '#growthmarketing', '#branding', '#analytics', '#conversion']],
  ['general', ['#trending', '#viral', '#fyp', '#foryou', '#foryourpage', '#mustwatch', '#trendingnow', '#explore', '#discover', '#popular']],
]);

let fallbackTags = ['#saas', '#startup', '#productivity', '#technology', '#innovation', '#futureofwork', '#automation', '#digitaltransformation'];

const YOUTUBE_TRENDING_IDS = [
  'UCBR8-60_BkzRkFDh-_VHNsg',
  'UC-9-kyTW8ZkZNDHQJ6FgpwQ',
];

export class TrendHijacker {
  constructor() {
    this._trends = { sounds: [], hashtags: [], metadata: [] };
    this._lastScrape = 0;
    this._scrapeCount = 0;
    this._enabled = false;
  }

  activate() {
    this._enabled = true;
    logger.info('trend-hijacker: active — tracking YouTube/TikTok virality signals');
    return { active: true, sources: TREND_SOURCES.length };
  }

  deactivate() {
    this._enabled = false;
    logger.info('trend-hijacker: deactivated');
  }

  async scrape() {
    if (!this._enabled) return { error: 'not_active' };
    const results = [];
    for (const source of TREND_SOURCES) {
      try {
        const data = await this._fetchSource(source);
        if (data) results.push(data);
      } catch (e) {
        logger.warn(`trend-hijacker: ${source.name} scrape failed — ${e.message}`);
      }
    }
    const tags = this._mergeAndScore(results);
    this._trends = tags;
    this._lastScrape = Date.now();
    this._scrapeCount++;
    logger.info(`trend-hijacker: scraped ${tags.hashtags.length} viral tags, ${tags.sounds.length} trending sounds`);
    return tags;
  }

  injectMetadata(product) {
    if (!this._enabled) return { tags: fallbackTags, sounds: [] };
    if (Date.now() - this._lastScrape > 3600000) {
      this.scrape().catch(() => {});
    }
    const category = (product.category || 'general').toLowerCase();
    const baseTagPool = VIRAL_HASHTAGS.get(category) || VIRAL_HASHTAGS.get('general') || fallbackTags;
    const freshTags = this._trends.hashtags || [];
    const combined = [...new Set([...freshTags.slice(0, 5), ...baseTagPool.slice(0, 5)])].slice(0, 10);
    const avgDecay = this._calculateDecay();
    const result = {
      tags: combined,
      sounds: (this._trends.sounds || []).slice(0, 3),
      metadata: this._trends.metadata || [],
      decayFactor: avgDecay,
      scrapeAge: Math.floor((Date.now() - this._lastScrape) / 60000),
    };
    return result;
  }

  getTrendHijackerStatus() {
    return {
      enabled: this._enabled,
      lastScrape: this._lastScrape ? new Date(this._lastScrape).toISOString() : null,
      scrapesPerformed: this._scrapeCount,
      currentTags: (this._trends.hashtags || fallbackTags).slice(0, 5),
      currentSounds: (this._trends.sounds || []).slice(0, 3),
    };
  }

  async _fetchSource(source) {
    if (source.name.includes('youtube')) {
      const params = new URLSearchParams({
        part: 'snippet,statistics',
        chart: 'mostPopular',
        regionCode: source.region,
        videoCategoryId: source.category,
        maxResults: 10,
        key: source.key,
      });
      if (!source.key) {
        return this._simulateYouTube(source);
      }
      const res = await fetch(`${source.url}?${params}`);
      if (!res.ok) return this._simulateYouTube(source);
      const data = await res.json();
      if (!data.items) return this._simulateYouTube(source);
      const tags = new Set();
      const sounds = [];
      for (const item of data.items.slice(0, 5)) {
        if (item.snippet.tags) item.snippet.tags.slice(0, 5).forEach(t => tags.add(t.startsWith('#') ? t : `#${t.replace(/[^a-zA-Z0-9]/g, '')}`));
        sounds.push({
          title: item.snippet.title,
          channel: item.snippet.channelTitle,
          viewCount: item.statistics?.viewCount || '0',
        });
      }
      return { source: source.name, hashtags: [...tags].slice(0, 15), sounds: sounds.slice(0, 5), metadata: { region: source.region, category: source.category, itemCount: data.items.length } };
    }
    if (source.name.includes('tiktok')) {
      if (!source.key) return this._simulateTikTok();
      const res = await fetch(source.url, { headers: { 'Authorization': `Bearer ${source.key}` } });
      if (!res.ok) return this._simulateTikTok();
      const data = await res.json();
      return { source: source.name, hashtags: (data.hashtags || []).slice(0, 10), sounds: (data.sounds || []).slice(0, 5), metadata: data };
    }
    return null;
  }

  _simulateYouTube(source) {
    const trendingTopics = ['AI', 'automation', 'startup', 'productivity', 'coding', 'design', 'marketing', 'growth', 'SaaS', 'remote work'];
    const tags = trendingTopics.slice(0, 8).map(t => `#${t.replace(/\s+/g, '')}`);
    const sounds = trendingTopics.slice(0, 3).map((t, i) => ({ title: `Top ${t} trends ${new Date().getFullYear()}`, channel: `Trending${i}`, viewCount: String(Math.floor(Math.random() * 1000000) + 100000) }));
    return { source: source.name, hashtags: tags, sounds, metadata: { region: source.region, simulated: true } };
  }

  _simulateTikTok() {
    const trendingSounds = [
      { title: 'Original Sound - trending creator', author: 'viral_creator', usageCount: 250000 },
      { title: 'Epic Motivation Mix', author: 'audio_hub', usageCount: 180000 },
      { title: 'Tech Review BGM', author: 'production_music', usageCount: 120000 },
      { title: 'Startup Storytelling', author: 'narrative_audio', usageCount: 95000 },
      { title: 'Productivity Lo-Fi', author: 'chill_beats', usageCount: 78000 },
    ];
    return { source: 'tiktok_trending', hashtags: ['#fyp', '#viral', '#trending', '#startup', '#tech', '#productivity', '#foryou', '#mustwatch'], sounds: trendingSounds, metadata: { simulated: true } };
  }

  _mergeAndScore(results) {
    const allTags = new Set(fallbackTags);
    const allSounds = [];
    const allMetadata = [];
    for (const r of results) {
      if (r.hashtags) r.hashtags.forEach(t => allTags.add(t.startsWith('#') ? t : `#${t.replace(/[^a-zA-Z0-9]/g, '')}`));
      if (r.sounds) r.sounds.forEach(s => { if (!allSounds.find(ex => ex.title === s.title)) allSounds.push(s); });
      if (r.metadata) allMetadata.push(r.metadata);
    }
    return {
      hashtags: [...allTags].slice(0, 20),
      sounds: allSounds.slice(0, 10),
      metadata: allMetadata,
    };
  }

  _calculateDecay() {
    if (!this._lastScrape) return 1;
    return Math.max(0.1, 1 - (Date.now() - this._lastScrape) / 7200000);
  }
}

const trendHijacker = new TrendHijacker();
export default trendHijacker;
