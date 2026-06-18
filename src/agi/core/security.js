import crypto from 'crypto';

const ALGORITHM = 'aes-256-gcm';
const KEY = crypto.scryptSync(process.env.MASTER_KEY, 'salt', 32);

export function decrypt(encryptedData) {
    const [ivHex, authTagHex, encrypted] = encryptedData.split(':');
    const decipher = crypto.createDecipheriv(ALGORITHM, KEY, Buffer.from(ivHex, 'hex'));
    decipher.setAuthTag(Buffer.from(authTagHex, 'hex'));
    let decrypted = decipher.update(encrypted, 'hex', 'utf8');
    decrypted += decipher.final('utf8');
    return decrypted;
}
