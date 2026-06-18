import fetch from 'node-fetch';
import logger from '../utils/logger.js';

const VERCEL_API = 'https://api.vercel.com';
const NETLIFY_API = 'https://api.netlify.com/api/v1';

function vercelHeaders(token) {
  return {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
}

function netlifyHeaders(token) {
  return {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
}

export async function deployToVercel(token, repo) {
  if (!token || token.startsWith('[INSERT')) {
    logger.warn('deployer: VERCEL_TOKEN not configured — skipping Vercel deploy');
    return null;
  }

  const repoFull = repo.fullName || repo.full_name;
  if (!repoFull) {
    logger.warn('deployer: no repo fullName available — skipping Vercel deploy');
    return null;
  }

  logger.info(`deployer: deploying ${repoFull} to Vercel...`);

  const [owner, repoName] = repoFull.split('/');

  const body = {
    name: repoName,
    gitSource: {
      type: 'github',
      repoId: repoFull,
      ref: 'main',
    },
    projectSettings: {
      framework: null,
      buildCommand: 'npm ci --omit=dev && npm start',
      outputDirectory: null,
      devCommand: null,
      installCommand: 'npm ci --omit=dev',
    },
  };

  try {
    const res = await fetch(`${VERCEL_API}/v1/deployments`, {
      method: 'POST',
      headers: vercelHeaders(token),
      body: JSON.stringify(body),
    });

    const data = await res.json();

    if (!res.ok) {
      logger.warn(`deployer: Vercel deploy failed — ${res.status} ${data.error?.message || JSON.stringify(data).slice(0, 200)}`);
      return null;
    }

    const url = `https://${data.url || `${repoName}.vercel.app`}`;
    logger.info(`deployer: Vercel deploy succeeded — ${url}`);
    return { platform: 'vercel', url, deploymentId: data.id || data.uid, repo: repoFull };
  } catch (e) {
    logger.warn(`deployer: Vercel deploy error — ${e.message}`);
    return null;
  }
}

export async function deployToNetlify(token, repo) {
  if (!token || token.startsWith('[INSERT')) {
    logger.warn('deployer: NETLIFY_TOKEN not configured — skipping Netlify deploy');
    return null;
  }

  const repoFull = repo.fullName || repo.full_name;
  if (!repoFull) {
    logger.warn('deployer: no repo fullName available — skipping Netlify deploy');
    return null;
  }

  logger.info(`deployer: deploying ${repoFull} to Netlify...`);

  const body = {
    name: repoFull.split('/')[1],
    repo: {
      provider: 'github',
      repo: repoFull,
      private: false,
      branch: 'main',
    },
  };

  try {
    const res = await fetch(`${NETLIFY_API}/sites`, {
      method: 'POST',
      headers: netlifyHeaders(token),
      body: JSON.stringify(body),
    });

    const data = await res.json();

    if (!res.ok) {
      logger.warn(`deployer: Netlify deploy failed — ${res.status} ${data.message || JSON.stringify(data).slice(0, 200)}`);
      return null;
    }

    const url = data.ssl_url || data.url || `https://${data.name}.netlify.app`;
    logger.info(`deployer: Netlify deploy succeeded — ${url}`);
    return { platform: 'netlify', url, siteId: data.id, repo: repoFull };
  } catch (e) {
    logger.warn(`deployer: Netlify deploy error — ${e.message}`);
    return null;
  }
}

export async function deployAll(repo, vercelToken, netlifyToken) {
  const results = [];

  const vercelResult = await deployToVercel(vercelToken, repo);
  if (vercelResult) results.push(vercelResult);

  const netlifyResult = await deployToNetlify(netlifyToken, repo);
  if (netlifyResult) results.push(netlifyResult);

  return results;
}

export default { deployToVercel, deployToNetlify, deployAll };
