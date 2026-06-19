const crypto = require('crypto');
const fs = require('fs');
const path = require('path');

const vaultPath = path.resolve(__dirname, '../../api_key');
const secret = process.env.MASTER_KEY || 'SuperSecretKey123';

// Read existing vault
let tokens = {};
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
  tokens = JSON.parse(d);
} catch (e) {
  console.error('Could not decrypt vault, starting fresh:', e.message);
}

// Add new key-value pairs from command line arguments
const args = process.argv.slice(2);
for (let i = 0; i < args.length; i += 2) {
  if (i + 1 < args.length) {
    tokens[args[i]] = args[i + 1];
    console.log(`Added: ${args[i]}=${args[i+1].substring(0,8)}...`);
  }
}

// Re-encrypt and write
const key = crypto.scryptSync(secret, 'salt', 32);
const iv = crypto.randomBytes(16);
const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);
let enc = cipher.update(JSON.stringify(tokens), 'utf8', 'hex');
enc += cipher.final('hex');
const authTag = cipher.getAuthTag();
const blob = Buffer.concat([iv, authTag, Buffer.from(enc, 'hex')]);
fs.writeFileSync(vaultPath, blob);

console.log(`Vault updated: ${Object.keys(tokens).length} tokens total`);
