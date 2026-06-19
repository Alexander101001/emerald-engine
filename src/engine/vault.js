const crypto = require('crypto');
const fs = require('fs');
const path = require('path');

const vaultPath = path.resolve(__dirname, '../../api_key');
const secret = process.env.MASTER_KEY || 'SuperSecretKey123';

try {
  const key = crypto.scryptSync(secret, 'salt', 32);
  const blob = fs.readFileSync(vaultPath);
  const iv = blob.subarray(0, 16);
  const authTag = blob.subarray(16, 32);
  const data = blob.subarray(32).toString('hex');
  const decipher = crypto.createDecipheriv('aes-256-gcm', key, iv);
  decipher.setAuthTag(authTag);
  let d = decipher.update(data, 'hex', 'utf8');
  d += decipher.final('utf8');
  const tokens = JSON.parse(d);
  process.stdout.write(JSON.stringify(tokens));
} catch (e) {
  process.stdout.write(JSON.stringify({}));
}
