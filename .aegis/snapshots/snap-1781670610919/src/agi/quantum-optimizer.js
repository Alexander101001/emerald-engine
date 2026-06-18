import logger from '../utils/logger.js';

export class QuantumOptimizer {
  constructor() {
    this._taskHistory = [];
    this._roiCache = new Map();
    this._explorationRate = 0.15;
    this._temperature = 1.0;
  }

  simulatedAnnealing(tasks, iterations = 100) {
    if (tasks.length === 0) return [];

    this._temperature = 1.0;
    const coolingRate = 0.95;
    let currentOrder = [...tasks];
    let currentScore = this._scoreSequence(currentOrder);
    let bestOrder = [...currentOrder];
    let bestScore = currentScore;

    for (let i = 0; i < iterations; i++) {
      const newOrder = this._swapTwo(currentOrder);
      const newScore = this._scoreSequence(newOrder);
      const delta = newScore - currentScore;

      if (delta > 0 || Math.exp(delta / this._temperature) > Math.random()) {
        currentOrder = newOrder;
        currentScore = newScore;
        if (newScore > bestScore) {
          bestOrder = [...newOrder];
          bestScore = newScore;
        }
      }

      this._temperature *= coolingRate;
    }

    return bestOrder;
  }

  monteCarloROI(task, samples = 50) {
    const cached = this._roiCache.get(task.id || task);
    if (cached && Date.now() - cached.ts < 60000) return cached.roi;

    let totalROI = 0;
    for (let i = 0; i < samples; i++) {
      const timeVariance = 0.5 + Math.random();
      const impactVariance = 0.3 + Math.random() * 1.4;
      const probability = Math.random() * 0.4 + 0.6;
      const estimatedValue = (task.estimatedValue || 10) * impactVariance;
      const estimatedTime = ((task.estimatedEffort || 1) * timeVariance) || 1;
      totalROI += (estimatedValue * probability) / estimatedTime;
    }

    const roi = Math.round((totalROI / samples) * 100) / 100;
    this._roiCache.set(task.id || task, { roi, ts: Date.now() });
    return roi;
  }

  thompsonSample(tasks) {
    if (tasks.length === 0) return null;
    const scores = tasks.map(t => {
      const alpha = (t.successCount || 1) + 1;
      const beta = (t.failureCount || 0) + 1;
      const sample = this._betaRandom(alpha, beta);
      return { task: t, score: sample };
    });
    scores.sort((a, b) => b.score - a.score);
    return scores[0].task;
  }

  parallelTaskAllocation(agentPool, tasks) {
    if (tasks.length === 0 || agentPool.length === 0) return {};

    const scored = tasks.map(t => ({
      task: t,
      roi: this.monteCarloROI(t),
      urgency: t.priority || 1,
      diversity: this._taskHistory.filter(h => h === (t.id || t)).length === 0 ? 0.2 : 0,
    }));

    scored.sort((a, b) => {
      const aScore = a.roi * 0.5 + a.urgency * 0.3 + a.diversity * 0.2;
      const bScore = b.roi * 0.5 + b.urgency * 0.3 + b.diversity * 0.2;
      return bScore - aScore;
    });

    const allocation = {};
    const agentQueue = [...agentPool];
    for (let i = 0; i < scored.length && agentQueue.length > 0; i++) {
      const agentIdx = i % agentQueue.length;
      const agent = agentQueue[agentIdx];
      if (!allocation[agent.id || agent]) allocation[agent.id || agent] = [];
      allocation[agent.id || agent].push(scored[i].task);
      this._taskHistory.push(scored[i].task.id || scored[i].task);
    }

    if (this._taskHistory.length > 1000) this._taskHistory.splice(0, 500);
    return allocation;
  }

  calculateHighestROIPath(tasks, agents) {
    const ordered = this.simulatedAnnealing(tasks, 50);
    const allocation = this.parallelTaskAllocation(agents, ordered);
    const totalROI = ordered.reduce((s, t) => s + this.monteCarloROI(t), 0);
    return { allocation, order: ordered, totalROI, agentCount: agents.length, taskCount: tasks.length };
  }

  _scoreSequence(tasks) {
    if (tasks.length === 0) return 0;
    let score = 0;
    for (let i = 0; i < tasks.length; i++) {
      const t = tasks[i];
      const roi = this.monteCarloROI(t);
      const penalty = i * 0.1;
      score += roi - penalty;
    }
    return score;
  }

  _swapTwo(arr) {
    const result = [...arr];
    const i = Math.floor(Math.random() * result.length);
    let j = Math.floor(Math.random() * result.length);
    while (j === i) j = Math.floor(Math.random() * result.length);
    [result[i], result[j]] = [result[j], result[i]];
    return result;
  }

  _betaRandom(alpha, beta) {
    const x = Math.pow(Math.random(), 1 / alpha);
    const y = Math.pow(Math.random(), 1 / beta);
    return x / (x + y);
  }

  recordResult(taskId, success) {
    if (success) {
      const existing = this._taskHistory.filter(h => h === taskId).length;
      if (existing === 0) this._taskHistory.push(taskId);
    }
  }

  getOptimizerStatus() {
    return {
      temperature: Math.round(this._temperature * 100) / 100,
      explorationRate: this._explorationRate,
      tasksSampled: this._roiCache.size,
      historyLength: this._taskHistory.length,
    };
  }
}

const optimizer = new QuantumOptimizer();
export default optimizer;
