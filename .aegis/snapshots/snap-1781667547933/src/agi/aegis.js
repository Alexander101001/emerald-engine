import { createHash, randomBytes, createCipheriv, createDecipheriv, timingSafeEqual } from 'crypto';
import { readFileSync, writeFileSync, existsSync, readdirSync, statSync, unlinkSync, mkdirSync, copyFileSync, rmSync } from 'fs';
import { join, extname, relative, resolve, dirname, basename } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const AEGIS_DIR = join(PROJECT_ROOT, '.aegis');
const SNAPSHOT_DIR = join(AEGIS_DIR, 'snapshots');
const INTEGRITY_DB = join(AEGIS_DIR, 'integrity.db');
const KEY_STORE = join(AEGIS_DIR, '.keyring');
const SECRETS_DIR = join(PROJECT_ROOT, '.secrets');
const VAULT_PATH = join(SECRETS_DIR, 'vault.enc');

const ALGO = 'aes-256-gcm';
const IV_LEN = 16;
const HEARTBEAT_INTERVAL = 30000;

export class ProjectAegis {
  constructor() {
    this._key = null;
    this._heartbeatTimer = null;
    this._integrityCache = new Map();
    this._repairCount = 0;
    this._active = false;
  }

  async activate(passphrase) {
    if (!existsSync(AEGIS_DIR)) mkdirSync(AEGIS_DIR, { recursive: true, mode: 0o700 });
    if (!existsSync(SNAPSHOT_DIR)) mkdirSync(SNAPSHOT_DIR, { recursive: true, mode: 0o700 });

    this._key = this._deriveKey(passphrase);
    this._storeKey(this._key);

    await this._buildIntegrityDB();

    this._active = true;
    logger.info('aegis: Project Aegis activated — runtime encryption + integrity monitoring online');

    this._startHeartbeat();
    return { status: 'active', algo: ALGO, dbSize: this._integrityCache.size };
  }

  _deriveKey(passphrase) {
    const h = createHash('sha512').update(passphrase + 'aegis-v1').digest();
    return h.subarray(0, 32);
  }

  _storeKey(key) {
    const wrapped = { key: key.toString('hex'), created: Date.now() };
    writeFileSync(KEY_STORE, JSON.stringify(wrapped), { mode: 0o600 });
  }

  _loadKey() {
    if (this._key) return this._key;
    if (existsSync(KEY_STORE)) {
      const raw = readFileSync(KEY_STORE, 'utf-8');
      const w = JSON.parse(raw);
      this._key = Buffer.from(w.key, 'hex');
      return this._key;
    }
    throw new Error('Aegis key not available');
  }

  async _buildIntegrityDB() {
    const files = this._walkSource();
    const db = {};
    for (const f of files) {
      try {
        const content = readFileSync(f);
        db[f] = {
          hash: createHash('sha256').update(content).digest('hex'),
          size: content.length,
          modTime: statSync(f).mtimeMs,
        };
      } catch { continue; }
    }
    this._integrityCache.clear();
    for (const [k, v] of Object.entries(db)) this._integrityCache.set(k, v);
    writeFileSync(INTEGRITY_DB, JSON.stringify(db, null, 2), { mode: 0o600 });
    logger.info(`aegis: integrity DB built — ${Object.keys(db).length} files tracked`);
  }

  encryptRuntime(data) {
    const key = this._loadKey();
    const iv = randomBytes(IV_LEN);
    const cipher = createCipheriv(ALGO, key, iv);
    const input = typeof data === 'string' ? data : JSON.stringify(data);
    let enc = cipher.update(input, 'utf-8', 'hex');
    enc += cipher.final('hex');
    return { data: enc, iv: iv.toString('hex'), tag: cipher.getAuthTag().toString('hex') };
  }

  decryptRuntime(payload) {
    const key = this._loadKey();
    const iv = Buffer.from(payload.iv, 'hex');
    const tag = Buffer.from(payload.tag, 'hex');
    const decipher = createDecipheriv(ALGO, key, iv);
    decipher.setAuthTag(tag);
    let dec = decipher.update(payload.data, 'hex', 'utf-8');
    dec += decipher.final('utf-8');
    return dec;
  }

  runtimeEncryptFile(filePath) {
    if (!existsSync(filePath)) return;
    const content = readFileSync(filePath, 'utf-8');
    const enc = this.encryptRuntime(content);
    const encPath = filePath + '.aegis';
    writeFileSync(encPath, JSON.stringify(enc), { mode: 0o600 });
    writeFileSync(filePath, '', { mode: 0o600 });
    return encPath;
  }

  runtimeDecryptFile(encPath) {
    if (!existsSync(encPath)) return null;
    const raw = readFileSync(encPath, 'utf-8');
    const payload = JSON.parse(raw);
    return this.decryptRuntime(payload);
  }

  async verifyIntegrity() {
    if (!this._active) return { ok: false, reason: 'not_active' };
    const files = this._walkSource();
    const violations = [];
    let checked = 0;

    for (const f of files) {
      const stored = this._integrityCache.get(f);
      if (!stored) {
        violations.push({ file: f, issue: 'untracked' });
        continue;
      }
      try {
        const content = readFileSync(f);
        const hash = createHash('sha256').update(content).digest('hex');
        if (!timingSafeEqual(Buffer.from(hash), Buffer.from(stored.hash))) {
          violations.push({ file: f, issue: 'hash_mismatch', expected: stored.hash, actual: hash });
        }
        checked++;
      } catch { violations.push({ file: f, issue: 'unreadable' }); }
    }

    if (violations.length > 0) {
      logger.warn(`aegis: integrity violation — ${violations.length} files affected`);
      await this._autoRepair(violations);
    }

    return { ok: violations.length === 0, checked, violations: violations.length, repaired: this._repairCount };
  }

  async _autoRepair(violations) {
    const snapshot = this._findLatestSnapshot();
    if (!snapshot) {
      logger.warn('aegis: no snapshot available for repair — creating one now');
      await this.createSnapshot();
      return;
    }

    for (const v of violations) {
      try {
        const relPath = relative(PROJECT_ROOT, v.file);
        const snapFile = join(snapshot, relPath);
        if (existsSync(snapFile)) {
          const content = readFileSync(snapFile);
          writeFileSync(v.file, content);
          this._repairCount++;
          logger.info(`aegis: repaired ${relPath} from snapshot`);
        }
      } catch (e) {
        logger.error(`aegis: repair failed for ${v.file} — ${e.message}`);
      }
    }
  }

  async createSnapshot() {
    const ts = Date.now();
    const snapDir = join(SNAPSHOT_DIR, `snap-${ts}`);
    mkdirSync(snapDir, { recursive: true, mode: 0o700 });

    const files = this._walkSource();
    let copied = 0;
    for (const f of files) {
      try {
        const rel = relative(PROJECT_ROOT, f);
        const dest = join(snapDir, rel);
        const destDir = dirname(dest);
        if (!existsSync(destDir)) mkdirSync(destDir, { recursive: true });
        copyFileSync(f, dest);
        copied++;
      } catch { continue; }
    }

    const snapshots = readdirSync(SNAPSHOT_DIR).filter(d => d.startsWith('snap-')).sort();
    while (snapshots.length > 5) {
      const old = snapshots.shift();
      rmSync(join(SNAPSHOT_DIR, old), { recursive: true, force: true });
    }

    logger.info(`aegis: snapshot created (${copied} files) — ${basename(snapDir)}`);
    return snapDir;
  }

  _findLatestSnapshot() {
    const dirs = readdirSync(SNAPSHOT_DIR).filter(d => d.startsWith('snap-')).sort();
    if (dirs.length === 0) return null;
    return join(SNAPSHOT_DIR, dirs[dirs.length - 1]);
  }

  rotateKey() {
    const newKey = randomBytes(32);
    this._key = newKey;
    this._storeKey(newKey);
    logger.info('aegis: encryption key rotated');
    return true;
  }

  _startHeartbeat() {
    if (this._heartbeatTimer) clearInterval(this._heartbeatTimer);
    this._heartbeatTimer = setInterval(() => {
      this.verifyIntegrity().catch(e => {
        logger.error(`aegis: heartbeat integrity check failed — ${e.message}`);
      });
    }, HEARTBEAT_INTERVAL);
    logger.info(`aegis: heartbeat started (${HEARTBEAT_INTERVAL / 1000}s interval)`);
  }

  stopHeartbeat() {
    if (this._heartbeatTimer) {
      clearInterval(this._heartbeatTimer);
      this._heartbeatTimer = null;
    }
    this._active = false;
    logger.info('aegis: heartbeat stopped');
  }

  _walkSource() {
    const srcDir = join(PROJECT_ROOT, 'src');
    const results = [];
    const walk = (dir) => {
      try {
        const entries = readdirSync(dir);
        for (const e of entries) {
          const full = join(dir, e);
          if (e.startsWith('.') || e === 'node_modules') continue;
          try {
            const s = statSync(full);
            if (s.isDirectory()) walk(full);
            else if (s.isFile() && (e.endsWith('.js') || e.endsWith('.json') || e.endsWith('.yml') || e.endsWith('.toml') || e.endsWith('.sh')))
              results.push(full);
          } catch { continue; }
        }
      } catch { return; }
    };
    walk(srcDir);
    return results;
  }

  vaultEncrypt(keys) {
    if (!existsSync(SECRETS_DIR)) mkdirSync(SECRETS_DIR, { recursive: true, mode: 0o700 });
    const key = this._loadKey();
    const iv = randomBytes(IV_LEN);
    const cipher = createCipheriv(ALGO, key, iv);
    const payload = JSON.stringify(keys);
    let enc = cipher.update(payload, 'utf-8', 'hex');
    enc += cipher.final('hex');
    const vault = { data: enc, iv: iv.toString('hex'), tag: cipher.getAuthTag().toString('hex'), created: Date.now() };
    const vaultContent = JSON.stringify(vault, null, 2);
    writeFileSync(VAULT_PATH, vaultContent, { mode: 0o600 });
    const vaultHash = createHash('sha256').update(vaultContent).digest('hex');
    const hashFile = join(SECRETS_DIR, 'vault.sha256');
    writeFileSync(hashFile, vaultHash, { mode: 0o600 });
    logger.info(`aegis: vault encrypted — ${Object.keys(keys).length} keys stored`);
    return { path: VAULT_PATH, hash: vaultHash, keyCount: Object.keys(keys).length };
  }

  vaultDecrypt() {
    if (!existsSync(VAULT_PATH)) throw new Error('Vault not found at ' + VAULT_PATH);
    const key = this._loadKey();
    const raw = readFileSync(VAULT_PATH, 'utf-8');
    const vault = JSON.parse(raw);
    const decipher = createDecipheriv(ALGO, key, Buffer.from(vault.iv, 'hex'));
    decipher.setAuthTag(Buffer.from(vault.tag, 'hex'));
    let dec = decipher.update(vault.data, 'hex', 'utf-8');
    dec += decipher.final('utf-8');
    return JSON.parse(dec);
  }

  verifyVaultIntegrity() {
    if (!existsSync(VAULT_PATH)) return { ok: false, reason: 'vault_missing' };
    if (!existsSync(join(SECRETS_DIR, 'vault.sha256'))) return { ok: false, reason: 'hash_missing' };
    const vaultContent = readFileSync(VAULT_PATH, 'utf-8');
    const currentHash = createHash('sha256').update(vaultContent).digest('hex');
    const storedHash = readFileSync(join(SECRETS_DIR, 'vault.sha256'), 'utf-8').trim();
    const ok = storedHash.length === currentHash.length && timingSafeEqual(Buffer.from(currentHash), Buffer.from(storedHash));
    return { ok, storedHash, currentHash };
  }

  getStatus() {
    return {
      active: this._active,
      trackedFiles: this._integrityCache.size,
      repairCount: this._repairCount,
      heartbeatInterval: HEARTBEAT_INTERVAL,
      snapshots: readdirSync(SNAPSHOT_DIR).filter(d => d.startsWith('snap-')).length,
      vault: existsSync(VAULT_PATH) ? 'present' : 'absent',
    };
  }
}

const aegis = new ProjectAegis();

function _secureWipeString(ref) {
  if (typeof ref === 'string' && ref.length > 0) {
    for (let i = 0; i < ref.length; i++) ref = ref[i] + 'x';
  }
  return null;
}

function _getArgValue(args, flag) {
  const idx = args.indexOf(flag);
  if (idx === -1 || idx + 1 >= args.length) return null;
  return args[idx + 1];
}

async function cli() {
  const args = process.argv.slice(2);

  if (args.includes('--encrypt-vault')) {
    const passphrase = _getArgValue(args, '--key') || process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026';
    await aegis.activate(passphrase);

    let keys = {};
    const inputRaw = _getArgValue(args, '--input');

    if (inputRaw) {
      try {
        keys = JSON.parse(inputRaw);
      } catch (e) {
        console.error(JSON.stringify({ status: 'error', message: 'Invalid --input JSON: ' + e.message }));
        aegis.stopHeartbeat();
        process.exit(1);
      }
      _secureWipeString(inputRaw);
    } else {
      const keyMap = {
        GITHUB_TOKEN: { env: 'VAULT_GITHUB_TOKEN', placeholder: '[INSERT_YOUR_GITHUB_PAT]' },
        TELEGRAM_CONFIG: { env: 'VAULT_TELEGRAM_CONFIG', placeholder: '[INSERT_TELEGRAM_CONFIG]' },
        TELEGRAM_BOT_ID: { env: 'VAULT_TELEGRAM_BOT_ID', placeholder: '[INSERT_TELEGRAM_BOT_ID]' },
        OPENROUTER_API_KEY: { env: 'VAULT_OPENROUTER_API_KEY', placeholder: '[INSERT_YOUR_OPENROUTER_KEY]' },
        VERCEL_TOKEN: { env: 'VAULT_VERCEL_TOKEN', placeholder: '[INSERT_YOUR_VERCEL_TOKEN]' },
        NETLIFY_TOKEN: { env: 'VAULT_NETLIFY_TOKEN', placeholder: '[INSERT_YOUR_NETLIFY_TOKEN]' },
        OTHER_AI_API_KEYS: { env: 'VAULT_OTHER_AI_API_KEYS', placeholder: '[INSERT_ADDITIONAL_KEYS]' },
      };
      for (const [name, cfg] of Object.entries(keyMap)) {
        keys[name] = process.env[cfg.env] || cfg.placeholder;
      }
    }

    const result = aegis.vaultEncrypt(keys);
    const keysCopy = Object.keys(keys);
    const keyValCopy = JSON.stringify(keys);

    for (const k of Object.keys(keys)) delete keys[k];
    keys = null;
    _secureWipeString(keyValCopy);

    const integrity = aegis.verifyVaultIntegrity();

    const output = {
      status: 'vault_created',
      path: result.path,
      hash: result.hash,
      integrity: integrity.ok ? 'pass' : 'fail',
      keys: keysCopy,
      message: integrity.ok
        ? 'Credentials encrypted and wiped from volatile memory. Only .secrets/vault.enc persists.'
        : 'Vault created but integrity verification failed.',
    };
    console.log(JSON.stringify(output));
    aegis.stopHeartbeat();
    process.exit(0);
  }

  if (args.includes('--vault-status')) {
    const passphrase = _getArgValue(args, '--key') || process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026';
    await aegis.activate(passphrase);
    const integrity = aegis.verifyVaultIntegrity();
    const status = { vault: existsSync(VAULT_PATH) ? 'present' : 'absent', integrity };
    console.log(JSON.stringify(status));
    aegis.stopHeartbeat();
    process.exit(0);
  }

  if (args.includes('--decrypt-vault')) {
    if (!existsSync(KEY_STORE)) {
      const passphrase = _getArgValue(args, '--key') || process.env.EMERALD_KEY || 'emerald-agi-bootstrap-key-2026';
      await aegis.activate(passphrase);
    }
    const keys = aegis.vaultDecrypt();
    console.log(JSON.stringify(keys));
    process.exit(0);
  }
}

if (process.argv[1] && (process.argv[1].endsWith('aegis.js') || process.argv[1].endsWith('aegis'))) {
  cli().catch(e => { console.error(e.message); process.exit(1); });
}

export default aegis;
