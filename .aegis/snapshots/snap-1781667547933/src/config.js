import { readFileSync, existsSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = resolve(__dirname, '..');

const defaults = {
  PORT: 3000,
  NODE_ENV: 'development',
  TELEGRAM_BOT_TOKEN: '',
  TELEGRAM_BOT_ID: '',
  TELEGRAM_CONFIG: '',
  OPENAI_API_KEY: '',
  OPENROUTER_API_KEY: '',
  TOGETHER_API_KEY: '',
  GITHUB_PAT: '',
  GITHUB_OWNER: '',
  SUPABASE_URL: '',
  SUPABASE_KEY: '',
  AD_SLOT_COUNT: 3,
  TREND_SOURCES: [
    'https://news.ycombinator.com',
    'https://www.reddit.com/r/startups/.json',
    'https://www.producthunt.com',
  ],
  DEPLOY_PLATFORM: process.env.VERCEL ? 'vercel' : process.env.NETLIFY ? 'netlify' : 'local',
  CACHE_TTL: 3600000,
  MAX_GENERATION_TOKENS: 800,
  SEO_CHECK_INTERVAL: 86400000,
  SUBSCRIPTION_TIERS: [
    { name: 'free', pages: 3, analytics: false, customDomain: false },
    { name: 'starter', pages: 10, analytics: true, customDomain: false },
    { name: 'pro', pages: 50, analytics: true, customDomain: true },
  ],
};

export function loadConfig() {
  const envPath = resolve(root, '.env');
  if (existsSync(envPath)) {
    const lines = readFileSync(envPath, 'utf-8').split('\n');
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) continue;
      const idx = trimmed.indexOf('=');
      if (idx === -1) continue;
      const key = trimmed.slice(0, idx).trim();
      const val = trimmed.slice(idx + 1).trim();
      if (!process.env[key]) process.env[key] = val;
    }
  }

  const config = {};
  for (const [key, val] of Object.entries(defaults)) {
    config[key] = process.env[key] || val;
  }
  return config;
}

const config = loadConfig();
export default config;
