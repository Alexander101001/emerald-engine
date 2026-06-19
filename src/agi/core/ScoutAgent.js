import axios from 'axios';
import fs from 'fs';
import path from 'path';
import ollama from './ollamaConnector.js';
import { SearchEngine } from './SearchEngine.js';
import { ProxyManager } from '../../services/proxyManager.js';

export class ScoutAgent {
    constructor() {
        this.proxy = new ProxyManager();
        this.searcher = new SearchEngine(process.env.SERPER_API_KEY || '');
        this.searchEndpoints = {
            github: 'https://api.github.com/search/repositories?q=saas+monetization+template+language:javascript&sort=stars&order=desc',
        };
    }

    async getSearchQueries() {
        const response = await ollama.generateThought("Give me 3 profitable SaaS strategy keywords");
        if (response) return response.split('\n').filter(k => k.trim());
        const web = await this.searcher.search("profitable SaaS monetization strategies 2026");
        if (web.length > 0) return web.map(r => r.title).slice(0, 3);
        return ['saas', 'monetization', 'revenue'];
    }

    async scout() {
        console.log("[SCOUT] Searching for new strategies...");
        try {
            const proxyName = await this.proxy.getRoute('stealth');
            console.log(`[SCOUT] Routing through ${proxyName}`);
            const ghToken = process.env.GITHUB_TOKEN || '';
            const headers = ghToken ? { Authorization: `Bearer ${ghToken}` } : {};
            const response = await axios.get(this.searchEndpoints.github, { headers });
            const items = response.data.items;
            
            for (const item of items) {
                await this.integrate(item);
            }
        } catch (error) {
            console.error("[SCOUT] Error scanning:", error.message);
        }
    }

    async integrate(repo) {
        const targetPath = path.join(process.cwd(), 'src/modules/monetization/strategies/dynamic', `${repo.name}.js`);
        
        if (!fs.existsSync(targetPath)) {
            console.log(`[INTEGRATOR] Found new strategy: ${repo.name}. Downloading...`);
            fs.writeFileSync(targetPath, `// Auto-fetched from ${repo.html_url}\nexport default { ... }`);
        }
    }
}
