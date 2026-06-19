import axios from 'axios';
import fs from 'fs';
import path from 'path';
import ollama from './ollamaConnector.js';
import { ProxyManager } from '../../services/proxyManager.js';

export class ScoutAgent {
    constructor() {
        this.proxy = new ProxyManager();
        this.searchEndpoints = {
            github: 'https://api.github.com/search/repositories?q=topic:saas-strategy+language:javascript',
        };
    }

    async getSearchQueries() {
        const response = await ollama.generateThought("Give me 3 profitable SaaS strategy keywords");
        if (!response) return ['saas', 'monetization', 'revenue'];
        return response.split('\n').filter(k => k.trim());
    }

    async scout() {
        console.log("[SCOUT] Searching for new strategies...");
        try {
            const proxyName = await this.proxy.getRoute('stealth');
            console.log(`[SCOUT] Routing through ${proxyName}`);
            const response = await axios.get(this.searchEndpoints.github);
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
