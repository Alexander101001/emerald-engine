import fs from 'fs';
import path from 'path';

export class DynamicMonetizationEngine {
    constructor() {
        this.strategies = [];
    }

    async loadStrategies() {
        const dir = path.join(process.cwd(), 'src/modules/monetization/strategies');
        if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
        const files = fs.readdirSync(dir).filter(f => f.endsWith('.js'));
        const imports = files.map(f => import(path.join(dir, f)).then(m => {
            this.strategies.push(m.default);
            console.log(`Loaded strategy: ${f}`);
        }));
        await Promise.all(imports);
        console.log(`MonetizationEngine: ${this.strategies.length} strategies loaded`);
    }

    async executeAll() {
        const results = [];
        for (const strategy of this.strategies) {
            try {
                const r = await strategy.execute();
                results.push(r);
                console.log(`[${strategy.name}] -> ${r.status} (${r.revenue || 0})`);
            } catch (e) {
                console.error(`[${strategy.name}] FAILED: ${e.message}`);
                results.push({ revenue: 0, status: 'error', reason: e.message });
            }
        }
        return results;
    }
}
