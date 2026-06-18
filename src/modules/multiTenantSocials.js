import { randomBytes, createHash } from 'crypto';
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const TENANT_STATE_DIR = resolve(PROJECT_ROOT, '.tenants');

const SESSION_POOLS = [
  { name: 'youtube_0', keys: { api: process.env.YOUTUBE_API_KEY_0 || process.env.YOUTUBE_API_KEY || '', refresh: process.env.YOUTUBE_REFRESH_TOKEN_0 || process.env.YOUTUBE_REFRESH_TOKEN || '', client: process.env.YOUTUBE_CLIENT_ID_0 || process.env.YOUTUBE_CLIENT_ID || '' }, channels: [], cooldown: 0 },
  { name: 'youtube_1', keys: { api: process.env.YOUTUBE_API_KEY_1 || '', refresh: process.env.YOUTUBE_REFRESH_TOKEN_1 || '', client: process.env.YOUTUBE_CLIENT_ID_1 || '' }, channels: [], cooldown: 0 },
  { name: 'tiktok_0', keys: { access: process.env.TIKTOK_ACCESS_TOKEN_0 || process.env.TIKTOK_ACCESS_TOKEN || '', api: process.env.TIKTOK_API_KEY_0 || process.env.TIKTOK_API_KEY || '' }, channels: [], cooldown: 0 },
  { name: 'tiktok_1', keys: { access: process.env.TIKTOK_ACCESS_TOKEN_1 || '', api: process.env.TIKTOK_API_KEY_1 || '' }, channels: [], cooldown: 0 },
  { name: 'twitter_0', keys: { api: process.env.TWITTER_API_KEY || '', secret: process.env.TWITTER_API_SECRET || '', token: process.env.TWITTER_ACCESS_TOKEN || '' }, channels: [], cooldown: 0 },
  { name: 'reddit_0', keys: { client: process.env.REDDIT_CLIENT_ID || '', secret: process.env.REDDIT_CLIENT_SECRET || '', refresh: process.env.REDDIT_REFRESH_TOKEN || '' }, channels: [], cooldown: 0 },
  { name: 'instagram_0', keys: { access: process.env.INSTAGRAM_ACCESS_TOKEN || '', business: process.env.INSTAGRAM_BUSINESS_ID || '' }, channels: [], cooldown: 0 },
  { name: 'linkedin_0', keys: { access: process.env.LINKEDIN_ACCESS_TOKEN || '', person: process.env.LINKEDIN_PERSON_ID || '' }, channels: [], cooldown: 0 },
];

export class MultiTenantSocials {
  constructor() {
    this._sessions = JSON.parse(JSON.stringify(SESSION_POOLS));
    this._tenants = new Map();
    this._enabled = false;
    this._tenantCounter = 0;
    this._idleRotations = 0;
  }

  activate() {
    this._enabled = true;
    if (!existsSync(TENANT_STATE_DIR)) {
      mkdirSync(TENANT_STATE_DIR, { recursive: true, mode: 0o700 });
    }
    const valid = this._sessions.filter(s => this._hasKeys(s)).length;
    logger.info(`multi-tenant: active — ${valid}/${this._sessions.length} isolated sessions available`);
    return { active: true, sessions: valid, total: this._sessions.length };
  }

  deactivate() {
    this._enabled = false;
    this._tenants.clear();
    logger.info('multi-tenant: deactivated — all tenants cleared');
  }

  registerTenant(saasName, platform) {
    if (!this._enabled) return { error: 'not_active' };
    const tenantId = `tn_${this._tenantCounter++}_${saasName.replace(/[^a-zA-Z0-9]/g, '_').slice(0, 20)}`;
    const session = this._allocateSession(platform);
    if (!session) return { error: `no_session_available_for_${platform}` };
    const tenant = {
      id: tenantId,
      saasName,
      platform,
      session: session.name,
      keys: { ...session.keys },
      channels: [],
      allocatedAt: Date.now(),
      lastActivity: Date.now(),
      commentCount: 0,
      warmupLevel: 0,
    };
    this._tenants.set(tenantId, tenant);
    session.channels.push(tenantId);
    this._persistTenant(tenant);
    logger.info(`multi-tenant: tenant "${tenantId}" registered — session ${session.name}`);
    return tenant;
  }

  getSession(tenantId) {
    const tenant = this._tenants.get(tenantId);
    if (!tenant) return null;
    const session = this._sessions.find(s => s.name === tenant.session);
    tenant.lastActivity = Date.now();
    if (session && Date.now() < session.cooldown) {
      return this._rotateSession(tenant);
    }
    return session || this._rotateSession(tenant);
  }

  recordActivity(tenantId, type) {
    const tenant = this._tenants.get(tenantId);
    if (!tenant) return false;
    tenant.lastActivity = Date.now();
    if (type === 'comment') tenant.commentCount++;
    if (type === 'upload') {
      const session = this._sessions.find(s => s.name === tenant.session);
      if (session) session.cooldown = Date.now() + 300000;
    }
    this._persistTenant(tenant);
    return true;
  }

  getTenant(tenantId) {
    return this._tenants.get(tenantId) || null;
  }

  listTenantsByPlatform(platform) {
    return Array.from(this._tenants.values()).filter(t => t.platform === platform);
  }

  warmupTenant(tenantId) {
    const tenant = this._tenants.get(tenantId);
    if (!tenant) return null;
    if (tenant.warmupLevel < 5) {
      tenant.warmupLevel++;
      this._persistTenant(tenant);
    }
    return tenant.warmupLevel;
  }

  releaseTenant(tenantId) {
    const tenant = this._tenants.get(tenantId);
    if (!tenant) return false;
    const session = this._sessions.find(s => s.name === tenant.session);
    if (session) {
      session.channels = session.channels.filter(c => c !== tenantId);
    }
    this._tenants.delete(tenantId);
    logger.info(`multi-tenant: tenant "${tenantId}" released`);
    return true;
  }

  getMultiTenantStatus() {
    const sessions = this._sessions.map(s => ({
      name: s.name,
      channels: s.channels.length,
      onCooldown: Date.now() < s.cooldown,
      hasKeys: this._hasKeys(s),
    }));
    return {
      enabled: this._enabled,
      tenants: this._tenants.size,
      sessions,
      idleRotations: this._idleRotations,
    };
  }

  _allocateSession(platform) {
    const candidates = this._sessions.filter(s =>
      s.name.startsWith(platform) && this._hasKeys(s)
    );
    if (candidates.length === 0) return null;
    candidates.sort((a, b) => a.channels.length - b.channels.length);
    return candidates[0];
  }

  _rotateSession(tenant) {
    const candidates = this._sessions.filter(s =>
      s.name.startsWith(tenant.platform) && this._hasKeys(s) && s.name !== tenant.session
    );
    if (candidates.length === 0) return this._sessions.find(s => s.name === tenant.session);
    const session = candidates[Math.floor(Math.random() * candidates.length)];
    const oldSession = this._sessions.find(s => s.name === tenant.session);
    if (oldSession) {
      oldSession.channels = oldSession.channels.filter(c => c !== tenant.id);
    }
    tenant.session = session.name;
    tenant.keys = { ...session.keys };
    session.channels.push(tenant.id);
    this._idleRotations++;
    this._persistTenant(tenant);
    logger.info(`multi-tenant: tenant "${tenant.id}" rotated to session ${session.name}`);
    return session;
  }

  _hasKeys(session) {
    const vals = Object.values(session.keys).filter(v => v && v.length > 5);
    return vals.length > 0;
  }

  _persistTenant(tenant) {
    try {
      const path = resolve(TENANT_STATE_DIR, `${tenant.id}.json`);
      writeFileSync(path, JSON.stringify(tenant, null, 2), { mode: 0o600 });
    } catch {}
  }
}

const multiTenant = new MultiTenantSocials();
export default multiTenant;
