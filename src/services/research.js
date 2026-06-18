import axios from 'axios';
import * as cheerio from 'cheerio';
import config from '../config.js';

const TREND_SOURCES = config.TREND_SOURCES;

async function fetchHackerNews() {
  try {
    const { data } = await axios.get('https://hacker-news.firebaseio.com/v0/topstories.json', {
      timeout: 10000,
    });
    const topIds = data.slice(0, 10);
    const stories = await Promise.all(
      topIds.map(id =>
        axios.get(`https://hacker-news.firebaseio.com/v0/item/${id}.json`, { timeout: 5000 })
          .then(r => r.data)
          .catch(() => null)
      )
    );
    return stories.filter(Boolean).map(s => ({
      source: 'hackernews',
      title: s.title,
      url: s.url || `https://news.ycombinator.com/item?id=${s.id}`,
      score: s.score || 0,
    }));
  } catch (err) {
    console.error('research: hn fetch failed', err.message);
    return [];
  }
}

async function fetchReddit() {
  try {
    const { data } = await axios.get('https://www.reddit.com/r/startups/hot.json?limit=10', {
      headers: { 'User-Agent': 'emerald-research/1.0' },
      timeout: 10000,
    });
    return (data.data?.children || []).map(child => ({
      source: 'reddit',
      title: child.data.title,
      url: `https://reddit.com${child.data.permalink}`,
      score: child.data.score || 0,
    }));
  } catch (err) {
    console.error('research: reddit fetch failed', err.message);
    return [];
  }
}

async function fetchProductHunt() {
  try {
    const { data } = await axios.get('https://www.producthunt.com/', {
      timeout: 10000,
    });
    const $ = cheerio.load(data);
    const posts = [];
    $('a[href*="/posts/"]').each((i, el) => {
      const title = $(el).text().trim();
      const url = 'https://www.producthunt.com' + $(el).attr('href');
      if (title && title.length > 2 && posts.length < 10) {
        posts.push({ source: 'producthunt', title, url, score: 0 });
      }
    });
    return posts;
  } catch (err) {
    console.error('research: producthunt fetch failed', err.message);
    return [];
  }
}

export async function researchMarketTrends() {
  console.log('research: scanning trend sources...');

  const [hn, reddit, ph] = await Promise.all([
    fetchHackerNews(),
    fetchReddit(),
    fetchProductHunt(),
  ]);

  const all = [...hn, ...reddit, ...ph];

  const keywords = ['saas', 'ai', 'startup', 'profit', 'monetize', 'revenue', 'growth', 'automation'];
  const filtered = all.filter(item =>
    keywords.some(kw => item.title.toLowerCase().includes(kw))
  );

  const results = filtered.length > 0 ? filtered : all.slice(0, 5);

  console.log(`research: found ${results.length} relevant trends`);
  return results;
}
