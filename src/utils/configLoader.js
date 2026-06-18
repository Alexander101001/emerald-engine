import crypto from 'crypto';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export function loadSecureConfig() {
    try {
        const masterKey = process.env.EMERALD_MASTER_KEY;
        if (!masterKey) {
            throw new Error('Master runtime key is missing.');
        }

        const filePath = path.join(__dirname, '../../secure-config.json');
        if (!fs.existsSync(filePath)) {
            throw new Error('Encrypted payload envelope not found.');
        }

        const rawData = fs.readFileSync(filePath, 'utf8');
        const payload = JSON.parse(rawData);

        const algorithm = 'aes-256-gcm';
        const iv = Buffer.from(payload.iv, 'hex');
        const salt = Buffer.from(payload.salt, 'hex');
        const tag = Buffer.from(payload.tag, 'hex');

        const key = crypto.pbkdf2Sync(masterKey, salt, 100000, 32, 'sha256');
        const decipher = crypto.createDecipheriv(algorithm, key, iv);
        decipher.setAuthTag(tag);

        let decrypted = decipher.update(payload.encryptedData, 'hex', 'utf8');
        decrypted += decipher.final('utf8');

        const config = JSON.parse(decrypted);
        Object.keys(config).forEach((key) => {
            process.env[key] = config[key];
        });

        return true;
    } catch (error) {
        process.exit(1);
    }
}
