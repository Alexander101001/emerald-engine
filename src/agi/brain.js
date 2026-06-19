import 'dotenv/config';
import { loadVault } from '../security/apiKeyVault.js';
import axios from 'axios';
import fs from 'fs';

async function performSystemCheck() {
    console.log("[SYSTEM] Starting Diagnostic...");

    if (!fs.existsSync('.env')) {
        throw new Error("MISSING_CONFIG_ERROR: .env file not found.");
    }

    if (!process.env.SERPER_API_KEY || process.env.SERPER_API_KEY === 'your_serper_key_here') {
        throw new Error("MISSING_CONFIG_ERROR: SERPER_API_KEY is not set.");
    }

    if (!process.env.GITHUB_TOKEN || process.env.GITHUB_TOKEN === 'your_github_token_here') {
        throw new Error("MISSING_CONFIG_ERROR: GITHUB_TOKEN is not set.");
    }

    try {
        await axios.get('http://localhost:11434');
        console.log("[OK] Ollama is reachable.");
    } catch (e) {
        throw new Error("OLLAMA_SERVICE_DOWN: Please run 'ollama serve' or check supervisor.");
    }

    console.log("[SYSTEM] Diagnostics Passed. Starting Engine.");
}

async function main() {
    loadVault();

    while (true) {
        try {
            await axios.post(process.env.OLLAMA_URL, {
                model: process.env.OLLAMA_MODEL || 'qwen2.5:1.5b',
                prompt: 'execute',
                stream: false
            });
            console.log('Cycle OK');
        } catch (e) {
            console.error('Cycle Fail');
        }
        await new Promise(r => setTimeout(r, 1000));
    }
}

performSystemCheck()
    .then(() => main())
    .catch(err => {
        console.error(err.message);
        process.exit(1);
    });
