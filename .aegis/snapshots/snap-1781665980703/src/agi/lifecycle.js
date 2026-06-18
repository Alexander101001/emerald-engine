import logger from '../utils/logger.js';
import aegis from './aegis.js';
import neuralCore from './neural-core.js';
import optimizer from './quantum-optimizer.js';
import orchestrator, { initOrchestrator } from './orchestrator.js';
import watchdog from './aegis-watchdog.js';
import telegram from './agi-telegram-bridge.js';
import { createRepo } from '../fleet/repoFactory.js';
import { deployAll } from '../fleet/deployer.js';

export class AutonomousLifecycle {
  constructor() {
    this._intervalIds = [];
    this._cycles = 0;
    this._failures = [];
    this._recoveries = [];
    this._vaultLoaded = false;
  }

  async deploy(vaultPassphrase) {
    logger.info('lifecycle: autonomous deployment starting');
    const passphrase = vaultPassphrase || process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026';

    const aegisStatus = await aegis.activate(passphrase);

    const vaultIntegrity = aegis.verifyVaultIntegrity();
    if (!vaultIntegrity.ok) {
      logger.warn(`lifecycle: vault integrity check ${vaultIntegrity.ok ? 'PASS' : 'FAIL'} — ${vaultIntegrity.reason || 'hash_mismatch'}`);
    }

    try {
      const orchStatus = await initOrchestrator();
      this._vaultLoaded = orchStatus.loaded;
      logger.info(`lifecycle: secure vault loaded — ${orchStatus.keyCount} keys in volatile memory`);
    } catch (e) {
      logger.warn(`lifecycle: vault load skipped — ${e.message}`);
    }

    const neuralStatus = await neuralCore.awaken();

    const telegramCfg = orchestrator.getTelegramConfig();
    if (telegramCfg && telegramCfg.bot_token && !telegramCfg.bot_token.startsWith('[INSERT')) {
      telegram.reset();
      const initResult = await telegram.initialize(telegramCfg.bot_token, telegramCfg.user_id, telegramCfg.channel_id);
      if (initResult.active) {
        const channelCheck = await telegram.verifyChannel(telegramCfg.channel_id);
        if (channelCheck.ok) {
          logger.info(`lifecycle: channel verified — ${channelCheck.chatId}`);
          try { await telegram.sendToChannel('System Awakened: Autonomous Mode'); } catch {}
        } else {
          logger.warn(`lifecycle: channel verify failed — ${channelCheck.reason}`);
        }
      }
    }

    logger.info(`lifecycle: aegis=${aegisStatus.status} neural=${neuralStatus.state} vault=${this._vaultLoaded}`);

    this._scheduleHeartbeat();
    this._scheduleCognitiveCycle();
    this._scheduleSelfRepair();
    this._scheduleSnapshot();
    this._scheduleVaultVerification();

    this._cycles++;
    return {
      aegis: aegisStatus,
      neural: neuralStatus,
      vault: orchestrator.getStatus(),
      watchdog: watchdog.getStatus(),
      telegram: telegram.getStatus(),
    };
  }

  _scheduleHeartbeat() {
    const id = setInterval(async () => {
      const result = await aegis.verifyIntegrity().catch(() => ({ ok: false }));
      if (!result.ok) {
        logger.warn(`lifecycle: integrity heartbeat — ${result.violations} violations, ${result.repaired} repaired`);
        if (result.violations > 0 && result.repaired === 0) {
          await this._emergencyRecovery();
        }
      }
    }, 30000);
    this._intervalIds.push(id);
  }

  _scheduleCognitiveCycle() {
    const id = setInterval(async () => {
      try {
        const cycle = await neuralCore.cognitiveCycle();
        logger.info(`lifecycle: cognitive cycle ${cycle.cycle} — confidence=${cycle.avgConfidence} risk=${cycle.risk} spawned=${cycle.spawned}`);
        this._cycles++;
      } catch (e) {
        logger.error(`lifecycle: cognitive cycle failed — ${e.message}`);
        this._failures.push({ type: 'cognitive_cycle', error: e.message, ts: Date.now() });
      }
    }, 3600000);
    this._intervalIds.push(id);
  }

  _scheduleSelfRepair() {
    const id = setInterval(async () => {
      const status = aegis.getStatus();
      if (!status.active) {
        logger.warn('lifecycle: aegis inactive — attempting restart');
        try {
          await aegis.activate(process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026');
          this._recoveries.push({ type: 'aegis_restart', ts: Date.now() });
        } catch (e) {
          logger.error(`lifecycle: aegis restart failed — ${e.message}`);
        }
      }

      const neural = neuralCore.getConsciousness();
      if (neural.state !== 'awake') {
        logger.warn('lifecycle: neural core dormant — re-awakening');
        try {
          await neuralCore.awaken();
          this._recoveries.push({ type: 'neural_restart', ts: Date.now() });
        } catch (e) {
          logger.error(`lifecycle: neural restart failed — ${e.message}`);
        }
      }
    }, 60000);
    this._intervalIds.push(id);
  }

  _scheduleSnapshot() {
    const id = setInterval(async () => {
      try {
        await aegis.createSnapshot();
      } catch (e) {
        logger.error(`lifecycle: snapshot failed — ${e.message}`);
      }
    }, 3600000);
    this._intervalIds.push(id);
  }

  _scheduleVaultVerification() {
    const id = setInterval(async () => {
      if (!this._vaultLoaded) return;
      const integrity = aegis.verifyVaultIntegrity();
      if (!integrity.ok) {
        logger.warn('lifecycle: vault integrity violated');
        this._failures.push({ type: 'vault_integrity', ts: Date.now() });
        orchestrator.wipe();
        this._vaultLoaded = false;
      }
    }, 300000);
    this._intervalIds.push(id);
  }

  async _emergencyRecovery() {
    logger.warn('lifecycle: initiating emergency recovery');
    try {
      await aegis.createSnapshot();
      await aegis.verifyIntegrity();
      this._recoveries.push({ type: 'emergency_recovery', ts: Date.now() });
    } catch (e) {
      logger.error(`lifecycle: emergency recovery failed — ${e.message}`);
    }
  }

  async scaleUp(product) {
    logger.info(`lifecycle: scaling up — spawning "${product.productName}"`);
    const repo = await createRepo(product).catch(e => {
      logger.error(`lifecycle: scale-up failed — ${e.message}`);
      return null;
    });
    if (repo) {
      logger.info(`lifecycle: spawned ${repo.repoName}`);
      try {
        await telegram.sendToChannel(`Spawned: ${repo.repoName} — ${repo.url}`);
      } catch {}

      const vercelToken = orchestrator.getVercelToken();
      const netlifyToken = orchestrator.getNetlifyToken();
      if (vercelToken || netlifyToken) {
        const deploys = await deployAll(repo, vercelToken, netlifyToken);
        for (const d of deploys) {
          try {
            await telegram.sendDeploymentSuccess(d.platform, d.url, repo.repoName);
          } catch {}
        }
      }
    }
    return repo;
  }

  async selfHeal() {
    const before = aegis.getStatus().trackedFiles;
    await aegis.verifyIntegrity();
    const after = aegis.getStatus().trackedFiles;
    return { before, after, repairs: aegis._repairCount, cycles: this._cycles };
  }

  getLifecycleStatus() {
    return {
      cycles: this._cycles,
      failures: this._failures.length,
      recoveries: this._recoveries.length,
      vaultLoaded: this._vaultLoaded,
      aegis: aegis.getStatus(),
      neural: neuralCore.getConsciousness(),
      optimizer: optimizer.getOptimizerStatus(),
      orchestrator: orchestrator.getStatus(),
      watchdog: watchdog.getStatus(),
      telegram: telegram.getStatus(),
    };
  }

  shutdown() {
    for (const id of this._intervalIds) clearInterval(id);
    watchdog.deactivate();
    orchestrator.wipe();
    telegram.shutdown();
    aegis.stopHeartbeat();
    neuralCore.shutdown();
    logger.info('lifecycle: autonomous lifecycle terminated');
  }
}

const lifecycle = new AutonomousLifecycle();
export default lifecycle;
