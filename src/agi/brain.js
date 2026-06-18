import { SYSTEM_PROMPT } from '../config/systemConfig.js';
import { isSafeToModify } from './core/guardrail.js';
import ollama from './core/ollamaConnector.js';
import { loadTokens } from './core/tokenLoader.js';
import { Orchestrator } from './core/orchestrator.js';

async function runBrain() {
  console.log('--- INITIALIZING EMERALD ARCHITECT ---');

  loadTokens();

  const orchestrator = new Orchestrator();
  await orchestrator.bootstrap();

  const task = 'Audit system, identify top 3 revenue streams, and propose a new child-agent.';
  const response = await ollama.generateThought(
    SYSTEM_PROMPT + '\n\nCURRENT TASK:\n' + task +
    '\n\nCurrent metrics: ' + JSON.stringify(orchestrator.metrics.summary())
  );
  console.log('ARCHITECT STRATEGY:', response);

  if (response && response.includes('NEW_MODULE')) {
    const targetPath = './src/modules/monetization/strategies/aiGenerated.js';
    if (isSafeToModify(targetPath)) {
      console.log('Deploying new module safely...');
      await orchestrator.deployer.writeAndDeploy(
        targetPath,
        `export default {\n  name: 'AI Generated Strategy',\n  tier: 'experimental',\n  async execute() {\n    console.log('[AI Strategy] Executing...');\n    return { revenue: 0, status: 'active', note: 'AI-generated strategy deployed' };\n  }\n};`,
        'Emerald: AI-generated strategy module'
      );
    } else {
      console.log('Modification blocked for safety.');
    }
  }

  console.log('\n--- EMERALD ARCHITECT ONLINE ---');
  await orchestrator.runForever(3600000);
}

runBrain().catch(e => {
  console.error('Brain fatal:', e.message);
  process.exit(1);
});
