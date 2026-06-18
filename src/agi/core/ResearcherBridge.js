import { exec } from 'child_process';

export function runMarketResearch() {
    return new Promise((resolve) => {
        exec('python3 src/agi/researcher/market_scanner.py', (err, stdout) => {
            console.log(stdout);
            resolve();
        });
    });
}
