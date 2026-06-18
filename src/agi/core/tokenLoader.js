import crypto from 'crypto';
import fs from 'fs';
import path from 'path';

const SECRET = process.env.MASTER_KEY || 'SuperSecretKey123';

export function loadTokens() {
  const filePath = path.join(process.cwd(), 'secure-tokens.json');
  if (!fs.existsSync(filePath)) {
    console.warn('tokenLoader: secure-tokens.json not found — using env vars');
    return false;
  }

  try {
    const blob = JSON.parse(fs.readFileSync(filePath, 'utf8'));
    const key = crypto.scryptSync(SECRET, 'salt', 32);
    const decipher = crypto.createDecipheriv('aes-256-gcm', key, Buffer.from(blob.iv, 'hex'));
    decipher.setAuthTag(Buffer.from(blob.authTag, 'hex'));
    let decrypted = decipher.update(blob.data, 'hex', 'utf8');
    decrypted += decipher.final('utf8');
    const tokens = JSON.parse(decrypted);
    for (const [k, v] of Object.entries(tokens)) {
      if (v && !process.env[k]) process.env[k] = v;
    }
    console.log(`tokenLoader: loaded ${Object.keys(tokens).length} tokens into env`);
    return true;
  } catch (e) {
    console.error('tokenLoader: decryption failed —', e.message);
    return false;
  }
}
