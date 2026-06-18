import core from './fleet/emerald-core.js';
import { runBootSequence } from './fleet/bootSequence.js';
import logger from './utils/logger.js';
import config from './config.js';

const AGI_MODE = process.argv.includes('--agi') || process.env.EMERALD_AGI === '1';
const BUILD_MODE = process.argv.includes('--build') || process.env.EMERALD_BUILD === '1';

async function main() {
  if (AGI_MODE) {
    logger.info(`========================================`);
    logger.info(`  Emerald AGI v1.0 — Unified Consciousness`);
    logger.info(`  Mode: ${BUILD_MODE ? 'BUILD' : 'SERVICE'}`);
    logger.info(`========================================`);

    const { default: aegis } = await import('./agi/aegis.js');
    const { default: lifecycle } = await import('./agi/index.js');
    const { default: orchestrator } = await import('./agi/orchestrator.js');
    const { default: watchdog } = await import('./agi/aegis-watchdog.js');
    const { default: telegram } = await import('./agi/agi-telegram-bridge.js');

    const vaultIntegrity = aegis.verifyVaultIntegrity();
    logger.info(`agi: vault integrity — ${vaultIntegrity.ok ? 'PASS' : 'FAIL'} ${vaultIntegrity.reason || ''}`);

    if (!vaultIntegrity.ok && BUILD_MODE) {
      logger.error('agi: BUILD ABORTED — vault integrity check failed');
      process.exit(1);
    }

    const deployStatus = await lifecycle.deploy();

    logger.info(`agi: aegis=${deployStatus.aegis.dbSize}files neural=${deployStatus.neural.neurons}neurons vault=${deployStatus.vault.keyCount}keys`);

    if (BUILD_MODE) {
      const { default: neuralCore } = await import('./agi/neural-core.js');
      const cycle = await neuralCore.cognitiveCycle();
      logger.info(`agi: build cycle ${cycle.cycle} — confidence=${cycle.avgConfidence} spawned=${cycle.spawned}`);

      await aegis.createSnapshot();

      try {
        await telegram.sendToOwner(
          `Emerald AGI Build Complete\nNeurons: ${deployStatus.neural.neurons}\nGuilds: ${deployStatus.neural.guilds}\nVault Keys: ${deployStatus.vault.keyCount}\nRepos Spawned: ${cycle.reposTotal}\nWatchdog: ${deployStatus.watchdog.active ? 'ACTIVE' : 'OFF'}`
        );
      } catch {}
    }

    const s = lifecycle.getLifecycleStatus();
    logger.info('agi: runtime online', JSON.stringify({
      neurons: s.neural.totalNeurons,
      repos: s.neural.spawnedRepos,
      files: s.aegis.trackedFiles,
      vault: s.orchestrator.keyCount + ' keys',
      watchdog: s.watchdog.active ? 'active' : 'off',
      telegram: s.telegram.active ? 'connected' : 'disabled',
    }));
    return lifecycle;
  }

  logger.info(`========================================`);
  logger.info(`  Emerald Fleet v1.0 — ${config.DEPLOY_PLATFORM}`);
  logger.info(`  ${BUILD_MODE ? 'BUILD MODE: spawning first-wave repos' : 'SERVICE MODE: running orchestration loops'}`);
  logger.info(`========================================`);

  if (BUILD_MODE) {
    const bootResult = await runBootSequence();
    logger.info(`index: boot sequence complete`);
    logger.info(`  spawned: ${bootResult.summary.spawned}`);
    logger.info(`  failed:  ${bootResult.summary.failed}`);
    if (bootResult.summary.spawned > 0) {
      logger.info(`  repos: ${bootResult.firstWave.filter(r => r.url).map(r => r.url).join(', ')}`);
    }
  }

  const boot = await core.bootstrap();
  logger.info('index: fleet engine started', JSON.stringify(boot));

  core.schedule(async () => {
    try {
      const cycle = await core.orchestrationCycle();
      logger.info(`index: cycle ${cycle.cycle} done — ${cycle.spawned.length} new repos`);
    } catch (e) {
      logger.error(`index: cycle failed — ${e.message}`);
    }
  }, 86400000);

  logger.info('index: daily orchestration cycle scheduled');
}

main().catch(e => {
  logger.error('index: fatal', e.message);
  process.exit(1);
});

export default core;
