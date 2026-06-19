import 'dotenv/config';
import { loadVault } from '../security/apiKeyVault.js';
import axios from 'axios';
import fs from 'fs';

loadVault();

const OLLAMA_URL = 'http://localhost:11434/api/generate';
const RETRY_DELAY = 60000;

async function saasFactory() {
    try {
        console.log("[FACTORY] Checking engine...");

        const response = await axios.post(OLLAMA_URL, {
            model: 'qwen2.5:1.5b',
            prompt: 'Generate viral landing page code for SaaS, include ad placeholders.',
            stream: false
        }, { timeout: 10000 });

        console.log("[FACTORY] Engine Active. Generating Assets...");

        const htmlCode = response.data.response;
        if (!fs.existsSync('./public')) fs.mkdirSync('./public');
        fs.writeFileSync('./public/index.html', htmlCode);

        console.log("[FACTORY] Assets Deployed to ./public/index.html");
    } catch (error) {
        console.error("[FACTORY] Engine unreachable. Entering standby mode for 60s.");
        await new Promise(resolve => setTimeout(resolve, RETRY_DELAY));
    }
}

async function main() {
    while (true) {
        await saasFactory();
        await new Promise(resolve => setTimeout(resolve, 5000));
    }
}

main();
