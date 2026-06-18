import simpleGit from 'simple-git';
import fs from 'fs';
import path from 'path';

export class SelfDeployer {
  constructor() {
    this.git = simpleGit();
    this.repoUrl = null;
  }

  async init() {
    const token = process.env.GITHUB_TOKEN;
    if (!token) {
      console.warn('selfDeployer: GITHUB_TOKEN not set — deploy disabled');
      return false;
    }
    const origin = await this.git.getConfig('remote.origin.url');
    if (origin.value) {
      this.repoUrl = origin.value.replace('https://github.com/', `https://${token}@github.com/`);
    } else {
      console.warn('selfDeployer: no git remote origin configured');
      return false;
    }
    console.log('selfDeployer: ready to push');
    return true;
  }

  async pushChanges(message) {
    if (!this.repoUrl) {
      const ok = await this.init();
      if (!ok) return false;
    }
    try {
      await this.git.add('.');
      await this.git.commit(message || 'Emerald: autonomous self-evolution');
      await this.git.push('origin', 'main');
      console.log('selfDeployer: pushed to', this.repoUrl.split('@')[1] || 'origin');
      return true;
    } catch (e) {
      console.error('selfDeployer: push failed —', e.message);
      return false;
    }
  }

  async writeAndDeploy(filePath, content, message) {
    const fullPath = path.join(process.cwd(), filePath);
    const dir = path.dirname(fullPath);
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(fullPath, content, 'utf8');
    console.log(`selfDeployer: wrote ${filePath}`);
    return this.pushChanges(message || `Emerald: add ${filePath}`);
  }
}
