import fetch from 'node-fetch';
import * as cheerio from 'cheerio';
import logger from '../utils/logger.js';
import { cache } from '../utils/helpers.js';

const trendCache = cache(1800000);

const domainDelays = new Map();
const circuitBreakers = new Map();

const MIN_DELAY = 2000;
const MAX_DELAY = 15000;
const BACKOFF_FACTOR = 2;
const MAX_RETRIES = 3;
const CIRCUIT_RESET_MS = 300000;
const MAX_CONCURRENT = 2;

function sleep(ms) {
  return new Promise(r => setTimeout(r, ms));
}

function jitter(base) {
  return base + Math.random() * base * 0.5;
}

function getDomain(url) {
  try {
    const u = new URL(url);
    return u.hostname.replace(/^www\./, '');
  } catch {
    return 'unknown';
  }
}

function isCircuitOpen(domain) {
  const entry = circuitBreakers.get(domain);
  if (!entry) return false;
  if (Date.now() > entry.resetAt) {
    circuitBreakers.delete(domain);
    return false;
  }
  return true;
}

function recordFailure(domain) {
  const entry = circuitBreakers.get(domain) || { count: 0, resetAt: 0 };
  entry.count++;
  if (entry.count >= 3) {
    entry.resetAt = Date.now() + CIRCUIT_RESET_MS;
    logger.warn(`trendScraper: circuit opened for ${domain} (${CIRCUIT_RESET_MS}ms)`);
  }
  circuitBreakers.set(domain, entry);
}

function recordSuccess(domain) {
  circuitBreakers.delete(domain);
  domainDelays.set(domain, MIN_DELAY);
}

async function rateLimitedFetch(url, headers, retryCount = 0) {
  const domain = getDomain(url);

  if (isCircuitOpen(domain)) {
    throw new Error(`Circuit open for ${domain}, skipping`);
  }

  const lastDelay = domainDelays.get(domain) || MIN_DELAY;
  const delay = Math.min(jitter(lastDelay), MAX_DELAY);
  await sleep(delay);

  try {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 10000);

    const res = await fetch(url, {
      headers,
      signal: controller.signal,
      redirect: 'follow',
      follow: 3,
    });
    clearTimeout(timeout);

    if (res.status === 429) {
      const retryAfter = parseInt(res.headers.get('retry-after') || '5', 10) * 1000;
      const backoff = Math.min(retryAfter + jitter(1000), MAX_DELAY);
      domainDelays.set(domain, backoff);

      if (retryCount < MAX_RETRIES) {
        logger.warn(`trendScraper: 429 for ${domain}, retry ${retryCount + 1}/${MAX_RETRIES} after ${backoff}ms`);
        await sleep(backoff);
        return rateLimitedFetch(url, headers, retryCount + 1);
      }
      recordFailure(domain);
      throw new Error(`Rate limited (429) after ${MAX_RETRIES} retries`);
    }

    if (!res.ok) {
      recordFailure(domain);
      throw new Error(`HTTP ${res.status}`);
    }

    recordSuccess(domain);
    return res;
  } catch (e) {
    if (e.name === 'AbortError') {
      recordFailure(domain);
      throw new Error(`Request timeout for ${domain}`);
    }
    if (retryCount < MAX_RETRIES && !e.message.includes('Circuit open')) {
      const backoff = Math.min(jitter(lastDelay * BACKOFF_FACTOR), MAX_DELAY);
      domainDelays.set(domain, backoff);
      logger.warn(`trendScraper: retry ${retryCount + 1}/${MAX_RETRIES} for ${domain} after ${Math.round(backoff)}ms`);
      await sleep(backoff);
      return rateLimitedFetch(url, headers, retryCount + 1);
    }
    recordFailure(domain);
    throw e;
  }
}

export async function scrapeTrends(sources) {
  const cached = trendCache.get('trends');
  if (cached) return cached;

  const results = [];
  const active = (sources || []).filter(src => {
    const domain = getDomain(src);
    return !isCircuitOpen(domain);
  });

  for (let i = 0; i < active.length; i += MAX_CONCURRENT) {
    const batch = active.slice(i, i + MAX_CONCURRENT);
    const settled = await Promise.allSettled(
      batch.map(src => scrapeSource(src).catch(e => {
        logger.warn(`trendScraper: failed ${getDomain(src)} - ${e.message}`);
        return [];
      }))
    );
    for (const s of settled) {
      if (s.status === 'fulfilled') results.push(...s.value);
    }
    if (i + MAX_CONCURRENT < active.length) {
      await sleep(jitter(1000));
    }
  }

  const sorted = results.sort((a, b) => b.score - a.score).slice(0, 20);
  trendCache.set('trends', sorted);
  logger.info(`trendScraper: collected ${sorted.length} trends from ${active.length} sources`);
  return sorted;
}

async function scrapeSource(url) {
  const items = [];
  const headers = {
    'User-Agent': 'Mozilla/5.0 (compatible; EmeraldBot/1.0; +https://emerald.app/bot)',
    Accept: 'text/html,application/json,*/*',
    'Accept-Language': 'en-US,en;q=0.5',
    'Cache-Control': 'max-age=0',
  };

  if (url.endsWith('.json')) {
    const delay = jitter(1500);
    await sleep(delay);
    const res = await rateLimitedFetch(url, headers);
    const text = await res.text();
    let json;
    try {
      json = JSON.parse(text);
    } catch {
      throw new Error('Non-JSON response from reddit');
    }
    return parseReddit(json);
  }

  const res = await rateLimitedFetch(url, headers);
  const html = await res.text();
  const $ = cheerio.load(html);

  if (url.includes('news.ycombinator.com')) return parseHN($);
  if (url.includes('producthunt.com')) return parsePH($);

  return items;
}

function parseHN($) {
  const items = [];
  $('.athing').each((_, el) => {
    const titleEl = $(el).find('.titleline > a');
    const title = titleEl.text().trim();
    const link = titleEl.attr('href') || '';
    const scoreEl = $(el).next().find('.score');
    const score = parseInt(scoreEl.text()) || 1;
    if (title) items.push({ title, link, source: 'hackernews', score });
  });
  return items;
}

function parseReddit(json) {
  const items = [];
  const posts = json?.data?.children || [];
  for (const post of posts) {
    const d = post.data;
    if (d && d.title && !d.stickied) {
      items.push({
        title: d.title,
        link: `https://reddit.com${d.permalink}`,
        source: 'reddit',
        score: (d.ups || 1) + (d.num_comments || 0),
      });
    }
  }
  return items;
}

function parsePH($) {
  const items = [];
  $('a[data-test="post-name"]').each((_, el) => {
    const title = $(el).text().trim();
    const link = $(el).attr('href') || '';
    if (title) items.push({ title, link: `https://producthunt.com${link}`, source: 'producthunt', score: 10 });
  });
  return items;
}

export function getCircuitStatus() {
  const status = {};
  for (const [domain, entry] of circuitBreakers) {
    status[domain] = { open: Date.now() < entry.resetAt, failures: entry.count, resetsAt: new Date(entry.resetAt).toISOString() };
  }
  return status;
}

export function getDomainDelays() {
  const delays = {};
  for (const [domain, ms] of domainDelays) {
    delays[domain] = `${Math.round(ms)}ms`;
  }
  return delays;
}

export default scrapeTrends;
