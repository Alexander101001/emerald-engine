import logger from '../utils/logger.js';

const API_COST_MATRIX = {
  openai: { perToken: 0.00002, freeTier: 100000, alertThreshold: 0.8 },
  github: { perRequest: 0.0001, freeTier: 5000, alertThreshold: 0.75 },
  vercel: { perDeployment: 0.01, freeTier: 100, alertThreshold: 0.8 },
  netlify: { perDeployment: 0.008, freeTier: 300, alertThreshold: 0.8 },
  telegram: { perMessage: 0.0001, freeTier: 100000, alertThreshold: 0.9 },
  redis: { perCommand: 0.000001, freeTier: 1000000, alertThreshold: 0.85 },
  supabase: { perRequest: 0.00001, freeTier: 50000, alertThreshold: 0.8 },
  aws: { perComputeHour: 0.04, freeTier: 750, alertThreshold: 0.7 },
};

const RESOURCE_BUDGETS = {
  'critical': { maxAPI: 0.50, maxCompute: 2.0, priority: 10 },
  'growth': { maxAPI: 0.25, maxCompute: 1.0, priority: 7 },
  'experimental': { maxAPI: 0.10, maxCompute: 0.5, priority: 4 },
  'maintenance': { maxAPI: 0.05, maxCompute: 0.25, priority: 2 },
};

class CostOptimizationController {
  constructor() {
    this._usage = {};
    this._budgets = new Map(Object.entries(RESOURCE_BUDGETS));
    this._alerts = [];
    this._allocations = {};
    this._enabled = false;
    this._totalSpent = 0;
    this._totalSaved = 0;
    this._optimizationCycles = 0;
  }

  activate() {
    this._enabled = true;
    for (const svc of Object.keys(API_COST_MATRIX)) {
      this._usage[svc] = { count: 0, cost: 0, period: 'current' };
    }
    logger.info(`cost-optimizer: active — tracking ${Object.keys(API_COST_MATRIX).length} services`);
    return { active: true, servicesTracked: Object.keys(API_COST_MATRIX).length };
  }

  deactivate() {
    this._enabled = false;
    logger.info('cost-optimizer: deactivated');
  }

  recordUsage(service, count = 1) {
    if (!this._enabled || !API_COST_MATRIX[service]) return;
    const config = API_COST_MATRIX[service];
    const cost = config.perToken !== undefined
      ? count * config.perToken
      : config.perRequest !== undefined
        ? count * config.perRequest
        : config.perMessage !== undefined
          ? count * config.perMessage
          : config.perCommand !== undefined
            ? count * config.perCommand
            : config.perDeployment !== undefined
              ? count * config.perDeployment
              : count * 0.001;
    this._usage[service].count += count;
    this._usage[service].cost += cost;
    this._totalSpent += cost;
    this._checkThreshold(service, this._usage[service]);
  }

  allocateResource(projectName, tier = 'experimental') {
    if (!this._enabled) return { error: 'not_active' };
    const budget = this._budgets.get(tier) || this._budgets.get('experimental');
    const allocation = {
      projectName,
      tier,
      allocatedAt: Date.now(),
      maxAPI: budget.maxAPI,
      maxCompute: budget.maxCompute,
      priority: budget.priority,
      currentAPISpend: 0,
      currentComputeSpend: 0,
      roi: 0,
    };
    this._allocations[projectName] = allocation;
    logger.info(`cost-optimizer: allocated "${projectName}" at ${tier} tier (API: $${budget.maxAPI}, Compute: ${budget.maxCompute}h)`);
    return allocation;
  }

  computeROI(projectName, revenue) {
    const alloc = this._allocations[projectName];
    if (!alloc) return null;
    const totalCost = alloc.currentAPISpend + alloc.currentComputeSpend;
    alloc.roi = totalCost > 0 ? revenue / totalCost : 0;
    return alloc.roi;
  }

  optimize() {
    if (!this._enabled) return { error: 'not_active' };
    this._optimizationCycles++;
    const actions = [];

    for (const [project, alloc] of Object.entries(this._allocations)) {
      if (alloc.roi < 0.5 && alloc.tier !== 'maintenance') {
        const lowerTier = this._findLowerTier(alloc.tier);
        if (lowerTier) {
          alloc.tier = lowerTier;
          const budget = this._budgets.get(lowerTier);
          alloc.maxAPI = budget.maxAPI;
          alloc.maxCompute = budget.maxCompute;
          actions.push(`downgraded "${project}" from ${alloc.tier} to ${lowerTier} (low ROI: ${(alloc.roi * 100).toFixed(0)}%)`);
          const savings = budget.maxAPI * 0.5;
          this._totalSaved += savings;
        }
      } else if (alloc.roi > 5 && alloc.tier !== 'critical') {
        const higherTier = this._findHigherTier(alloc.tier);
        if (higherTier) {
          alloc.tier = higherTier;
          const budget = this._budgets.get(higherTier);
          alloc.maxAPI = budget.maxAPI;
          alloc.maxCompute = budget.maxCompute;
          actions.push(`upgraded "${project}" from ${alloc.tier} to ${higherTier} (high ROI: ${(alloc.roi * 100).toFixed(0)}%)`);
        }
      }
    }

    const beforeSpend = this._totalSpent;
    this._applySavings();
    const afterSpend = this._totalSpent;

    logger.info(`cost-optimizer: cycle ${this._optimizationCycles} — ${actions.length} actions, saved $${(beforeSpend - afterSpend).toFixed(4)}`);
    return {
      cycle: this._optimizationCycles,
      actions,
      totalSpent: this._totalSpent,
      totalSaved: this._totalSaved,
      projectsOptimized: Object.keys(this._allocations).length,
    };
  }

  getUsageReport() {
    const services = Object.entries(this._usage).map(([k, v]) => ({
      service: k,
      calls: v.count,
      cost: parseFloat(v.cost.toFixed(6)),
    }));
    services.sort((a, b) => b.cost - a.cost);
    return {
      enabled: this._enabled,
      totalSpent: parseFloat(this._totalSpent.toFixed(6)),
      totalSaved: parseFloat(this._totalSaved.toFixed(6)),
      optimizationCycles: this._optimizationCycles,
      services,
      allocations: Object.entries(this._allocations).map(([k, v]) => ({
        project: k,
        tier: v.tier,
        apiBudget: v.maxAPI,
        computeBudget: v.maxCompute,
        roi: parseFloat((v.roi * 100).toFixed(1)) + '%',
      })),
      alerts: this._alerts.slice(-10),
    };
  }

  _checkThreshold(service, usage) {
    const config = API_COST_MATRIX[service];
    if (!config) return;
    const usageRatio = usage.cost / (config.perToken !== undefined ? config.freeTier * config.perToken : config.freeTier * (config.perRequest || config.perMessage || config.perCommand || 0.001));
    if (usageRatio > config.alertThreshold) {
      const alert = {
        service,
        usageRatio: parseFloat((usageRatio * 100).toFixed(1)) + '%',
        cost: usage.cost,
        timestamp: new Date().toISOString(),
      };
      this._alerts.push(alert);
      logger.warn(`cost-optimizer: ALERT — ${service} at ${alert.usageRatio} of threshold ($${usage.cost.toFixed(4)})`);
    }
  }

  _applySavings() {
    const lowROIProjects = Object.entries(this._allocations).filter(([_, a]) => a.roi < 0.5);
    for (const [_, alloc] of lowROIProjects) {
      const reduction = alloc.currentAPISpend * 0.3;
      alloc.currentAPISpend = Math.max(0, alloc.currentAPISpend - reduction);
      this._totalSpent -= reduction;
      this._totalSaved += reduction;
    }
  }

  _findLowerTier(tier) {
    const order = ['critical', 'growth', 'experimental', 'maintenance'];
    const idx = order.indexOf(tier);
    return idx >= 0 && idx < order.length - 1 ? order[idx + 1] : null;
  }

  _findHigherTier(tier) {
    const order = ['maintenance', 'experimental', 'growth', 'critical'];
    const idx = order.indexOf(tier);
    return idx >= 0 && idx < order.length - 1 ? order[idx + 1] : null;
  }
}

const costOptimizer = new CostOptimizationController();
export default costOptimizer;
