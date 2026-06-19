import 'dotenv/config';
import { loadVault } from '../security/apiKeyVault.js';
import axios from 'axios';
import fs from 'fs';

const CONFIG = {
    ollamaUrl: process.env.OLLAMA_URL || 'http://localhost:11434/api/generate',
    checkInterval: 1000
};

async function checkSystem() {
    console.log("[DIAGNOSTIC] Checking environment...");

    if (!fs.existsSync('.env')) {
        throw new Error("SYSTEM_HALT: .env file missing.");
    }

    try {
        await axios.get('http://localhost:11434');
        console.log("[DIAGNOSTIC] Ollama: CONNECTED");
    } catch (e) {
        throw new Error("SYSTEM_HALT: Ollama not reachable at port 11434.");
    }
}

async function runCycle() {
    try {
        const response = await axios.post(CONFIG.ollamaUrl, {
            model: 'qwen2.5:1.5b',
            prompt: 'Analyze current market trends and return strategy.',
            stream: false
        });
        console.log("[SUCCESS] Cycle completed at:", new Date().toISOString());
    } catch (error) {
        console.error("[ERROR] Cycle execution failed:", error.message);
    }
}

async function main() {
    try {
        await checkSystem();
        console.log("[SYSTEM] Initialization complete. Starting loop...");

        loadVault();

        while (true) {
            await runCycle();
            await new Promise(resolve => setTimeout(resolve, CONFIG.checkInterval));
        }
    } catch (err) {
        console.error("[CRITICAL]", err.message);
        process.exit(1);
    }
}

main();
