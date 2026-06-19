import 'dotenv/config';
import { loadVault } from '../security/apiKeyVault.js';
import axios from 'axios';

loadVault();

const OLLAMA_URL = 'http://localhost:11434/api/generate';

async function run() {
    while(true) {
        try {
            await axios.post(OLLAMA_URL, {
                model: 'qwen2.5:1.5b',
                prompt: 'execute',
                stream: false
            });
            console.log('Cycle OK');
        } catch(e) {
            console.error('Cycle Fail');
        }
        await new Promise(r => setTimeout(r, 1000));
    }
}

run();
