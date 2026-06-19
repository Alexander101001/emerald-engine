import { SearchEngine } from './SearchEngine.js';
import fs from 'fs';

const searcher = new SearchEngine(process.env.SERPER_API_KEY || '');

const RESEARCH_TOPICS = [
  "trending SaaS products 2026",
  "AI tools making money online",
  "profitable affiliate niches",
  "crypto trading bot opportunities",
  "side hustle automation tools"
];

export async function runMarketResearch() {
  console.log("[RESEARCHER] Scanning real market data...");
  const allResults = [];

  for (const topic of RESEARCH_TOPICS) {
    const results = await searcher.search(topic);
    allResults.push(...results);
  }

  const opportunities = allResults.map(r => ({
    title: r.title,
    url: r.link,
    snippet: r.snippet,
    source: 'web',
    timestamp: new Date().toISOString(),
  }));

  fs.writeFileSync('opportunities.json', JSON.stringify(opportunities, null, 2));
  console.log(`[RESEARCHER] Saved ${opportunities.length} real market opportunities`);
  return opportunities;
}
