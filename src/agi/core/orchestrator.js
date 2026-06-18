import { loadTokens } from './tokenLoader.js';
import { SelfDeployer } from './selfDeployer.js';
import { DynamicMonetizationEngine } from '../../modules/monetization/DynamicMonetizationEngine.js';
import { SystemMetrics } from './systemMetrics.js';

export class Orchestrator {
  constructor() {
    this.deployer = new SelfDeployer();
    this.monetization = new DynamicMonetizationEngine();
    this.metrics = new SystemMetrics();
    this.cycleCount = 0;
    this.startTime = Date.now();
  }

  async bootstrap() {
    console.log('========================================');
    console.log('  Emerald Engine v1.0 — BOOT SEQUENCE');
    console.log('========================================');

    const tokensLoaded = loadTokens();
    console.log(`  Tokens:     ${tokensLoaded ? 'LOADED' : 'ENV_ONLY'}`);

    const deployReady = await this.deployer.init();
    console.log(`  Deployer:   ${deployReady ? 'READY' : 'DISABLED'}`);

    await this.monetization.loadStrategies();
    console.log(`  Strategies: ${this.monetization.strategies.length} loaded`);

    console.log('========================================\n');
    return { tokensLoaded, deployReady };
  }

  async executeCycle() {
    this.cycleCount++;
    console.log(`\n--- Cycle ${this.cycleCount} ---`);

    const strategyResults = await this.monetization.executeAll();
    this.metrics.recordStrategies(strategyResults);

    const report = this.metrics.summary();
    console.log('Metrics:', JSON.stringify(report, null, 2));

    const shouldDeploy = strategyResults.some(r => r && r.status === 'active');
    if (shouldDeploy) {
      await this.deployer.pushChanges(`Emerald: cycle ${this.cycleCount} auto-update`);
    }

    return { cycle: this.cycleCount, strategies: strategyResults, metrics: report };
  }

  async runForever(intervalMs = 3600000) {
    console.log(`Orchestrator running — cycle every ${intervalMs / 60000}min\n`);
    await this.executeCycle();
    setInterval(() => this.executeCycle().catch(e => console.error('Cycle error:', e)), intervalMs);
  }
}
