import 'dotenv/config';
import { loadVault } from '../security/apiKeyVault.js';
import { SearchEngine } from './core/SearchEngine.js';
import axios from 'axios';
import fs from 'fs';
import { execSync } from 'child_process';

const CONFIG = {
    ollamaUrl: process.env.OLLAMA_URL || 'http://localhost:11434/api/generate',
    checkInterval: 5000
};

const searchEngine = new SearchEngine(process.env.SERPER_API_KEY || '');

async function isOllamaRunning() {
    try {
        await axios.post(CONFIG.ollamaUrl, {
            model: 'qwen2.5:1.5b',
            prompt: 'ping',
            stream: false
        });
        return true;
    } catch (e) {
        return false;
    }
}

const generateHTML = (trend) => `<!DOCTYPE html>
<html lang="en">
<head>
    <title>${trend.title}</title>
    <meta name="description" content="${trend.desc}">
    <meta name="keywords" content="${trend.keywords}">
</head>
<body>
    <div class="ad-slot">[ADSENSE_CODE_HERE]</div>
    <h1>${trend.title}</h1>
    <p>${trend.content}</p>
</body>
</html>`;

function deploy() {
    try {
        execSync('git add . && git commit -m "Auto-Deploy: New SaaS Page" && git push origin main', { stdio: 'pipe' });
        console.log("[SUCCESS] Deployed to GitHub/HuggingFace.");
    } catch (e) {
        console.log("[ERROR] Deploy failed, check git status.");
    }
}

async function runFactory() {
    if (!(await isOllamaRunning())) {
        console.log("[SYSTEM] Ollama offline. Waiting...");
        return;
    }

    console.log("[FACTORY] Generating SaaS Content...");

    const trends = await searchEngine.search("trending software tools 2026");
    if (!trends || trends.length === 0) {
        console.log("[FACTORY] No trends found this cycle.");
        return;
    }

    const trend = {
        title: trends[0].title,
        desc: trends[0].snippet || 'Discover the best software tools for 2026',
        keywords: trends[0].title.toLowerCase().replace(/\s+/g, ', '),
        content: trends[0].snippet || 'Learn more about this trending software tool.'
    };

    const html = generateHTML(trend);
    if (!fs.existsSync('./public')) fs.mkdirSync('./public');
    fs.writeFileSync('./public/index.html', html);
    console.log(`[FACTORY] Page generated: ${trend.title}`);

    deploy();
}

loadVault();
setInterval(runFactory, CONFIG.checkInterval);
