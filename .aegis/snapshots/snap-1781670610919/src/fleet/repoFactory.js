import fetch from 'node-fetch';
import logger from '../utils/logger.js';
import { uid, slugify } from '../utils/helpers.js';
import {
  generateStripeTemplate,
  generateCheckoutAPI,
  generateSEOConfig,
  generatePrivacyPage,
  generateTermsPage,
} from '../modules/saasPipeline.js';

const GITHUB_API = 'https://api.github.com';

function getToken() {
  return process.env.GITHUB_PAT || '';
}

function getOwner() {
  return process.env.GITHUB_OWNER || '';
}

function headers() {
  const token = getToken();
  return {
    Authorization: `Bearer ${token}`,
    Accept: 'application/vnd.github.v3+json',
    'User-Agent': 'EmeraldFleet/1.0',
    'Content-Type': 'application/json',
  };
}

export async function checkRateLimit() {
  const res = await fetch(`${GITHUB_API}/rate_limit`, { headers: headers() });
  const data = await res.json();
  return {
    remaining: data.resources?.core?.remaining || 0,
    limit: data.resources?.core?.limit || 60,
    resetAt: new Date((data.resources?.core?.reset || 0) * 1000).toISOString(),
  };
}

export async function createRepo(product) {
  const token = getToken();
  const owner = getOwner();

  if (!token || !owner) {
    logger.warn('repoFactory: GITHUB_PAT or GITHUB_OWNER not set — simulating repo creation');
    return simulateCreate(product);
  }

  const rateLimit = await checkRateLimit();
  if (rateLimit.remaining < 5) {
    throw new Error(`GitHub rate limit low (${rateLimit.remaining}), resets at ${rateLimit.resetAt}`);
  }

  const repoName = slugify(`emerald-${product.productName}`).slice(0, 40);
  const description = product.tagline || `${product.productName} — micro-SaaS built with Emerald Fleet`;

  logger.info(`repoFactory: creating repository "${repoName}"`);

  const createRes = await fetch(`${GITHUB_API}/user/repos`, {
    method: 'POST',
    headers: headers(),
    body: JSON.stringify({
      name: repoName,
      description: description.slice(0, 120),
      homepage: `https://${repoName}.vercel.app`,
      private: false,
      auto_init: true,
      license_template: 'mit',
      gitignore_template: 'Node',
    }),
  });

  if (!createRes.ok) {
    const err = await createRes.text();
    if (createRes.status === 422 && err.includes('already exists')) {
      logger.warn(`repoFactory: ${repoName} already exists — skipping`);
      return { repoName, exists: true, url: `https://github.com/${owner}/${repoName}` };
    }
    throw new Error(`GitHub create repo failed: ${createRes.status} ${err.slice(0, 200)}`);
  }

  const repo = await createRes.json();
  logger.info(`repoFactory: created ${repo.full_name} (${repo.html_url})`);

  await populateRepo(repo, product);

  await createWorkflows(repo);

  return {
    repoName,
    url: repo.html_url,
    cloneUrl: repo.clone_url,
    fullName: repo.full_name,
    exists: false,
  };
}

async function populateRepo(repo, product) {
  const owner = getOwner();
  const files = generateProductFiles(product);

  for (const file of files) {
    try {
      await putFile(repo, file);
      logger.info(`repoFactory: wrote ${file.path}`);
    } catch (e) {
      logger.warn(`repoFactory: failed to write ${file.path} — ${e.message}`);
    }
  }
}

async function putFile(repo, file) {
  const res = await fetch(`${GITHUB_API}/repos/${repo.full_name}/contents/${file.path}`, {
    method: 'PUT',
    headers: headers(),
    body: JSON.stringify({
      message: `Emerald Fleet: add ${file.path}`,
      content: Buffer.from(file.content).toString('base64'),
      branch: 'main',
    }),
  });

  if (!res.ok && res.status !== 422) {
    const err = await res.text();
    throw new Error(`PUT ${file.path}: ${res.status} ${err.slice(0, 100)}`);
  }
  return res;
}

async function createWorkflows(repo) {
  const workflows = generateWorkflowFiles();
  for (const wf of workflows) {
    try {
      await putFile(repo, wf);
    } catch (e) {
      logger.warn(`repoFactory: workflow ${wf.path} skipped — ${e.message}`);
    }
  }
}

function generateProductFiles(product) {
  const name = product.productName;
  const slug = slugify(name);
  const year = new Date().getFullYear();
  const tagline = product.tagline || 'A micro-SaaS built with Emerald Fleet.';
  const category = product.category || 'productivity';

  const stripeFile = generateStripeTemplate(name);
  const checkoutAPI = generateCheckoutAPI();
  const seoCfg = generateSEOConfig(name, tagline);
  const privacyPage = generatePrivacyPage(name);
  const termsPage = generateTermsPage(name);

  return [
    {
      path: 'README.md',
      content: `# ${name}\n\n${product.tagline || 'A micro-SaaS built with Emerald Fleet.'}\n\n## Features\n${(product.keyFeatures || []).map(f => `- ${f}`).join('\n')}\n\n## Deploy\n[![Deploy to Vercel](https://vercel.com/button)](https://vercel.com/new)\n[![Deploy to Netlify](https://www.netlify.com/img/deploy/button.svg)](https://app.netlify.com/start)\n\n---\n*Built by [Emerald Fleet](https://github.com/${getOwner() || 'emerald'}/emerald)*`,
    },
    {
      path: 'package.json',
      content: JSON.stringify({
        name: slug, version: '1.0.0', type: 'module', private: true,
        scripts: { start: 'node src/index.js', dev: 'node --watch src/index.js', deploy: 'bash deploy.sh' },
        dependencies: { 'node-fetch': '^3.3.2' },
        engines: { node: '>=18.0.0' },
      }, null, 2),
    },
    {
      path: 'src/index.js',
      content: `import fetch from 'node-fetch';

async function main() {
  console.log('${name} — online');
  // ${product.tagline || 'Micro-SaaS generated by Emerald Fleet'}
}

main().catch(console.error);
`,
    },
    {
      path: 'public/index.html',
      content: `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>${name} — ${product.tagline || ''}</title>
  <meta name="description" content="${product.tagline || name}" />
  <script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js"></script>
</head>
<body>
  <header style="text-align:center;padding:4rem 2rem;background:linear-gradient(135deg,#667eea,#764ba2);color:#fff">
    <h1>${name}</h1>
    <p>${product.tagline || 'Built with Emerald'}</p>
  </header>
  <main style="max-width:800px;margin:0 auto;padding:2rem">
    <section>
      <h2>Features</h2>
      <ul>${(product.keyFeatures || []).map(function(f) { return '<li>' + f + '</li>'; }).join('')}</ul>
    </section>
    <section class="emerald-ads">
      <!-- AD_SLOT_0 -->
      <!-- AD_SLOT_1 -->
    </section>
    <section style="text-align:center;padding:2rem">
      <a href="/subscribe" style="display:inline-block;padding:1rem 2rem;background:#667eea;color:#fff;border-radius:6px;text-decoration:none">Get Started Free</a>
    </section>
  </main>
  <footer style="text-align:center;padding:2rem;color:#666">
    <p>&copy; ${year} ${name} — <a href="/privacy">Privacy</a></p>
  </footer>
  <script src="/js/ads.js"></script>
</body>
</html>`,
    },
    {
      path: 'public/js/ads.js',
      content: `(function(){function d(){var e=document.querySelectorAll('.adsbygoogle');e.forEach(function(a){try{(adsbygoogle=window.adsbygoogle||[]).push({})}catch(b){}})}window.addEventListener('load',d);setTimeout(d,3000);var f=document.createElement('div');f.style.cssText='height:1px;width:1px;position:absolute;left:-9999px';f.className='adsbygoogle';document.body.appendChild(f);setTimeout(function(){if(!f.offsetParent||f.offsetHeight===0){document.querySelectorAll('.emerald-ads').forEach(function(a){a.innerHTML='<div style=\"padding:1rem;background:#f5f5f5;border:1px dashed #ccc;border-radius:4px;text-align:center\"><p>Ad blocked — <a href=\"/subscribe\">go ad-free</a></p></div>'})}f.remove()},2000)})();`,
    },
    {
      path: 'vercel.json',
      content: JSON.stringify({ rewrites: [{ source: '/(.*)', destination: '/api/entry' }], functions: { 'api/*.js': { maxDuration: 10 } } }, null, 2),
    },
    {
      path: 'netlify.toml',
      content: `[build]\n  command = "echo static"\n  functions = "netlify/functions"\n  publish = "."\n\n[[redirects]]\n  from = "/*"\n  to = "/.netlify/functions/entry"\n  status = 200`,
    },
    {
      path: 'api/entry.js',
      content: `export default async function handler(req, res) {
  const { method } = req;
  if (method === 'GET' && req.url === '/api/health') {
    return res.status(200).json({ status: 'ok', product: '${name}' });
  }
  return res.status(404).json({ error: 'not_found' });
};`,
    },
    {
      path: 'deploy.sh',
      content: `#!/bin/sh\nset -e\necho "Deploying ${name}..."\nnpm ci --omit=dev\nnpx vercel --prod --yes || echo "Vercel deploy skipped"\nnpx netlify-cli deploy --prod --dir=. || echo "Netlify deploy skipped"`,
    },
    {
      path: '.env.example',
      content: `EMERALD_KEY=your-12-char-min-passphrase\nSTRIPE_SECRET_KEY=\nSTRIPE_PUBLISHABLE_KEY=\nADSENSE_CLIENT_ID=ca-pub-xxxxxxxxxxxx`,
    },
    {
      path: '.gitignore',
      content: `node_modules/\n.env\n.env.local\n*.log\n.vercel\n.netlify\n.tmp-emerald/`,
    },
    {
      path: 'src/security/crypto-config.js',
      content: `import { createCipheriv, createDecipheriv, randomBytes, createHash, timingSafeEqual } from 'crypto';
import { readFileSync, writeFileSync, existsSync, unlinkSync } from 'fs';
const ALGO='aes-256-gcm';const IV_LEN=16;const SALT='emerald-v1-salt';
export function deriveKey(p){const k=createHash('sha256').update(p+SALT).digest();return k}
export function encrypt(d,p){const k=deriveKey(p);const iv=randomBytes(IV_LEN);const c=createCipheriv(ALGO,k,iv);let e=c.update(d,'utf-8','hex');e+=c.final('hex');return{encrypted:e,iv:iv.toString('hex'),tag:c.getAuthTag().toString('hex')}}
export function decrypt(d,p){const k=deriveKey(p);const dc=createDecipheriv(ALGO,k,Buffer.from(d.iv,'hex'));dc.setAuthTag(Buffer.from(d.tag,'hex'));let de=dc.update(d.encrypted,'hex','utf-8');de+=dc.final('utf-8');return de}
export function secureWipe(f,passes=3){if(!existsSync(f))return;const s=readFileSync(f).length;for(let i=0;i<passes;i++){writeFileSync(f,randomBytes(s))}unlinkSync(f)}
export default{encrypt,decrypt,deriveKey,secureWipe}`,
    },
    {
      path: 'src/monetization/adsense.js',
      content: `export const AD_CLIENT=process.env.ADSENSE_CLIENT_ID||'ca-pub-xxxxxxxxxxxx';
export function createAdSlot(){const id='emerald-ad-'+Math.random().toString(36).slice(2,8);return{id,html:'<ins class=\"adsbygoogle\" style=\"display:block\" data-ad-client=\"'+AD_CLIENT+'\" data-ad-slot=\"'+id+'\" data-ad-format=\"auto\"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({})<\\/script>',fallback:'<div style=\"padding:1rem;background:#f5f5f5\"><p>Ad blocked — <a href=\"/subscribe\">upgrade</a></p></div>'}}
export default{createAdSlot}`,
    },
    stripeFile,
    checkoutAPI,
    seoCfg,
    privacyPage,
    termsPage,
  ];
}

function generateWorkflowFiles() {
  var yaml = 'name: Deploy\n' +
    'on:\n  push:\n    branches: [main]\n' +
    'jobs:\n  deploy:\n    runs-on: ubuntu-latest\n    steps:\n' +
    '      - uses: actions/checkout@v4\n' +
    '      - uses: actions/setup-node@v4\n        with:\n          node-version: 20\n          cache: npm\n' +
    '      - run: npm ci --omit=dev --ignore-scripts --no-audit\n' +
    '      - name: Deploy to Vercel\n' +
    "        run: npx vercel --token=${{ secrets.VERCEL_TOKEN }} --prod --yes || echo skipped\n";
  return [
    { path: '.github/workflows/deploy.yml', content: yaml },
  ];
}

function simulateCreate(product) {
  const repoName = `emerald-${slugify(product.productName)}`.slice(0, 40);
  logger.info(`repoFactory: SIMULATED repo "${repoName}" created`);
  const productFiles = generateProductFiles(product);
  const workflowFiles = generateWorkflowFiles();
  const allFiles = [...productFiles, ...workflowFiles];
  return {
    repoName,
    url: `https://github.com/${getOwner() || 'emerald'}/${repoName}`,
    cloneUrl: `https://github.com/${getOwner() || 'emerald'}/${repoName}.git`,
    fullName: `${getOwner() || 'emerald'}/${repoName}`,
    simulated: true,
    files: allFiles.map(f => f.path),
  };
}

export default { createRepo, checkRateLimit, generateProductFiles };
