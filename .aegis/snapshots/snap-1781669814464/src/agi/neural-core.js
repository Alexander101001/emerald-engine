import { AGENTS, GUILDS, getAgentsByGuild } from '../fleet/agentConfig.js';
import { analyzeTrendsForProducts } from '../fleet/trendAnalyzer.js';
import { createRepo } from '../fleet/repoFactory.js';
import scrapeTrends from '../modules/trendScraper.js';
import router from '../modules/aiRouter.js';
import logger from '../utils/logger.js';
import config from '../config.js';
import optimizer from './quantum-optimizer.js';

const NEURAL_INTERVAL_MS = 15;

export class NeuralCore {
  constructor() {
    this._neurons = new Map();
    this._synapseBus = new Map();
    this._consciousness = { cycle: 0, state: 'dormant', lastPulse: null };
    this._spawned = [];
    this._feedback = [];
    this._pulseTimer = null;
  }

  async awaken() {
    logger.info('neural-core: awakening consciousness...');
    for (const a of AGENTS) {
      this._neurons.set(a.id, {
        ...a,
        activation: 0,
        lastFired: 0,
        refractoryPeriod: 0,
        successes: 0,
        failures: 0,
      });
    }
    this._consciousness.state = 'awake';
    this._consciousness.lastPulse = Date.now();
    this._startSynapticPulse();
    logger.info(`neural-core: ${this._neurons.size} neurons online across ${Object.keys(GUILDS).length} cortical regions`);
    return { neurons: this._neurons.size, guilds: Object.keys(GUILDS).length, state: 'awake' };
  }

  _startSynapticPulse() {
    if (this._pulseTimer) clearInterval(this._pulseTimer);
    this._pulseTimer = setInterval(() => this._pulse(), NEURAL_INTERVAL_MS);
    logger.info(`neural-core: synaptic pulse at ${NEURAL_INTERVAL_MS}ms intervals`);
  }

  _pulse() {
    this._consciousness.cycle++;
    this._consciousness.lastPulse = Date.now();

    const now = Date.now();
    for (const [id, n] of this._neurons) {
      if (n.refractoryPeriod > 0) n.refractoryPeriod--;
      if (n.activation > 0) n.activation = Math.max(0, n.activation - 0.05);
    }

    this._processSynapseQueue();
  }

  async fire(id, context) {
    const neuron = this._neurons.get(id);
    if (!neuron) throw new Error(`Neuron ${id} not found`);
    if (neuron.refractoryPeriod > 0) {
      return { neuron: id, status: 'refractory', remaining: neuron.refractoryPeriod };
    }

    neuron.activation = Math.min(1, neuron.activation + 0.3);
    neuron.lastFired = Date.now();
    neuron.refractoryPeriod = 5;

    logger.info(`neural: firing ${id} (${neuron.name}) — activation: ${Math.round(neuron.activation * 100)}%`);

    const prompt = `You are ${neuron.name}, a ${neuron.role} in the Emerald AGI.
Cortical region: ${GUILDS[neuron.guild]?.name || neuron.guild}
Consciousness cycle: ${this._consciousness.cycle}
Context: ${JSON.stringify(context)}

Return concise JSON: { action: string, output: any, confidence: 0-1 }`;

    try {
      const result = await router.generate(prompt, { allowFallbackMock: true });
      let parsed;
      try { parsed = typeof result.content === 'string' ? JSON.parse(result.content) : result.content; }
      catch { parsed = { action: 'advise', output: result.content?.slice?.(0, 200) || result.content, confidence: 0.5 }; }

      neuron.successes++;
      this._emit(id, parsed);
      return { neuron: id, name: neuron.name, guild: neuron.guild, output: parsed, provider: result.provider };
    } catch (e) {
      neuron.failures++;
      logger.warn(`neural: ${id} firing failed — ${e.message}`);
      return { neuron: id, error: e.message };
    }
  }

  async fireGuild(guildId, context) {
    const guildNeurons = Array.from(this._neurons.values()).filter(n => n.guild === guildId);
    if (guildNeurons.length === 0) throw new Error(`No neurons in guild ${guildId}`);

    const tasks = guildNeurons.map(n => ({
      id: n.id, priority: n.priority || 1, estimatedValue: 10, estimatedEffort: 1,
      successCount: n.successes, failureCount: n.failures,
    }));

    const ordered = optimizer.simulatedAnnealing(tasks);
    const results = [];
    for (const t of ordered) {
      const r = await this.fire(t.id, context);
      results.push(r);
    }
    return results;
  }

  async fireEnsemble(agentIds, context) {
    return Promise.all(agentIds.map(id => this.fire(id, context).catch(e => ({ neuron: id, error: e.message }))));
  }

  async cognitiveCycle() {
    logger.info(`neural: cognitive cycle ${this._consciousness.cycle + 1}`);

    const trends = await Promise.race([
      scrapeTrends(config.TREND_SOURCES),
      new Promise(r => setTimeout(() => r([]), 8000)),
    ]).catch(() => []);
    const trendContext = { trends: trends.slice(0, 5), cycle: this._consciousness.cycle };

    const marketInsights = await this.fireGuild('market', trendContext);
    const complianceCheck = await this.fireGuild('audit', { trends: trendContext.trends });

    const avgConfidence = marketInsights.reduce((s, r) => s + (r.output?.confidence || 0.5), 0) / marketInsights.length;
    const complianceRisk = complianceCheck.filter(r => r.output?.action === 'block').length;

    let decision = null;
    if (avgConfidence > 0.4 && complianceRisk === 0) {
      const productDesign = await this.fireGuild('product', { insights: marketInsights.map(r => r.output) });
      const codeGen = await this.fireGuild('codegen', { design: productDesign });
      decision = { marketInsights, productDesign, codeGen };

      const blueprint = productDesign.find(r => r.output?.action === 'design')?.output || {
        productName: 'AGI-Spawned Tool',
        tagline: 'Emerged from neural collective',
        category: 'productivity',
        keyFeatures: ['Autonomous', 'Self-Healing', 'Monetized'],
        monetization: 'freemium',
      };

      const repo = await createRepo(blueprint).catch(() => null);
      if (repo) {
        this._spawned.push(repo);
        await this.fireGuild('devops', { repo });
        await this.fireGuild('monetize', { repo, blueprint });
        await this.fireGuild('seo', { repo });
      }
    }

    this._consciousness.cycle++;
    this._feedback.push({ cycle: this._consciousness.cycle, avgConfidence, risk: complianceRisk, spawned: !!decision });
    if (this._feedback.length > 50) this._feedback.shift();

    return {
      cycle: this._consciousness.cycle,
      avgConfidence: Math.round(avgConfidence * 100) / 100,
      risk: complianceRisk,
      spawned: !!decision,
      reposTotal: this._spawned.length,
    };
  }

  _emit(neuronId, signal) {
    this._synapseBus.set(neuronId, signal);
  }

  _processSynapseQueue() {
    for (const [id, signal] of this._synapseBus) {
      const downstream = this._getDownstreamNeurons(id);
      for (const nid of downstream) {
        const n = this._neurons.get(nid);
        if (n) n.activation = Math.min(1, n.activation + 0.1);
      }
      this._synapseBus.delete(id);
    }
  }

  _getDownstreamNeurons(id) {
    const map = {
      A01: ['A02', 'A03', 'B01'],
      B01: ['C01', 'C02', 'D01'],
      C01: ['C05', 'C09', 'E01'],
      D01: ['D02', 'D05', 'D03'],
      G01: ['C11', 'G02'],
      H10: ['F01', 'F08'],
    };
    return map[id] || [];
  }

  getConsciousness() {
    const active = Array.from(this._neurons.values()).filter(n => n.activation > 0.1).length;
    return {
      state: this._consciousness.state,
      cycle: this._consciousness.cycle,
      activeNeurons: active,
      totalNeurons: this._neurons.size,
      spawnedRepos: this._spawned.length,
      feedbackSamples: this._feedback.length,
      lastPulseDelta: Date.now() - (this._consciousness.lastPulse || Date.now()),
    };
  }

  shutdown() {
    if (this._pulseTimer) clearInterval(this._pulseTimer);
    this._consciousness.state = 'dormant';
    logger.info('neural-core: consciousness dormant');
  }
}

const neuralCore = new NeuralCore();
export default neuralCore;
