import fs from 'fs';
import path from 'path';

export class InfrastructureManager {
    constructor() {
        this.dirs = {
            logs: './logs',
            dist: './dist',
            secure: './secure-tokens.json'
        };
    }

    initialize() {
        Object.values(this.dirs).forEach(dir => {
            if (dir.includes('.') && fs.existsSync(dir)) {
                console.log(`[INFRA] Checking file: ${dir}`);
            } else if (!fs.existsSync(dir)) {
                fs.mkdirSync(dir);
                console.log(`[INFRA] Created directory: ${dir}`);
            }
        });
    }

    async loadSecureConfig() {
        if (!fs.existsSync(this.dirs.secure)) {
            throw new Error('SECURE_TOKENS_MISSING');
        }
        const data = JSON.parse(fs.readFileSync(this.dirs.secure, 'utf8'));
        return data;
    }
}
