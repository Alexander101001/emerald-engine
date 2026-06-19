import crypto from 'crypto';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const VAULT_PATH = path.resolve(__dirname, '../../api_key');
const SECRET = process.env.MASTER_KEY || 'SuperSecretKey123';
const KEY = crypto.scryptSync(SECRET, 'salt', 32);
const ALGO = 'aes-256-gcm';

export function encryptVault(tokens) {
  const iv = crypto.randomBytes(16);
  const cipher = crypto.createCipheriv(ALGO, KEY, iv);
  let enc = cipher.update(JSON.stringify(tokens), 'utf8', 'hex');
  enc += cipher.final('hex');
  const blob = Buffer.concat([iv, cipher.getAuthTag(), Buffer.from(enc, 'hex')]);
  fs.writeFileSync(VAULT_PATH, blob);
  console.log('[VAULT] Encrypted → api_key');
}

export function decryptVault() {
  if (!fs.existsSync(VAULT_PATH)) return {};
  const blob = fs.readFileSync(VAULT_PATH);
  const iv = blob.subarray(0, 16);
  const authTag = blob.subarray(16, 32);
  const data = blob.subarray(32).toString('hex');
  const decipher = crypto.createDecipheriv(ALGO, KEY, iv);
  decipher.setAuthTag(authTag);
  let d = decipher.update(data, 'hex', 'utf8');
  d += decipher.final('utf8');
  return JSON.parse(d);
}

export function loadVault() {
  const tokens = decryptVault();
  for (const [key, val] of Object.entries(tokens)) {
    if (val && !process.env[key]) {
      process.env[key] = val;
    }
  }
  const count = Object.keys(tokens).length;
  console.log(`[VAULT] ${count} tokens loaded into memory (invisible)`);
}
