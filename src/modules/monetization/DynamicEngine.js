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

  loadStrategies() {
    const files = fs.readdirSync(this.pluginsDir);
    files.forEach(file => {
      const strategy = import(path.join(this.pluginsDir, file));
      this.strategies.push(strategy);
    });
    console.log(`Loaded ${this.strategies.length} revenue streams dynamically.`);
  }

  async runAll() {
    await Promise.all(this.strategies.map(s => s.execute()));
  }
}

export default new DynamicMonetizationEngine();
