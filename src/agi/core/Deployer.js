import { exec } from 'child_process';

export async function deployToCloud(platform) {
    console.log(`[DEPLOY] Starting deployment to ${platform}...`);
    
    const cmd = platform === 'HF' 
        ? 'git push huggingface main' 
        : 'git push origin main';

    return new Promise((resolve, reject) => {
        exec(cmd, (error, stdout, stderr) => {
            if (error) reject(error);
            resolve(stdout);
        });
    });
}
