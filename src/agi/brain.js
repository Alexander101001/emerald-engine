import 'dotenv/config';
import { decrypt } from './core/security.js';
import { ScoutAgent } from './core/ScoutAgent.js';
import { syncToCloud } from './core/deployer.js';
import { exec } from 'child_process';

// فك التشفير عند التشغيل فقط (في الذاكرة العشوائية)
try {
    if (process.env.GITHUB_TOKEN) global.GITHUB_TOKEN = decrypt(process.env.GITHUB_TOKEN);
    if (process.env.HF_TOKEN) global.HF_TOKEN = decrypt(process.env.HF_TOKEN);
    if (process.env.TELEGRAM_BOT_TOKEN && process.env.TELEGRAM_BOT_TOKEN.includes(':')) {
        process.env.TELEGRAM_BOT_TOKEN = decrypt(process.env.TELEGRAM_BOT_TOKEN);
    }
    if (process.env.TELEGRAM_USER_ID && process.env.TELEGRAM_USER_ID.includes(':')) {
        process.env.TELEGRAM_USER_ID = decrypt(process.env.TELEGRAM_USER_ID);
    }
    console.log("System Security: Decryption successful.");
} catch (e) {
    console.log("System Security: Tokens not encrypted in env (using secure-tokens.json or raw values).");
}

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
