import 'dotenv/config';
import { loadVault } from '../security/apiKeyVault.js';
import axios from 'axios';
import fs from 'fs';

const OLLAMA_URL = 'http://localhost:11434/api/generate';
const OLLAMA_MODEL = process.env.OLLAMA_MODEL || 'qwen2.5:1.5b';

async function diagnostic() {
    console.log('[DIAGNOSTIC] Starting system audit...\n');

    // 1. ENVIRONMENT AUDIT
    console.log('[1/4] Environment audit...');
    const errors = [];

    if (!fs.existsSync('.env')) {
        errors.push('MISSING_CONFIG_ERROR: .env file not found');
    }

    if (!process.env.SERPER_API_KEY) {
        errors.push('MISSING_CONFIG_ERROR: SERPER_API_KEY is not set');
    }

    if (!process.env.GITHUB_TOKEN) {
        errors.push('MISSING_CONFIG_ERROR: GITHUB_TOKEN is not set');
    }

    if (errors.length > 0) {
        console.error('[FAIL] Environment audit failed:');
        errors.forEach(e => console.error('  ', e));
        return errors;
    }
    console.log('  [PASS] All required env vars present');

    // 2. SERVICE VERIFICATION
    console.log('[2/4] Service verification (Ollama :11434)...');
    try {
        const res = await axios.get('http://localhost:11434', { timeout: 5000 });
        if (res.status === 200) {
            console.log('  [PASS] Ollama is reachable');
        } else {
            const msg = `OLLAMA_SERVICE_DOWN: Unexpected status ${res.status}`;
            console.error('  [FAIL]', msg);
            errors.push(msg);
        }
    } catch (e) {
        const msg = 'OLLAMA_SERVICE_DOWN: Please run \'ollama serve\'';
        console.error('  [FAIL]', msg);
        errors.push(msg);
    }

    // 3. DEPENDENCY VERIFICATION
    console.log('[3/4] Dependency verification...');
    try {
        await import('axios');
        console.log('  [PASS] axios is installed');
    } catch {
        const msg = "DEPENDENCY_ERROR: run 'npm install axios'";
        console.error('  [FAIL]', msg);
        errors.push(msg);
    }

    // 4. OPERATIONAL READY STATE
    console.log('[4/4] Final verdict...');
    if (errors.length > 0) {
        console.error('\n[BOOT HALTED] Diagnostic failed — see errors above');
        console.log('\n=== DIAGNOSTIC LOG ===');
        errors.forEach((e, i) => console.log(`  ${i + 1}. ${e}`));
        process.exit(1);
    } else {
        console.log('\n[PASS] All checks passed — entering main loop');
    }
}

async function main() {
    const diagErrors = await diagnostic();
    if (diagErrors && diagErrors.length > 0) return;

    loadVault();

    while (true) {
        try {
            await axios.post(OLLAMA_URL, {
                model: OLLAMA_MODEL,
                prompt: 'execute',
                stream: false
            });
            console.log('Cycle OK');
        } catch (e) {
            console.error('Cycle Fail');
        }
        await new Promise(r => setTimeout(r, 1000));
    }
}

main();
