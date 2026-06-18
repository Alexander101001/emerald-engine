import core from './emerald-core.js';
import logger from '../utils/logger.js';

export async function runBootSequence() {
  logger.info('bootSequence: starting first-wave repository generation');

  const bootStatus = await core.bootstrap();
  logger.info(`bootSequence: fleet online — ${bootStatus.agents} agents`);

  const firstWave = [
    { productName: 'LinkShortener Pro', tagline: 'Smart link management with analytics', category: 'marketing', keyFeatures: ['Custom slugs', 'Click analytics', 'QR codes', 'Team workspaces'], monetization: 'freemium' },
    { productName: 'FormBuilder Lite', tagline: 'Build forms that convert', category: 'productivity', keyFeatures: ['Drag-drop builder', 'Email notifications', 'Response export', 'Template library'], monetization: 'freemium' },
    { productName: 'UptimeRadar', tagline: 'Free website monitoring for indie hackers', category: 'developer', keyFeatures: ['5-min checks', 'Slack alerts', 'Status pages', 'SSL monitoring'], monetization: 'freemium' },
  ];

  const results = [];

  for (const blueprint of firstWave) {
    logger.info(`bootSequence: spawning "${blueprint.productName}"...`);

    const approved = core._phaseComplianceCheck(blueprint);
    if (!approved) {
      logger.warn(`bootSequence: ${blueprint.productName} failed compliance — skipping`);
      results.push({ product: blueprint.productName, status: 'compliance_failed' });
      continue;
    }

    try {
      const repo = await core._phaseSpawnRepo(blueprint);
      results.push({
        product: blueprint.productName,
        status: repo.simulated ? 'simulated' : 'spawned',
        url: repo.url,
        files: repo.files?.length || repo.fileCount || 0,
      });
      logger.info(`bootSequence: ${repo.simulated ? 'simulated' : 'spawned'} ${blueprint.productName} → ${repo.url}`);
    } catch (e) {
      logger.error(`bootSequence: ${blueprint.productName} failed — ${e.message}`);
      results.push({ product: blueprint.productName, status: 'failed', error: e.message });
    }
  }

  const spawned = results.filter(r => r.status === 'spawned' || r.status === 'simulated');
  const failed = results.filter(r => r.status === 'failed');

  logger.info(`bootSequence: first-wave complete — ${spawned.length} spawned, ${failed.length} failed`);

  return {
    fleetStatus: bootStatus,
    firstWave: results,
    summary: { total: results.length, spawned: spawned.length, failed: failed.length },
    nextSteps: [
      'Set GITHUB_PAT and GITHUB_OWNER env vars for real repo creation',
      'Run core.orchestrationCycle() for continuous trend-driven spawning',
      'Monitor spawned repos via core.getFleetSummary()',
    ],
  };
}

export default { runBootSequence };
