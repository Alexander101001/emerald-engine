import { Orchestrator } from './agi/core/orchestrator.js';

const orchestrator = new Orchestrator();

async function main() {
  await orchestrator.bootstrap();
  await orchestrator.runForever(3600000);
}

main().catch(e => {
  console.error('Fatal:', e.message);
  process.exit(1);
});
