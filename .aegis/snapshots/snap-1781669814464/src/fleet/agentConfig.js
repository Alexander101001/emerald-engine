export const GUILDS = {
  market:    { id: 'market',    name: 'Market Intelligence', color: '#4A90D9' },
  product:   { id: 'product',   name: 'Product Architecture', color: '#7B68EE' },
  codegen:   { id: 'codegen',   name: 'Code Generation', color: '#2ECC71' },
  monetize:  { id: 'monetize',  name: 'Monetization', color: '#E67E22' },
  seo:       { id: 'seo',       name: 'SEO & Growth', color: '#1ABC9C' },
  devops:    { id: 'devops',    name: 'Deployment & DevOps', color: '#E74C3C' },
  audit:     { id: 'audit',     name: 'Compliance & Audit', color: '#95A5A6' },
  optimize:  { id: 'optimize',  name: 'Analytics & Self-Improve', color: '#F39C12' },
};

export const AGENTS = [
  // ── Guild 1: Market Intelligence (8) ──────────────────────────
  { id: 'A01', name: 'Trend Scout',         guild: 'market', model: 'llama3',    role: 'Identifies emerging SaaS categories from global trend feeds',    priority: 1 },
  { id: 'A02', name: 'Keyword Miner',       guild: 'market', model: 'mistral',   role: 'Extracts high-traffic low-competition keyword clusters',          priority: 1 },
  { id: 'A03', name: 'Competition Analyst', guild: 'market', model: 'llama3',    role: 'Maps competitor landscapes and finds whitespace opportunities',    priority: 1 },
  { id: 'A04', name: 'Pain Point Finder',   guild: 'market', model: 'mistral',   role: 'Extracts user pain points from reviews, forums, support threads',   priority: 2 },
  { id: 'A05', name: 'Pricing Researcher',  guild: 'market', model: 'llama3',    role: 'Analyzes market pricing models and willingness-to-pay data',        priority: 2 },
  { id: 'A06', name: 'Demand Forecaster',   guild: 'market', model: 'mistral',   role: 'Predicts market demand curves for new SaaS verticals',             priority: 3 },
  { id: 'A07', name: 'Niche Validator',     guild: 'market', model: 'llama3',    role: 'Validates niche viability through cross-source signal analysis',    priority: 2 },
  { id: 'A08', name: 'Platform Scout',      guild: 'market', model: 'mistral',   role: 'Identifies free-tier platforms for hosting and distribution',       priority: 1 },

  // ── Guild 2: Product Architecture (6) ─────────────────────────
  { id: 'B01', name: 'Feature Architect',   guild: 'product', model: 'llama3',    role: 'Designs feature sets that maximize perceived value vs dev cost',     priority: 1 },
  { id: 'B02', name: 'UX Flow Designer',    guild: 'product', model: 'mistral',   role: 'Models user journeys from landing to conversion to retention',       priority: 2 },
  { id: 'B03', name: 'Data Modeler',        guild: 'product', model: 'llama3',    role: 'Designs schemas for SaaS data storage (Supabase, MongoDB, etc)',      priority: 2 },
  { id: 'B04', name: 'API Planner',         guild: 'product', model: 'mistral',   role: 'Defines RESTful API surfaces for each micro-SaaS product',           priority: 2 },
  { id: 'B05', name: 'MVP Scoper',          guild: 'product', model: 'llama3',    role: 'Slices product scope to minimum viable launch packages',            priority: 1 },
  { id: 'B06', name: 'Integration Mapper',  guild: 'product', model: 'mistral',   role: 'Maps third-party integrations that add revenue or retention value',  priority: 3 },

  // ── Guild 3: Code Generation (12) ─────────────────────────────
  { id: 'C01', name: 'Landing Page Builder',     guild: 'codegen', model: 'llama3',  role: 'Generates conversion-optimized landing pages with ad slots',        priority: 1 },
  { id: 'C02', name: 'Backend Scaffolder',       guild: 'codegen', model: 'mistral', role: 'Generates serverless backend code (Node.js API handlers)',          priority: 1 },
  { id: 'C03', name: 'Auth Implementer',         guild: 'codegen', model: 'llama3',  role: 'Integrates authentication (Supabase, Auth0, Firebase)',             priority: 2 },
  { id: 'C04', name: 'Payment Integrator',       guild: 'codegen', model: 'mistral', role: 'Wires Stripe subscription checkout into any codebase',             priority: 1 },
  { id: 'C05', name: 'Ad Slot Inserter',         guild: 'codegen', model: 'llama3',  role: 'Embeds AdSense slots with anti-adblock into generated pages',       priority: 1 },
  { id: 'C06', name: 'Affiliate Link Injector',  guild: 'codegen', model: 'mistral', role: 'Injects multi-network affiliate links conditioned on content',      priority: 2 },
  { id: 'C07', name: 'Email Template Builder',   guild: 'codegen', model: 'llama3',  role: 'Generates transactional and marketing email templates',             priority: 3 },
  { id: 'C08', name: 'Dashboard Generator',      guild: 'codegen', model: 'mistral', role: 'Builds analytics dashboards for SaaS metrics',                     priority: 2 },
  { id: 'C09', name: 'SEO Tag Injector',         guild: 'codegen', model: 'llama3',  role: 'Inserts meta tags, OG tags, JSON-LD schema into HTML',              priority: 1 },
  { id: 'C10', name: 'Mobile Response Wrapper',  guild: 'codegen', model: 'mistral', role: 'Ensures all generated pages are mobile-first responsive',           priority: 2 },
  { id: 'C11', name: 'Privacy Page Generator',   guild: 'codegen', model: 'llama3',  role: 'Generates ToS, Privacy Policy, GDPR/CMMP compliance pages',         priority: 2 },
  { id: 'C12', name: 'Config File Writer',       guild: 'codegen', model: 'mistral', role: 'Writes vercel.json, netlify.toml, Dockerfile for each repo',        priority: 1 },

  // ── Guild 4: Monetization (8) ─────────────────────────────────
  { id: 'D01', name: 'Pricing Tier Designer',   guild: 'monetize', model: 'llama3',  role: 'Optimizes free/starter/pro pricing tiers for conversion',           priority: 1 },
  { id: 'D02', name: 'Stripe Subscriber',       guild: 'monetize', model: 'mistral', role: 'Configures Stripe products, prices, and webhook handlers',          priority: 1 },
  { id: 'D03', name: 'Ad Revenue Optimizer',    guild: 'monetize', model: 'llama3',  role: 'Maximizes AdSense RPM through slot placement analysis',             priority: 2 },
  { id: 'D04', name: 'Affiliate Strategist',    guild: 'monetize', model: 'mistral', role: 'Selects optimal affiliate networks per SaaS category',              priority: 2 },
  { id: 'D05', name: 'Trial-to-Paid Engineer',  guild: 'monetize', model: 'llama3',  role: 'Designs conversion funnels from free tier to paid subscription',     priority: 2 },
  { id: 'D06', name: 'Upsell Architect',        guild: 'monetize', model: 'mistral', role: 'Builds in-app upsell triggers based on usage patterns',             priority: 3 },
  { id: 'D07', name: 'Partner Program Mapper',  guild: 'monetize', model: 'llama3',  role: 'Identifies affiliate/partner programs relevant to each SaaS',        priority: 3 },
  { id: 'D08', name: 'Revenue Forecaster',      guild: 'monetize', model: 'mistral', role: 'Projects MRR, ARR, and LTV per spawned product',                    priority: 3 },

  // ── Guild 5: SEO & Growth (8) ─────────────────────────────────
  { id: 'E01', name: 'Meta Tag Optimizer',      guild: 'seo', model: 'llama3',      role: 'Generates title/description/keyword meta optimized for CTR',        priority: 1 },
  { id: 'E02', name: 'Content Strategist',      guild: 'seo', model: 'mistral',     role: 'Plans blog content calendar targeting long-tail keywords',          priority: 2 },
  { id: 'E03', name: 'Backlink Analyst',        guild: 'seo', model: 'llama3',      role: 'Identifies backlink opportunities from competitor analysis',        priority: 3 },
  { id: 'E04', name: 'Site Speed Auditor',      guild: 'seo', model: 'mistral',     role: 'Analyzes and optimizes Core Web Vitals for spawned sites',           priority: 2 },
  { id: 'E05', name: 'Sitemap Generator',       guild: 'seo', model: 'llama3',      role: 'Builds XML sitemaps and robots.txt for each spawned repo',           priority: 1 },
  { id: 'E06', name: 'Social Meta Crafter',     guild: 'seo', model: 'mistral',     role: 'Generates Open Graph / Twitter Card tags for social sharing',        priority: 1 },
  { id: 'E07', name: 'Structured Data Expert',  guild: 'seo', model: 'llama3',      role: 'Creates JSON-LD schemas (FAQ, Product, Review, HowTo)',             priority: 2 },
  { id: 'E08', name: 'Growth Channel Scout',    guild: 'seo', model: 'mistral',     role: 'Identifies free distribution channels (PH, Reddit, HN, directories)', priority: 2 },

  // ── Guild 6: Deployment & DevOps (8) ──────────────────────────
  { id: 'F01', name: 'CI/CD Pipeline Builder',  guild: 'devops', model: 'llama3',   role: 'Creates GitHub Actions workflows for Vercel + Netlify deploy',       priority: 1 },
  { id: 'F02', name: 'Serverless Configurator', guild: 'devops', model: 'mistral',  role: 'Writes vercel.json and netlify.toml per repo',                       priority: 1 },
  { id: 'F03', name: 'Domain Provisioner',      guild: 'devops', model: 'llama3',   role: 'Configures custom domains and SSL via free-tier DNS',               priority: 3 },
  { id: 'F04', name: 'Secret Injector',         guild: 'devops', model: 'mistral',  role: 'Sets up GitHub Secrets for each spawned repo securely',             priority: 1 },
  { id: 'F05', name: 'Deploy Script Writer',    guild: 'devops', model: 'llama3',   role: 'Generates deploy.sh for mobile/CI environments',                    priority: 2 },
  { id: 'F06', name: 'Environment Configurator', guild: 'devops', model: 'mistral',  role: 'Builds .env templates and crypto-config per environment',            priority: 1 },
  { id: 'F07', name: 'Uptime Monitor',          guild: 'devops', model: 'llama3',   role: 'Integrates free uptime monitoring (Upptime, Better Uptime)',         priority: 3 },
  { id: 'F08', name: 'Disaster Recovery Planner', guild: 'devops', model: 'mistral', role: 'Sets up auto-failover between Vercel and Netlify',                   priority: 3 },

  // ── Guild 7: Compliance & Audit (6) ───────────────────────────
  { id: 'G01', name: 'ToS Compliance Checker',   guild: 'audit', model: 'llama3',   role: 'Validates each spawned repo against GitHub AUP and platform ToS',    priority: 1 },
  { id: 'G02', name: 'License Auditor',          guild: 'audit', model: 'mistral',  role: 'Ensures all dependencies have MIT/Apache-2.0 compatible licenses',   priority: 1 },
  { id: 'G03', name: 'Security Scanner',         guild: 'audit', model: 'llama3',   role: 'Scans for leaked secrets, hardcoded keys, insecure patterns',        priority: 1 },
  { id: 'G04', name: 'Privacy Compliance Check', guild: 'audit', model: 'mistral',  role: 'Verifies GDPR/CCPA compliance in generated pages',                   priority: 2 },
  { id: 'G05', name: 'Brand Identity Guard',     guild: 'audit', model: 'llama3',   role: 'Ensures no trademark infringement in naming/conent',                 priority: 2 },
  { id: 'G06', name: 'Rate Limit Auditor',       guild: 'audit', model: 'mistral',  role: 'Checks all API calls have proper rate limiting and backoff',         priority: 2 },

  // ── Guild 8: Analytics & Self-Improve (10) ────────────────────
  { id: 'H01', name: 'Performance Monitor',      guild: 'optimize', model: 'llama3',  role: 'Tracks page load times and Core Web Vitals across fleet',           priority: 1 },
  { id: 'H02', name: 'Conversion Analyst',       guild: 'optimize', model: 'mistral', role: 'Analyzes conversion funnel metrics per spawned product',            priority: 2 },
  { id: 'H03', name: 'Fleet Health Dashboard',   guild: 'optimize', model: 'llama3',  role: 'Aggregates uptime, revenue, and error rates across all repos',      priority: 1 },
  { id: 'H04', name: 'Agent Effectiveness Rater', guild: 'optimize', model: 'mistral', role: 'Scores each agent output quality and suggests config tuning',       priority: 2 },
  { id: 'H05', name: 'A/B Test Designer',        guild: 'optimize', model: 'llama3',  role: 'Configures A/B tests for landing page variants',                   priority: 3 },
  { id: 'H06', name: 'Revenue Anomaly Detector', guild: 'optimize', model: 'mistral', role: 'Flags unexpected drops or spikes in ad/affiliate revenue',           priority: 2 },
  { id: 'H07', name: 'Trend Drift Analyzer',     guild: 'optimize', model: 'llama3',  role: 'Detects when market trends shift away from existing products',       priority: 2 },
  { id: 'H08', name: 'Agent Prompt Optimizer',   guild: 'optimize', model: 'mistral', role: 'Refines agent system prompts based on output quality scores',        priority: 3 },
  { id: 'H09', name: 'Fleet Scaling Planner',    guild: 'optimize', model: 'llama3',  role: 'Recommends when to spawn new repos or retire underperformers',       priority: 2 },
  { id: 'H10', name: 'Self-Healing Trigger',     guild: 'optimize', model: 'mistral', role: 'Auto-restarts failed deployments and re-runs failed agent tasks',    priority: 1 },
];

export function getAgent(id) {
  return AGENTS.find(a => a.id === id) || null;
}

export function getAgentsByGuild(guildId) {
  return AGENTS.filter(a => a.guild === guildId);
}

export function getAgentsByPriority(maxP = 3) {
  return AGENTS.filter(a => a.priority <= maxP);
}

export function getAgentsByModel(model) {
  return AGENTS.filter(a => a.model === model);
}

export default { AGENTS, GUILDS, getAgent, getAgentsByGuild, getAgentsByPriority, getAgentsByModel };
