import { spawn } from 'child_process';
import { appendFileSync, existsSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const ERROR_LOG = resolve(PROJECT_ROOT, 'emerald_error.log');

function _logError(msg) {
  const line = `[${new Date().toISOString()}] [UPTIME-WATCHDOG] ${msg}\n`;
  try { appendFileSync(ERROR_LOG, line); } catch {}
}

export class UptimeWatchdog {
  constructor() {
    this._checks = new Map();
    this._healthTimer = null;
    this._active = false;
    this._restartCount = 0;
    this._lastRestartAt = null;
    this._coreProcess = null;
  }

  activate() {
    this._active = true;
    this._startHealthCheck();
    logger.info('uptime-watchdog: activated — monitoring core subsystems');
    return { active: true };
  }

  registerCheck(name, checkFn, intervalMs = 30000) {
    this._checks.set(name, {
      fn: checkFn,
      interval: intervalMs,
      lastOk: Date.now(),
      failures: 0,
    });
    logger.info(`uptime-watchdog: registered check "${name}" (${intervalMs}ms)`);
  }

  _startHealthCheck() {
    if (this._healthTimer) clearInterval(this._healthTimer);
    this._healthTimer = setInterval(() => this._runChecks(), 15000);
  }

  async _runChecks() {
    for (const [name, check] of this._checks) {
      try {
        const ok = await check.fn();
        if (ok) {
          check.lastOk = Date.now();
          check.failures = 0;
        } else {
          check.failures++;
          _logError(`health check "${name}" failed (${check.failures}x)`);
          if (check.failures >= 3) {
            await this._restartSubsystem(name);
          }
        }
      } catch (e) {
        check.failures++;
        _logError(`health check "${name}" error: ${e.message}`);
        if (check.failures >= 3) {
          await this._restartSubsystem(name);
        }
      }
    }
  }

  async _restartSubsystem(name) {
    this._restartCount++;
    this._lastRestartAt = Date.now();
    _logError(`restarting subsystem "${name}" — attempt #${this._restartCount}`);
    try {
      const { default: lifecycle } = await import('./lifecycle.js');
      if (name === 'neural-core') {
        const { default: neuralCore } = await import('./neural-core.js');
        neuralCore.shutdown();
        await neuralCore.awaken();
      } else if (name === 'telegram-bridge') {
        const { default: telegram } = await import('./agi-telegram-bridge.js');
        telegram.reset();
        const { default: orchestrator } = await import('./orchestrator.js');
        const cfg = orchestrator.getTelegramConfig();
        if (cfg) {
          await telegram.initialize(cfg.bot_token, cfg.user_id, cfg.channel_id, orchestrator.getTelegramBotId());
        }
      } else if (name === 'aegis') {
        const { default: aegis } = await import('./aegis.js');
        const { default: orchestrator } = await import('./orchestrator.js');
        aegis.stopHeartbeat();
        await aegis.activate(process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026');
        await orchestrator.loadVault(process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026');
      }
    } catch (e) {
      _logError(`restart of "${name}" failed: ${e.message}`);
    }
  }

  restartCoreProcess(scriptPath = 'src/index.js') {
    const fullPath = resolve(PROJECT_ROOT, scriptPath);
    if (!existsSync(fullPath)) {
      _logError(`cannot restart — script not found: ${fullPath}`);
      return;
    }
    if (this._coreProcess) {
      this._coreProcess.kill('SIGTERM');
      this._coreProcess = null;
    }
    _logError(`launching core process: ${scriptPath}`);
    this._coreProcess = spawn('node', [fullPath, '--agi', '--build'], {
      cwd: PROJECT_ROOT,
      stdio: ['ignore', 'pipe', 'pipe'],
      env: { ...process.env, EMERALD_BUILD: '1' },
    });
    this._coreProcess.stdout.on('data', (d) => process.stdout.write(d));
    this._coreProcess.stderr.on('data', (d) => process.stderr.write(d));
    this._coreProcess.on('exit', (code) => {
      _logError(`core process exited (code ${code}) — restarting in 5s`);
      setTimeout(() => this.restartCoreProcess(scriptPath), 5000);
    });
    this._restartCount++;
  }

  getStatus() {
    return {
      active: this._active,
      registeredChecks: Array.from(this._checks.keys()),
      restartCount: this._restartCount,
      lastRestartAt: this._lastRestartAt ? new Date(this._lastRestartAt).toISOString() : null,
      processAlive: this._coreProcess !== null,
    };
  }

  deactivate() {
    this._active = false;
    if (this._healthTimer) {
      clearInterval(this._healthTimer);
      this._healthTimer = null;
    }
    if (this._coreProcess) {
      this._coreProcess.kill('SIGTERM');
      this._coreProcess = null;
    }
    logger.info('uptime-watchdog: deactivated');
  }
}

const uptimeWatchdog = new UptimeWatchdog();
export default uptimeWatchdog;
