export class SystemMetrics {
  constructor() {
    this.history = [];
    this.startTime = Date.now();
  }

  recordStrategies(results) {
    const entry = {
      timestamp: new Date().toISOString(),
      strategies: results,
      uptime: Math.floor((Date.now() - this.startTime) / 1000)
    };
    this.history.push(entry);
  }

  summary() {
    const last = this.history[this.history.length - 1];
    const activeStrategies = last ? last.strategies.filter(s => s && s.status === 'active').length : 0;
    const projectedRevenue = last ? last.strategies.reduce((sum, s) => sum + (s?.revenue || 0), 0) : 0;
    return {
      uptime: last?.uptime || 0,
      cycles: this.history.length,
      activeStrategies,
      projectedMonthlyRevenue: projectedRevenue,
      dailyTarget: projectedRevenue > 0 ? `${(projectedRevenue / 30).toFixed(0)} $/day` : 'N/A'
    };
  }
}
