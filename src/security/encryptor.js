import crypto from 'crypto';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const TOKENS_PATH = path.join(__dirname, '../../secure-tokens.json');
const SECRET = process.env.MASTER_KEY || 'SuperSecretKey123';

const MODE = process.argv[2] || 'encrypt';

function encrypt() {
  const tokens = {
    GITHUB_TOKEN: process.env.GITHUB_TOKEN || '',
    HF_TOKEN: process.env.HF_TOKEN || '',
    TELEGRAM_TOKEN: process.env.TELEGRAM_BOT_TOKEN || process.env.TELEGRAM_TOKEN || '',
    TELEGRAM_ID: process.env.TELEGRAM_USER_ID || process.env.TELEGRAM_ID || ''
  };

  const iv = crypto.randomBytes(12);
  const cipher = crypto.createCipheriv('aes-256-gcm', crypto.scryptSync(SECRET, 'salt', 32), iv);
  let encrypted = cipher.update(JSON.stringify(tokens), 'utf8', 'hex');
  encrypted += cipher.final('hex');

  const blob = { iv: iv.toString('hex'), authTag: cipher.getAuthTag().toString('hex'), data: encrypted };
  fs.writeFileSync(TOKENS_PATH, JSON.stringify(blob, null, 2));
  console.log('encryptor: tokens encrypted -> secure-tokens.json');
}

function decrypt() {
  if (!fs.existsSync(TOKENS_PATH)) {
    console.error('encryptor: secure-tokens.json not found');
    process.exit(1);
  }
  try {
    const blob = JSON.parse(fs.readFileSync(TOKENS_PATH, 'utf8'));
    const key = crypto.scryptSync(SECRET, 'salt', 32);
    const decipher = crypto.createDecipheriv('aes-256-gcm', key, Buffer.from(blob.iv, 'hex'));
    decipher.setAuthTag(Buffer.from(blob.authTag, 'hex'));
    let d = decipher.update(blob.data, 'hex', 'utf8');
    d += decipher.final('utf8');
    console.log('encryptor: decrypted tokens ->');
    console.log(JSON.stringify(JSON.parse(d), null, 2));
  } catch (e) {
    console.error('encryptor: decryption failed —', e.message);
    process.exit(1);
  }
}

if (MODE === 'decrypt') decrypt();
else encrypt();
