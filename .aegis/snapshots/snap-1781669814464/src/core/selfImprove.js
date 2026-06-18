import { readFileSync, writeFileSync, readdirSync, statSync } from 'fs';
import { join, extname } from 'path';
import { fileURLToPath } from 'url';
import { dirname } from 'path';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));

const MAX_FILES_TO_LINT = 100;
const MAX_LINT_RUNTIME_MS = 30000;
const MIN_CYCLE_INTERVAL_MS = 60000;
const MAX_SUGGESTIONS_REPORTED = 20;

let lastRunAt = 0;

export class SelfImprover {
  constructor() {
    this.metrics = { lintErrors: 0, deprecations: [], suggestions: [], skipped: 0, runtimeMs: 0 };
  }

  runLint() {
    const start = Date.now();
    const elapsed = Date.now() - lastRunAt;

    if (elapsed < MIN_CYCLE_INTERVAL_MS && lastRunAt > 0) {
      logger.warn(`selfImprove: rate-limited (${Math.round(elapsed / 1000)}s since last run, need ${MIN_CYCLE_INTERVAL_MS / 1000}s)`);
      this.metrics.skipped++;
      return { ...this.metrics, rateLimited: true };
    }

    logger.info('selfImprove: running lint');
    const srcDir = join(__dirname, '..');
    let files = this._walk(srcDir).filter(f => extname(f) === '.js');

    if (files.length > MAX_FILES_TO_LINT) {
      logger.warn(`selfImprove: truncating ${files.length} files to ${MAX_FILES_TO_LINT}`);
      files = files.slice(0, MAX_FILES_TO_LINT);
    }

    let errors = 0;
    let suggestionCount = 0;

    for (const file of files) {
      if (Date.now() - start > MAX_LINT_RUNTIME_MS) {
        logger.warn(`selfImprove: runtime limit (${MAX_LINT_RUNTIME_MS}ms) reached, stopping`);
        this.metrics.skipped += files.length - files.indexOf(file);
        break;
      }

      try {
        const content = readFileSync(file, 'utf-8');
        const lines = content.split('\n');

        if (content.includes('console.log') && !content.includes('// eslint-disable')) {
          this.metrics.suggestions.push(`${file}: replace console.log with logger`);
          errors++;
          suggestionCount++;
        }

        lines.forEach((line, i) => {
          if (suggestionCount >= MAX_SUGGESTIONS_REPORTED) return;
          if (line.length > 200) {
            this.metrics.suggestions.push(`${file}:${i + 1} line too long (${line.length})`);
            errors++;
            suggestionCount++;
          }
          if (/var\s/.test(line)) {
            this.metrics.suggestions.push(`${file}:${i + 1} use const/let instead of var`);
            errors++;
            suggestionCount++;
          }
          if ((line.includes('TODO') || line.includes('FIXME')) && !line.includes('// eslint')) {
            this.metrics.deprecations.push(`${file}:${i + 1} has unresolved TODO/FIXME`);
          }
        });

        if (content.includes('require(') && !content.includes('import ')) {
          this.metrics.suggestions.push(`${file}: convert require() to import`);
          errors++;
          suggestionCount++;
        }
      } catch (e) {
        logger.warn(`selfImprove: error reading ${file} - ${e.message}`);
      }
    }

    this.metrics.lintErrors = errors;
    this.metrics.runtimeMs = Date.now() - start;
    lastRunAt = Date.now();
    logger.info(`selfImprove: lint complete - ${errors} issues in ${this.metrics.runtimeMs}ms`);
    return this.metrics;
  }

  autoFixLint() {
    const report = this.runLint();
    if (report.rateLimited) return report;

    const srcDir = join(__dirname, '..');
    let files = this._walk(srcDir).filter(f => extname(f) === '.js');
    if (files.length > MAX_FILES_TO_LINT) files = files.slice(0, MAX_FILES_TO_LINT);

    let fixed = 0;
    for (const file of files) {
      try {
        let content = readFileSync(file, 'utf-8');
        let changed = false;

        if (!content.includes("import logger from '../utils/logger.js'") &&
            !content.includes("import logger from './utils/logger.js'") &&
            content.includes('console.log')) {
          const rel = file.includes('utils') ? './logger.js' : '../utils/logger.js';
          content = `import logger from '${rel}';\n${content}`;
          content = content.replace(/console\.log\(/g, 'logger.info(');
          changed = true;
        }

        content = content.replace(/\bvar\s+/g, 'const ');
        if (content.includes('const ')) changed = true;

        if (changed) {
          writeFileSync(file, content, 'utf-8');
          logger.info(`selfImprove: auto-fixed ${file}`);
          fixed++;
        }
      } catch (e) {
        logger.warn(`selfImprove: fix error on ${file} - ${e.message}`);
      }
    }

    logger.info(`selfImprove: auto-fixed ${fixed} files`);
    return { ...report, filesFixed: fixed };
  }

  _walk(dir) {
    const results = [];
    try {
      const list = readdirSync(dir);
      for (const entry of list) {
        const full = join(dir, entry);
        try {
          const stat = statSync(full);
          if (stat.isDirectory() && entry !== 'node_modules' && entry !== '.git') {
            results.push(...this._walk(full));
          } else if (stat.isFile()) {
            results.push(full);
          }
        } catch {
          continue;
        }
      }
    } catch {
      return [];
    }
    return results;
  }
}

if (process.argv[1] && process.argv[1].includes('selfImprove')) {
  const improver = new SelfImprover();
  if (process.argv.includes('--fix')) {
    const r = improver.autoFixLint();
    console.log(JSON.stringify(r, null, 2));
  } else if (process.argv.includes('--lint')) {
    const r = improver.runLint();
    console.log(JSON.stringify(r, null, 2));
  } else if (process.argv.includes('--watch')) {
    const run = () => {
      const r = improver.runLint();
      console.log(JSON.stringify(r, null, 2));
    };
    run();
    setInterval(run, 300000);
  }
}

const improver = new SelfImprover();
export default improver;
