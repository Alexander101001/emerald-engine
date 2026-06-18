import { createCipheriv, createDecipheriv, randomBytes, createHash, timingSafeEqual } from 'crypto';
import { readFileSync, writeFileSync, existsSync, unlinkSync, appendFileSync, mkdirSync } from 'fs';
import { resolve, dirname, basename, join } from 'path';
import { fileURLToPath } from 'url';
import { createInterface } from 'readline';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');

const ALGO = 'aes-256-gcm';
const IV_LEN = 16;
const SALT = 'emerald-v1-salt';

const IS_CI = process.env.EMERALD_CI === '1' || process.env.CI === 'true';
const TMPDIR = process.env.TMPDIR || (IS_CI ? '/tmp/emerald-ci' : join(PROJECT_ROOT, '.tmp-emerald'));

function ensureTempDir() {
  if (!existsSync(TMPDIR)) {
    mkdirSync(TMPDIR, { recursive: true, mode: 0o700 });
  }
}

function getTempPath(fileName) {
  ensureTempDir();
  return join(TMPDIR, fileName);
}

export function deriveKey(passphrase) {
  if (!passphrase || passphrase.length < 12) {
    throw new Error('EMERALD_KEY must be at least 12 characters');
  }
  return createHash('sha256').update(passphrase + SALT).digest();
}

export function encrypt(data, passphrase) {
  const key = deriveKey(passphrase);
  const iv = randomBytes(IV_LEN);
  const cipher = createCipheriv(ALGO, key, iv);
  const input = typeof data === 'string' ? data : JSON.stringify(data);
  let encrypted = cipher.update(input, 'utf-8', 'hex');
  encrypted += cipher.final('hex');
  const tag = cipher.getAuthTag();
  return { encrypted, iv: iv.toString('hex'), tag: tag.toString('hex') };
}

export function decrypt(payload, passphrase) {
  const key = deriveKey(passphrase);
  const iv = Buffer.from(payload.iv, 'hex');
  const tag = Buffer.from(payload.tag, 'hex');
  const decipher = createDecipheriv(ALGO, key, iv);
  decipher.setAuthTag(tag);
  let decrypted = decipher.update(payload.encrypted, 'hex', 'utf-8');
  decrypted += decipher.final('utf-8');
  return decrypted;
}

export function encryptEnvFile(envPath, passphrase, outputPath) {
  if (!existsSync(envPath)) throw new Error(`File not found: ${envPath}`);
  const content = readFileSync(envPath, 'utf-8');
  const payload = encrypt(content, passphrase);

  const out = outputPath || envPath + '.encrypted';
  writeFileSync(out, JSON.stringify(payload, null, 2), 'utf-8');
  console.log(`[crypto] Encrypted ${basename(envPath)} → ${basename(out)}`);

  const gitignorePath = resolve(dirname(envPath), '.gitignore');
  const entry = basename(out);
  if (existsSync(gitignorePath)) {
    const gitignore = readFileSync(gitignorePath, 'utf-8');
    if (!gitignore.includes(entry)) {
      appendFileSync(gitignorePath, `\n# Encrypted env\n${entry}\n`);
    }
  }

  const checksum = createHash('sha256').update(content).digest('hex');
  writeFileSync(out + '.sha256', checksum, 'utf-8');
  return { output: out, checksum };
}

export function decryptEnvFile(encryptedPath, passphrase, outputPath) {
  if (!existsSync(encryptedPath)) throw new Error(`File not found: ${encryptedPath}`);
  const raw = readFileSync(encryptedPath, 'utf-8');
  const payload = JSON.parse(raw);
  const decrypted = decrypt(payload, passphrase);

  const out = outputPath || encryptedPath.replace('.encrypted', '');
  const tempOut = getTempPath(basename(out));
  writeFileSync(tempOut, decrypted, { mode: 0o600 });
  writeFileSync(out, decrypted, { mode: 0o600 });
  console.log(`[crypto] Decrypted ${basename(encryptedPath)} → ${basename(out)} (temp copy: ${basename(tempOut)})`);
  return out;
}

export function verifyChecksum(encryptedPath) {
  const shaPath = encryptedPath + '.sha256';
  if (!existsSync(shaPath)) return false;
  const raw = readFileSync(encryptedPath, 'utf-8');
  const payload = JSON.parse(raw);
  const content = decrypt(payload, getEmitterKey());

  const stored = readFileSync(shaPath, 'utf-8').trim();
  const computed = createHash('sha256').update(content).digest('hex');
  if (computed.length !== stored.length) return false;
  return timingSafeEqual(Buffer.from(computed), Buffer.from(stored));
}

export function getEmitterKey() {
  const key = process.env.EMERALD_KEY || '';
  if (!key && IS_CI) {
    throw new Error('EMERALD_KEY not set in CI environment');
  }
  return key;
}

export function promptForPassphrase() {
  const rl = createInterface({ input: process.stdin, output: process.stdout });
  return new Promise(resolve => {
    rl.question('EMERALD_KEY passphrase (min 12 chars): ', answer => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

export function secureWipe(filePath, passes = 3) {
  if (!existsSync(filePath)) return;
  try {
    const stat = { size: readFileSync(filePath).length };
    for (let i = 0; i < passes; i++) {
      writeFileSync(filePath, randomBytes(stat.size));
    }
    unlinkSync(filePath);
    console.log(`[crypto] Secure-wiped ${basename(filePath)} (${passes} passes)`);
  } catch (e) {
    console.error(`[crypto] Wipe failed for ${filePath}: ${e.message}`);
  }
}

export function encryptAllEnvFiles(passphrase) {
  const envFiles = [
    resolve(PROJECT_ROOT, '.env'),
    resolve(PROJECT_ROOT, '.env.local'),
    resolve(PROJECT_ROOT, '.env.production'),
    resolve(PROJECT_ROOT, '.env.staging'),
  ];
  const results = [];
  for (const f of envFiles) {
    if (existsSync(f)) {
      try {
        results.push(encryptEnvFile(f, passphrase));
      } catch (e) {
        console.error(`[crypto] Failed ${f}: ${e.message}`);
      }
    }
  }
  return results;
}

export function secureWipeTempDir() {
  if (!existsSync(TMPDIR)) return;
  console.log(`[crypto] Secure-wiping temp directory: ${TMPDIR}`);
  try {
    const entries = readdirSync(TMPDIR) || [];
    for (const entry of entries) {
      const full = join(TMPDIR, entry);
      try {
        const stat = statSync(full);
        if (stat.isDirectory()) {
          const inner = readdirSync(full);
          for (const i of inner) secureWipe(join(full, i), 1);
          unlinkSync(full);
        } else {
          secureWipe(full, 1);
        }
      } catch { unlinkSync(full); }
    }
    unlinkSync(TMPDIR);
    console.log(`[crypto] Temp directory removed: ${TMPDIR}`);
  } catch (e) {
    console.error(`[crypto] Temp dir wipe failed: ${e.message}`);
  }
}

if (process.argv[1]?.includes('crypto-config')) {
  const mode = process.argv[2];
  const key = process.env.EMERALD_KEY || process.argv[3];

  if (mode === 'encrypt') {
    if (!key) { console.error('Usage: EMERALD_KEY=... node crypto-config.js encrypt'); process.exit(1); }
    encryptAllEnvFiles(key);
  } else if (mode === 'decrypt') {
    if (!key) { console.error('Usage: EMERALD_KEY=... node crypto-config.js decrypt'); process.exit(1); }
    const files = ['.env.encrypted', '.env.local.encrypted', '.env.production.encrypted', '.env.staging.encrypted'];
    for (const f of files) {
      const p = resolve(PROJECT_ROOT, f);
      if (existsSync(p)) decryptEnvFile(p, key);
    }
  } else if (mode === 'wipe') {
    secureWipeTempDir();
  } else if (mode === 'verify') {
    const files = ['.env.encrypted', '.env.local.encrypted', '.env.production.encrypted'];
    let allOk = true;
    for (const f of files) {
      const p = resolve(PROJECT_ROOT, f);
      if (existsSync(p)) {
        const ok = verifyChecksum(p);
        console.log(`${f}: ${ok ? 'PASS' : 'FAIL'}`);
        if (!ok) allOk = false;
      }
    }
    process.exit(allOk ? 0 : 1);
  } else {
    console.log('Usage: node src/security/crypto-config.js [encrypt|decrypt|verify|wipe]');
  }
}

export default {
  encrypt, decrypt, encryptEnvFile, decryptEnvFile,
  deriveKey, secureWipe, verifyChecksum, getEmitterKey,
  secureWipeTempDir, getTempPath, TMPDIR,
};
