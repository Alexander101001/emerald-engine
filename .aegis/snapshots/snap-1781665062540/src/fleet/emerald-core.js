import { AGENTS, GUILDS, getAgentsByGuild, getAgentsByPriority } from './agentConfig.js';
import { analyzeTrendsForProducts } from './trendAnalyzer.js';
import { createRepo, checkRateLimit } from './repoFactory.js';
import scrapeTrends from '../modules/trendScraper.js';
import router from '../modules/aiRouter.js';
import affiliateManager from '../monetization/affiliate.js';
import subManager from '../modules/subscription.js';
import logger from '../utils/logger.js';
import config from '../config.js';

export class EmeraldCore {
  constructor() {
    this.agents = AGENTS;
    this.guilds = GUILDS;
    this.spawnedRepos = [];
    this.fleetMetrics = { totalRepos: 0, totalDeployments: 0, lastCycle: null, errors: [] };
    this._intervalIds = [];
    this._cycleCount = 0;
  }

  async bootstrap() {
    logger.info(`emerald-core: booting fleet v1.0 — ${this.agents.length} agents in ${Object.keys(this.guilds).length} guilds`);

    for (const [key, guild] of Object.entries(this.guilds)) {
      const guildAgents = getAgentsByGuild(key);
      logger.info(`  guild ${guild.id}: ${guild.name} (${guildAgents.length} agents)`);
    }

    subManager.registerUser('fleet-admin', 'pro');

    logger.info('emerald-core: fleet ready');
    return {
      status: 'ok',
      agents: this.agents.length,
      guilds: Object.keys(this.guilds).length,
      platform: config.DEPLOY_PLATFORM,
    };
  }

  async orchestrationCycle() {
    this._cycleCount++;
    logger.info(`emerald-core: starting cycle ${this._cycleCount}`);

    const phaseResults = {};

    phaseResults.trends = await this._phaseTrendAnalysis();

    phaseResults.blueprint = await this._phaseProductDesign(phaseResults.trends);

    phaseResults.approved = this._phaseComplianceCheck(phaseResults.blueprint);

    if (phaseResults.approved) {
      phaseResults.repo = await this._phaseSpawnRepo(phaseResults.blueprint);
      phaseResults.monetized = this._phaseMonetize(phaseResults.blueprint);
    }

    phaseResults.optimized = await this._phaseOptimizeFleet();

    this.fleetMetrics.lastCycle = new Date().toISOString();
    this.fleetMetrics.totalRepos = this.spawnedRepos.length;

    logger.info(`emerald-core: cycle ${this._cycleCount} complete — ${this.spawnedRepos.length} repos spawned`);
    return {
      cycle: this._cycleCount,
      timestamp: this.fleetMetrics.lastCycle,
      spawned: phaseResults.repo ? [phaseResults.repo.repoName] : [],
      reposTotal: this.spawnedRepos.length,
      phases: Object.keys(phaseResults),
    };
  }

  async _phaseTrendAnalysis() {
    logger.info('[Phase 1/5] Trend Analysis');
    const trends = await scrapeTrends(config.TREND_SOURCES).catch(() => []);
    const analysis = await analyzeTrendsForProducts().catch(() => null);
    return { trends, analysis };
  }

  async _phaseProductDesign(trendData) {
    logger.info('[Phase 2/5] Product Design');
    let blueprint = trendData.analysis?.blueprint;

    if (!blueprint) {
      const prompt = `Design a micro-SaaS product for a trending niche. Return JSON with productName, tagline, category, targetAudience, keyFeatures (array of 3-5), monetization (freemium/free/paid).`;
      const aiResult = await router.generate(prompt, { allowFallbackMock: true });
      try {
        blueprint = typeof aiResult.content === 'string' ? JSON.parse(aiResult.content) : aiResult.content;
      } catch {
        blueprint = {
          productName: 'Emerald Micro-SaaS',
          tagline: 'AI-powered productivity for modern teams',
          category: 'productivity',
          targetAudience: 'freelancers and small teams',
          keyFeatures: ['Smart Dashboard', 'Automated Reports', 'Team Collaboration', 'API Access'],
          monetization: 'freemium',
        };
      }
    }

    return blueprint;
  }

  _phaseComplianceCheck(blueprint) {
    logger.info('[Phase 3/5] Compliance Audit');

    const checks = [];

    const nameOk = blueprint.productName && blueprint.productName.length >= 3;
    checks.push({ check: 'product_name', pass: nameOk });

    const featuresOk = blueprint.keyFeatures && blueprint.keyFeatures.length >= 1;
    checks.push({ check: 'has_features', pass: featuresOk });

    const categoryOk = blueprint.category && !blueprint.category.includes('crypto') && !blueprint.category.includes('gambling');
    checks.push({ check: 'category_compliant', pass: categoryOk });

    const monetOk = ['freemium', 'free', 'paid'].includes(blueprint.monetization);
    checks.push({ check: 'monetization_valid', pass: monetOk });

    const allPass = checks.every(c => c.pass);
    logger.info(`  compliance: ${allPass ? 'PASS' : 'FAIL'} (${checks.filter(c => c.pass).length}/${checks.length})`);

    return allPass;
  }

  async _phaseSpawnRepo(blueprint) {
    logger.info(`[Phase 4/5] Spawning Repository — "${blueprint.productName}"`);

    const rateLimit = await checkRateLimit().catch(() => ({ remaining: 0 }));
    if (rateLimit.remaining < 3) {
      logger.warn(`  rate limit low (${rateLimit.remaining}) — simulating instead`);
      const repo = await createRepo(blueprint);
      this.spawnedRepos.push(repo);
      return repo;
    }

    const repo = await createRepo(blueprint);
    this.spawnedRepos.push(repo);
    return repo;
  }

  _phaseMonetize(blueprint) {
    logger.info('[Phase 4b/5] Monetization Injection');

    const networks = affiliateManager.networks.map(n => n.name);
    const category = blueprint.category || 'general';
    const bestNetwork = affiliateManager.getLink('default', { category });

    return {
      affiliateNetworks: networks,
      recommendedNetwork: bestNetwork?.network || 'amazon',
      adSlots: config.AD_SLOT_COUNT,
      pricing: blueprint.monetization,
    };
  }

  async _phaseOptimizeFleet() {
    logger.info('[Phase 5/5] Fleet Optimization');

    const agentsByGuild = {};
    for (const key of Object.keys(this.guilds)) {
      agentsByGuild[key] = getAgentsByGuild(key).map(a => a.name);
    }

    this.fleetMetrics.totalDeployments += this.spawnedRepos.length;

    return {
      agentsPerGuild: Object.fromEntries(
        Object.entries(agentsByGuild).map(([k, v]) => [k, v.length])
      ),
      totalRepos: this.spawnedRepos.length,
      totalDeployments: this.fleetMetrics.totalDeployments,
    };
  }

  async runAgent(id, context) {
    const agent = AGENTS.find(a => a.id === id);
    if (!agent) throw new Error(`Agent ${id} not found`);

    logger.info(`agent ${agent.id} (${agent.name}) triggered — ${agent.role}`);

    const prompt = `You are ${agent.name}, a ${agent.role} in the Emerald fleet.
Your guild: ${GUILDS[agent.guild]?.name || agent.guild}
Context: ${JSON.stringify(context)}

Execute your role and return a concise JSON result with { action, output }.`;

    const result = await router.generate(prompt, { allowFallbackMock: true });
    let parsed;
    try {
      parsed = typeof result.content === 'string' ? JSON.parse(result.content) : result.content;
    } catch {
      parsed = { action: 'advise', output: result.content.slice(0, 200) };
    }

    return { agent: agent.id, name: agent.name, guild: agent.guild, result: parsed, provider: result.provider };
  }

  async runGuild(guildId, context = {}) {
    const guildAgents = getAgentsByGuild(guildId);
    if (guildAgents.length === 0) throw new Error(`No agents in guild ${guildId}`);

    logger.info(`running guild ${guildId} (${guildAgents.length} agents)`);

    const results = [];
    for (const agent of guildAgents) {
      try {
        const r = await this.runAgent(agent.id, context);
        results.push(r);
      } catch (e) {
        logger.warn(`agent ${agent.id} failed: ${e.message}`);
        results.push({ agent: agent.id, error: e.message });
      }
    }
    return results;
  }

  getFleetSummary() {
    return {
      agents: this.agents.length,
      guilds: Object.keys(this.guilds).length,
      spawnedRepos: this.spawnedRepos.length,
      repos: this.spawnedRepos.map(r => ({ name: r.repoName, url: r.url })),
      cycleCount: this._cycleCount,
      lastCycle: this.fleetMetrics.lastCycle,
      totalDeployments: this.fleetMetrics.totalDeployments,
      errors: this.fleetMetrics.errors.slice(-5),
    };
  }

  schedule(fn, intervalMs) {
    const id = setInterval(fn, intervalMs);
    this._intervalIds.push(id);
    return id;
  }

  async shutdown() {
    for (const id of this._intervalIds) clearInterval(id);
    logger.info('emerald-core: fleet shutdown');
  }
}

const core = new EmeraldCore();
export default core;
