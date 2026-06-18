import fetch from 'node-fetch';
import { randomBytes } from 'crypto';
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const DOMAIN_STATE_DIR = resolve(PROJECT_ROOT, '.data');
const DOMAIN_STATE_PATH = resolve(DOMAIN_STATE_DIR, 'domain_state.json');

const DEFAULT_DOMAIN = process.env.PRIMARY_DOMAIN || 'emerald-saas.io';
const CLOUDFLARE_API = 'https://api.cloudflare.com/client/v4';
const SUBDOMAIN_TEMPLATES = [
  '{name}-app', '{name}-hub', '{name}-platform', '{name}-dash', 'go-{name}',
  '{name}-io', 'get-{name}', '{name}-pro', 'use-{name}', 'app-{name}',
];

export class DomainRotator {
  constructor() {
    this._enabled = false;
    this._provisionedDomains = [];
    this._cloudflareToken = process.env.CLOUDFLARE_API_TOKEN || '';
    this._cloudflareZoneId = process.env.CLOUDFLARE_ZONE_ID || '';
    this._accountId = process.env.CLOUDFLARE_ACCOUNT_ID || '';
    this._nextSubIndex = 0;
    this._domainCounter = 0;
  }

  activate() {
    this._enabled = true;
    if (!existsSync(DOMAIN_STATE_DIR)) mkdirSync(DOMAIN_STATE_DIR, { recursive: true, mode: 0o700 });
    this._loadState();
    if (this._cloudflareToken && this._cloudflareZoneId) {
      logger.info('domain-rotator: active — Cloudflare subdomain provisioning ready');
    } else {
      logger.info('domain-rotator: active — simulated subdomain generation (set CLOUDFLARE_API_TOKEN + CLOUDFLARE_ZONE_ID for live provisioning)');
    }
    return { active: true, liveProvisioning: !!(this._cloudflareToken && this._cloudflareZoneId) };
  }

  deactivate() {
    this._enabled = false;
    this._saveState();
    logger.info('domain-rotator: deactivated');
  }

  async provision(productName, category) {
    if (!this._enabled) return { error: 'not_active' };
    const slug = this._slugify(productName);
    const subdomain = this._generateSubdomain(slug, category);
    const fullDomain = `${subdomain}.${DEFAULT_DOMAIN}`;

    if (this._cloudflareToken && this._cloudflareZoneId) {
      const cfResult = await this._createCloudflareRecord(subdomain, fullDomain);
      if (cfResult.success) {
        this._provisionedDomains.push({
          productName, subdomain, fullDomain, category,
          provisionedAt: Date.now(),
          cfRecordId: cfResult.recordId,
          live: true,
        });
        this._domainCounter++;
        this._saveState();
        logger.info(`domain-rotator: provisioned ${fullDomain} for "${productName}" via Cloudflare`);
        return { domain: fullDomain, subdomain, live: true, cfRecordId: cfResult.recordId };
      }
      logger.warn(`domain-rotator: Cloudflare provisioning failed — falling back to simulated domain`);
    }

    this._provisionedDomains.push({
      productName, subdomain, fullDomain, category,
      provisionedAt: Date.now(),
      live: false,
    });
    this._domainCounter++;
    this._saveState();
    logger.info(`domain-rotator: generated ${fullDomain} for "${productName}" (simulated)`);
    return { domain: fullDomain, subdomain, live: false };
  }

  async deprovision(domain) {
    const record = this._provisionedDomains.find(d => d.fullDomain === domain || d.subdomain === domain);
    if (!record) return { error: 'domain_not_found' };
    if (record.live && record.cfRecordId && this._cloudflareToken && this._cloudflareZoneId) {
      try {
        const res = await fetch(
          `${CLOUDFLARE_API}/zones/${this._cloudflareZoneId}/dns_records/${record.cfRecordId}`,
          { method: 'DELETE', headers: { 'Authorization': `Bearer ${this._cloudflareToken}`, 'Content-Type': 'application/json' } }
        );
        const data = await res.json();
        if (!data.success) {
          logger.warn(`domain-rotator: deprovision DNS failed for ${domain}`);
        }
      } catch (e) {
        logger.warn(`domain-rotator: deprovision failed — ${e.message}`);
      }
    }
    this._provisionedDomains = this._provisionedDomains.filter(d => d.fullDomain !== domain && d.subdomain !== domain);
    this._saveState();
    logger.info(`domain-rotator: deprovisioned ${domain}`);
    return { success: true };
  }

  getDomain(productName) {
    const record = this._provisionedDomains.find(d =>
      d.productName.toLowerCase() === productName.toLowerCase()
    );
    return record ? record.fullDomain : null;
  }

  getDomainRotatorStatus() {
    return {
      enabled: this._enabled,
      domainsProvisioned: this._provisionedDomains.length,
      liveRecords: this._provisionedDomains.filter(d => d.live).length,
      cloudflareConfigured: !!(this._cloudflareToken && this._cloudflareZoneId),
      primaryDomain: DEFAULT_DOMAIN,
    };
  }

  _generateSubdomain(slug, category) {
    const template = SUBDOMAIN_TEMPLATES[this._nextSubIndex % SUBDOMAIN_TEMPLATES.length];
    this._nextSubIndex++;
    let sub = template.replace('{name}', slug);
    const catMap = {
      productivity: 'prod', automation: 'auto', analytics: 'alytics',
      marketing: 'mktg', devtools: 'dev', finance: 'fin',
      ecommerce: 'shop', communication: 'comm', crm: 'crm', seo: 'seo',
    };
    const catSlug = catMap[(category || '').toLowerCase()] || 'saas';
    const suffix = randomBytes(2).toString('hex').slice(0, 4);
    sub = `${sub}-${catSlug}-${suffix}`;
    return sub.replace(/[^a-zA-Z0-9-]/g, '').toLowerCase().slice(0, 40);
  }

  async _createCloudflareRecord(subdomain, fullDomain) {
    try {
      const proxyTarget = process.env.CLOUDFLARE_PROXY_TARGET || `https://${fullDomain}`;
      const res = await fetch(
        `${CLOUDFLARE_API}/zones/${this._cloudflareZoneId}/dns_records`,
        {
          method: 'POST',
          headers: { 'Authorization': `Bearer ${this._cloudflareToken}`, 'Content-Type': 'application/json' },
          body: JSON.stringify({
            type: 'CNAME',
            name: subdomain,
            content: proxyTarget.replace(/https?:\/\//, ''),
            ttl: 120,
            proxied: true,
          }),
        }
      );
      const data = await res.json();
      if (data.success && data.result?.id) {
        return { success: true, recordId: data.result.id };
      }
      logger.warn(`domain-rotator: Cloudflare API response — ${JSON.stringify(data.errors || data)}`);
      return { success: false, errors: data.errors };
    } catch (e) {
      logger.warn(`domain-rotator: Cloudflare API call failed — ${e.message}`);
      return { success: false, error: e.message };
    }
  }

  _loadState() {
    try {
      if (!existsSync(DOMAIN_STATE_PATH)) return;
      const raw = readFileSync(DOMAIN_STATE_PATH, 'utf-8');
      const data = JSON.parse(raw);
      this._provisionedDomains = data.domains || [];
      this._domainCounter = data.counter || 0;
      this._nextSubIndex = data.nextIndex || 0;
    } catch {}
  }

  _saveState() {
    try {
      writeFileSync(DOMAIN_STATE_PATH, JSON.stringify({
        domains: this._provisionedDomains.slice(-200),
        counter: this._domainCounter,
        nextIndex: this._nextSubIndex % SUBDOMAIN_TEMPLATES.length,
      }, null, 2), { mode: 0o600 });
    } catch {}
  }

  _slugify(text) {
    return String(text || '').toLowerCase().replace(/[^\w\s-]/g, '').replace(/[\s_]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 20) || 'app';
  }
}

const domainRotator = new DomainRotator();
export default domainRotator;
