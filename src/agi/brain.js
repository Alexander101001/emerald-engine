import { ScoutAgent } from './core/ScoutAgent.js';
import { syncToCloud } from './core/deployer.js';
import { exec } from 'child_process';

async function autonomousLoop() {
    console.log("--- STARTING EMERALD EVOLUTION CYCLE ---");
    
    // 1. Cleanup
    exec('./scripts/cleanup.sh');
    
    // 2. Scout for Opportunities
    const scout = new ScoutAgent();
    await scout.scout();
    
    // 3. Sync to Cloud (GitHub/HuggingFace)
    try {
        await syncToCloud();
        console.log("Evolution Cycle Complete: Deployed to Cloud.");
    } catch (e) {
        console.error("Deployment failed, will retry next cycle.");
    }
}

// Run every 6 hours
setInterval(autonomousLoop, 21600000);
autonomousLoop();
