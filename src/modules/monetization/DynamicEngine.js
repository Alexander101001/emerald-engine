import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

class DynamicMonetizationEngine {
  constructor() {
    this.strategies = [];
    this.pluginsDir = path.join(__dirname, 'strategies');
  }

  async loadStrategies() {
    const files = fs.readdirSync(this.pluginsDir).filter(f => f.endsWith('.js'));
    const imports = files.map(f =>
      import(path.join(this.pluginsDir, f)).then(m => {
        this.strategies.push(m.default);
        console.log(`Loaded strategy: ${f}`);
      })
    );
    await Promise.all(imports);
    console.log(`Loaded ${this.strategies.length} revenue streams dynamically.`);
  }

  async runAll() {
    await Promise.all(this.strategies.map(s => s.execute()));
  }

  async executeAll() {
    return this.runAll();
  }
}

export { DynamicMonetizationEngine };
export default new DynamicMonetizationEngine();
