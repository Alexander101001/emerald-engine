import { exec } from 'child_process';
import util from 'util';

const execPromise = util.promisify(exec);

export const syncToCloud = () => {
    return new Promise((resolve, reject) => {
        const commands = [
            'git add .',
            'git commit -m "Auto-Evolution: New strategies and system update"',
            'git push origin main'
        ].join(' && ');
        exec(commands, (error, stdout, stderr) => {
            if (error) {
                console.error("Sync Failed:", stderr);
                return reject(error);
            }
            console.log("Sync Success:", stdout);
            resolve(stdout);
        });
    });
};

export async function autoDeploy() {
    try {
        console.log("Deploying to Hugging Face Spaces...");
        await execPromise('git add .');
        await execPromise('git commit -m "Auto-evolution: New strategies added"');
        await execPromise('git push huggingface main');
        console.log("Deployment successful.");
    } catch (error) {
        console.error("Deployment failed:", error);
    }
}
