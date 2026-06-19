import { execSync } from 'child_process';
import fs from 'fs';

const REMOTES = {
  github: 'origin',
  huggingface: 'huggingface',
};

export async function syncToCloud() {
  console.log('[DEPLOY] Syncing to all remotes...');
  const results = { github: null, huggingface: null };

  for (const [name, remote] of Object.entries(REMOTES)) {
    try {
      execSync('git add .', { stdio: 'pipe' });
      execSync(`git commit -m "Auto-evolution: ${new Date().toISOString().slice(0, 10)}" --allow-empty`, { stdio: 'pipe' });
      execSync(`git push ${remote} main`, { stdio: 'pipe', timeout: 60000 });
      results[name] = 'ok';
      console.log(`[DEPLOY] ${name} → pushed`);
    } catch (err) {
      const msg = err.stderr?.toString() || err.message;
      if (msg.includes('hex auth token')) {
        console.warn(`[DEPLOY] ${name} → invalid token, skipping`);
        results[name] = 'auth_error';
      } else if (msg.includes('Could not read from remote')) {
        console.warn(`[DEPLOY] ${name} → remote not configured, skipping`);
        results[name] = 'no_remote';
      } else {
        console.warn(`[DEPLOY] ${name} → ${msg.slice(0, 100)}`);
        results[name] = 'error';
      }
    }
  }
  return results;
}

export async function autoDeploy() {
  const results = await syncToCloud();
  const ok = Object.values(results).filter(r => r === 'ok').length;
  console.log(`[DEPLOY] ${ok}/${Object.keys(REMOTES).length} remotes synced`);
  return results;
}
