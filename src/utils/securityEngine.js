import crypto from 'crypto';

const ALGORITHM = 'aes-256-gcm';
const IV_LENGTH = 12;
const SALT_LENGTH = 64;
const KEY_LENGTH = 32;
const ITERATIONS = 100000;

function deriveKey(masterPassword, salt) {
    return crypto.pbkdf2Sync(masterPassword, salt, ITERATIONS, KEY_LENGTH, 'sha512');
}

function encryptSecret(plainText, masterPassword) {
    const salt = crypto.randomBytes(SALT_LENGTH);
    const iv = crypto.randomBytes(IV_LENGTH);
    const key = deriveKey(masterPassword, salt);
    
    const cipher = crypto.createCipheriv(ALGORITHM, key, iv);
    let encrypted = cipher.update(plainText, 'utf8', 'hex');
    encrypted += cipher.final('hex');
    
    const authTag = cipher.getAuthTag().toString('hex');
    
    return {
        payload: encrypted,
        iv: iv.toString('hex'),
        salt: salt.toString('hex'),
        tag: authTag
    };
}

function decryptSecret(encryptedData, masterPassword) {
    const salt = Buffer.from(encryptedData.salt, 'hex');
    const iv = Buffer.from(encryptedData.iv, 'hex');
    const tag = Buffer.from(encryptedData.tag, 'hex');
    const key = deriveKey(masterPassword, salt);
    
    const decipher = crypto.createDecipheriv(ALGORITHM, key, iv);
    decipher.setAuthTag(tag);
    
    let decrypted = decipher.update(encryptedData.payload, 'hex', 'utf8');
    decrypted += decipher.final('utf8');
    return decrypted;
}

export { encryptSecret, decryptSecret };
