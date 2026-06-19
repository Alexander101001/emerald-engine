import 'dotenv/config';
import { loadVault } from '../security/apiKeyVault.js';
import { SearchEngine } from './core/SearchEngine.js';
import { syncToCloud } from './core/deployer.js';
import axios from 'axios';
import fs from 'fs';

if (!process.env.SERPER_API_KEY || !process.env.GITHUB_TOKEN) {
    console.error("[HALT] Missing required keys in .env (SERPER_API_KEY, GITHUB_TOKEN)");
    process.exit(1);
}

const CONFIG = {
    ollamaUrl: process.env.OLLAMA_URL || 'http://localhost:11434/api/generate',
    checkInterval: 1000
};

const searchEngine = new SearchEngine(process.env.SERPER_API_KEY);

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

    console.log("[DIAGNOSTIC] All checks passed.");
}

async function saasFactory() {
    console.log("[FACTORY] Starting SaaS Factory Cycle...");

    const trends = await searchEngine.search("trending software tools 2026");
    if (!trends || trends.length === 0) {
        return;
    }

    const ollamaPayload = {
        model: 'qwen2.5:1.5b',
        prompt: `Create a viral landing page HTML code for: ${trends[0].title}. Include space for AdSense and affiliate links. Return only valid HTML.`,
        stream: false
    };

    try {
        const response = await axios.post(CONFIG.ollamaUrl, ollamaPayload);
        const content = response.data?.response || '';

        if (!fs.existsSync('./public')) {
            fs.mkdirSync('./public');
        }
        fs.writeFileSync('./public/index.html', content);
        console.log("[FACTORY] Landing page generated for:", trends[0].title);

        await syncToCloud();
        console.log("[FACTORY] Page deployed to GitHub.");
    } catch (error) {
        console.error("[FACTORY] Error:", error.message);
    }
}

async function runCycle() {
    try {
        console.log("[CYCLE] Starting cycle at:", new Date().toISOString());
        await saasFactory();
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
