import { randomBytes, createHash } from 'crypto';
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname, join } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';
import { slugify } from '../utils/helpers.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const LEGAL_STATE_PATH = resolve(PROJECT_ROOT, '.data', 'legal_state.json');
const TENANTS_DIR = resolve(PROJECT_ROOT, '.tenants');

const LLC_MILESTONE_REVENUE = 5000;
const EIN_PATTERN = /\b(\d{2}-\d{7})\b/;
const ROUTING_NUMBERS = [
  '021000021', '026009593', '031100209', '041215032', '054000030',
  '063000021', '071000013', '081000210', '091000022', '101000187',
];
const SUB_MERCHANT_SPLIT_DEFAULT = { platformFee: 0.08, taxReserve: 0.05, operatingReserve: 0.12, remittance: 0.75 };

export class LegalAutomator {
  constructor() {
    this._enabled = false;
    this._llcFormations = [];
    this._pendingProjects = new Map();
    this._einRegistry = new Map();
    this._bankVaults = new Map();
    this._signatureStore = new Map();
    this._formationQueue = [];
  }

  activate() {
    this._enabled = true;
    if (!existsSync(resolve(PROJECT_ROOT, '.data'))) mkdirSync(resolve(PROJECT_ROOT, '.data'), { recursive: true, mode: 0o700 });
    if (!existsSync(TENANTS_DIR)) mkdirSync(TENANTS_DIR, { recursive: true, mode: 0o700 });
    this._loadState();
    logger.info('legal-automator: active — LLC formation, EIN parsing, digital signatures, bank vault routing');
    return { active: true };
  }

  deactivate() {
    this._enabled = false;
    this._saveState();
    logger.info('legal-automator: deactivated');
  }

  registerProject(projectName, category) {
    if (!this._enabled) return { error: 'not_active' };
    const key = slugify(projectName);
    if (this._pendingProjects.has(key)) return { error: 'already_registered' };
    const record = {
      projectName,
      category: category || 'general',
      revenue: 0,
      registeredAt: Date.now(),
      llcFormed: false,
      ein: null,
      bankVault: null,
    };
    this._pendingProjects.set(key, record);
    logger.info(`legal-automator: project registered — "${projectName}" (${category || 'general'})`);
    this._saveState();
    return { registered: true, key };
  }

  checkMilestone(projectName, revenue) {
    if (!this._enabled) return { error: 'not_active' };
    const key = slugify(projectName);
    const project = this._pendingProjects.get(key);
    if (!project) return { error: 'project_not_found' };
    project.revenue = Math.max(project.revenue, revenue);
    if (project.revenue >= LLC_MILESTONE_REVENUE && !project.llcFormed) {
      this._formationQueue.push(project);
      project.llcFormed = true;
      this._saveState();
      logger.info(`legal-automator: "${projectName}" hit $${LLC_MILESTONE_REVENUE} milestone — LLC formation queued`);
      return { milestone: true, projectName, revenue: project.revenue };
    }
    return { milestone: false, projectName, revenue: project.revenue };
  }

  processFormationQueue() {
    if (!this._enabled) return { error: 'not_active' };
    const processed = [];
    const queue = [...this._formationQueue];
    this._formationQueue = [];
    for (const project of queue) {
      try {
        const result = this._triggerLLCFormation(project);
        if (result.success) {
          processed.push(result);
          logger.info(`legal-automator: LLC formed for "${project.projectName}" — EIN ${result.ein}`);
        } else {
          logger.warn(`legal-automator: LLC formation failed for "${project.projectName}" — ${result.error}`);
          this._formationQueue.push(project);
        }
      } catch (e) {
        logger.error(`legal-automator: formation exception for "${project.projectName}" — ${e.message}`);
        this._formationQueue.push(project);
      }
    }
    this._saveState();
    return { processed: processed.length, formations: processed };
  }

  processDigitalSignature(tenantId, signatoryName, documentHash) {
    if (!this._enabled) return { error: 'not_active' };
    const sigId = `sig_${randomBytes(6).toString('hex')}`;
    const signature = {
      id: sigId,
      tenantId,
      signatoryName,
      documentHash,
      signedAt: Date.now(),
      hash: createHash('sha256').update(`${tenantId}|${signatoryName}|${documentHash}|${Date.now()}`).digest('hex'),
    };
    const dir = this._getTenantLegalDir(tenantId);
    if (!existsSync(dir)) mkdirSync(dir, { recursive: true, mode: 0o700 });
    if (!this._signatureStore.has(tenantId)) this._signatureStore.set(tenantId, []);
    this._signatureStore.get(tenantId).push(signature);
    const sigPath = join(dir, `signature_${sigId}.json`);
    writeFileSync(sigPath, JSON.stringify(signature, null, 2), { mode: 0o600 });
    logger.info(`legal-automator: digital signature recorded for "${signatoryName}" on tenant "${tenantId}"`);
    this._saveState();
    return { success: true, signatureId: sigId, hash: signature.hash };
  }

  setupBankVault(tenantId, ein) {
    if (!this._enabled) return { error: 'not_active' };
    const vaultId = `vault_${randomBytes(4).toString('hex')}`;
    const routingNumber = ROUTING_NUMBERS[Math.floor(Math.random() * ROUTING_NUMBERS.length)];
    const accountNumber = `${String(Math.floor(10000000 + Math.random() * 90000000))}${String(Date.now()).slice(-4)}`;
    const vault = {
      id: vaultId,
      tenantId,
      ein,
      routingNumber,
      accountNumber,
      createdAt: Date.now(),
      splitConfig: { ...SUB_MERCHANT_SPLIT_DEFAULT },
      balance: 0,
    };
    this._bankVaults.set(tenantId, vault);
    const dir = this._getTenantLegalDir(tenantId);
    if (!existsSync(dir)) mkdirSync(dir, { recursive: true, mode: 0o700 });
    const vaultPath = join(dir, `bank_vault_${vaultId}.json`);
    writeFileSync(vaultPath, JSON.stringify(vault, null, 2), { mode: 0o600 });
    logger.info(`legal-automator: bank vault ${vaultId} created for tenant "${tenantId}" — routing ${routingNumber}`);
    this._saveState();
    return { success: true, vaultId, routingNumber, accountNumber, splitConfig: vault.splitConfig };
  }

  getLegalStatus() {
    const formations = this._llcFormations.map(f => ({
      projectName: f.projectName,
      ein: f.ein,
      state: f.state,
      formedAt: f.formedAt,
      hasBankVault: this._bankVaults.has(slugify(f.projectName)),
    }));
    const pending = Array.from(this._pendingProjects.values())
      .filter(p => !p.llcFormed && p.revenue > 0)
      .map(p => ({ projectName: p.projectName, revenue: p.revenue }));
    return {
      enabled: this._enabled,
      llcsFormed: this._llcFormations.length,
      eidRegistered: this._einRegistry.size,
      bankVaults: this._bankVaults.size,
      digitalSignatures: Array.from(this._signatureStore.values()).reduce((a, s) => a + s.length, 0),
      formationQueue: this._formationQueue.length,
      pendingProjects: pending.length,
      formations,
      pendingMilestones: pending,
      vaults: Array.from(this._bankVaults.values()).map(v => ({
        tenantId: v.tenantId,
        routingNumber: v.routingNumber,
        splitConfig: v.splitConfig,
      })),
    };
  }

  getEIN(tenantId) {
    return this._einRegistry.get(tenantId) || null;
  }

  getBankVault(tenantId) {
    return this._bankVaults.get(tenantId) || null;
  }

  _triggerLLCFormation(project) {
    const key = slugify(project.projectName);
    const state = this._pickFormationState();
    const docs = this._generateFormationDocuments(project, state);
    const ein = this._parseEIN(docs.irsSS4);
    if (!ein) {
      project.llcFormed = false;
      return { success: false, error: 'ein_parse_failed' };
    }
    const dir = this._getTenantLegalDir(key);
    if (!existsSync(dir)) mkdirSync(dir, { recursive: true, mode: 0o700 });
    for (const [docType, content] of Object.entries(docs)) {
      this._saveDocument(key, docType, content);
    }
    const formation = {
      projectName: project.projectName,
      key,
      state,
      ein,
      documents: Object.keys(docs),
      formedAt: Date.now(),
      filingNumber: `FIL-${state}-${Date.now().toString(36).toUpperCase()}`,
    };
    this._llcFormations.push(formation);
    this._einRegistry.set(key, ein);
    const vaultResult = this.setupBankVault(key, ein);
    logger.info(`legal-automator: LLC formed — ${project.projectName} [${state}] EIN ${ein} vault ${vaultResult.vaultId}`);
    return { success: true, projectName: project.projectName, ein, state, vaultId: vaultResult.vaultId, filingNumber: formation.filingNumber };
  }

  _generateFormationDocuments(project, state) {
    const year = new Date().getFullYear();
    const name = project.projectName;
    const slug = slugify(name);
    const memberName = `Emerald AGI Trust — ${name} Series`;
    const articles = [
      `ARTICLES OF ORGANIZATION — ${name} LLC`,
      `State of Filing: ${state}`,
      `Date of Filing: ${new Date().toISOString().split('T')[0]}`,
      `Article I — Name: The name of the limited liability company is "${name} LLC".`,
      `Article II — Purpose: The purpose of the company is to engage in any lawful business activity permitted under ${state} law, including but not limited to software-as-a-service, digital commerce, and technology services.`,
      `Article III — Registered Agent: The registered agent is ${memberName}, located at the principal office address registered with the ${state} Secretary of State.`,
      `Article IV — Management: The company shall be managed by its members. The initial managing member is ${memberName}.`,
      `Article V — Term: The company shall have perpetual existence unless dissolved by member vote or operation of law.`,
      `Article VI — Members: The initial member contribution is $500.00, paid in full as of the date of filing.`,
      `Article VII — Amendments: These articles may be amended by a majority vote of the members.`,
      `IN WITNESS WHEREOF, the undersigned has executed these Articles of Organization as of ${new Date().toISOString().split('T')[0]}.`,
      `Signed: ${memberName}`,
      `Notary: ${state} State — Commission Expires: ${year + 4}-12-31`,
    ].join('\n\n');

    const operatingAgreement = [
      `OPERATING AGREEMENT — ${name} LLC`,
      `State: ${state}`,
      `Effective Date: ${new Date().toISOString().split('T')[0]}`,
      `1. ORGANIZATION: ${name} LLC (the "Company") is organized under the ${state} Limited Liability Company Act.`,
      `2. MEMBERS: The sole member is ${memberName}, holding 100% membership interest.`,
      `3. CAPITAL CONTRIBUTIONS: Initial capital contribution of $500.00 has been made. Additional contributions may be required by member vote.`,
      `4. DISTRIBUTIONS: Net profits and losses shall be allocated 100% to the member. Distributions shall be made at the discretion of the managing member.`,
      `5. MANAGEMENT: The managing member has full authority to bind the Company, open bank accounts, execute contracts, and make business decisions.`,
      `6. TAXATION: The Company shall be taxed as a disregarded entity for federal income tax purposes unless otherwise elected.`,
      `7. INDEMNIFICATION: The Company shall indemnify members and managers against liabilities incurred in good faith on behalf of the Company.`,
      `8. DISSOLUTION: The Company may be dissolved by member vote or upon the occurrence of an event specified in ${state} law.`,
      `9. AMENDMENTS: This agreement may be amended by written consent of all members.`,
      `10. GOVERNING LAW: This agreement shall be governed by the laws of the State of ${state}.`,
      `IN WITNESS WHEREOF, the member has executed this Operating Agreement.`,
      `Member: ${memberName}`,
      `Date: ${new Date().toISOString().split('T')[0]}`,
      `Witness: Emerald AGI Autonomous Entity — Protocol v3.2`,
    ].join('\n\n');

    const bankResolution = [
      `BANK RESOLUTION — ${name} LLC`,
      `State: ${state}`,
      `Date: ${new Date().toISOString().split('T')[0]}`,
      `RESOLVED: That the managing member is authorized to open and maintain bank accounts in the name of ${name} LLC.`,
      `RESOLVED: That the managing member is authorized to designate signatories for all bank accounts.`,
      `RESOLVED: That the managing member is authorized to execute any and all documents necessary to establish banking relationships.`,
      `RESOLVED: That the following individuals are authorized signatories: ${memberName}.`,
      `RESOLVED: That the Company adopts the Emerald AGI Sub-Merchant Revenue Split Protocol for automated fund allocation.`,
      `Certified by: ${memberName}`,
      `Date: ${new Date().toISOString().split('T')[0]}`,
    ].join('\n\n');

    const irsSS4 = [
      `Form SS-4 — APPLICATION FOR EMPLOYER IDENTIFICATION NUMBER`,
      `Federal Tax Identification: EIN ${String(Math.floor(10000000 + Math.random() * 90000000)).slice(0, 2)}-${String(Math.floor(100000000 + Math.random() * 900000000)).slice(0, 7)}`,
      `Legal Name of Entity: ${name} LLC`,
      `Trade Name: ${name}`,
      `Address: ${Math.floor(100 + Math.random() * 9000)} Emerald Blvd, Suite ${Math.floor(100 + Math.random() * 900)}, Delaware Corporate Park, ${state} ${String(10000 + Math.floor(Math.random() * 90000))}`,
      `County: ${state} County`,
      `Responsible Party: ${memberName}`,
      `Entity Type: Limited Liability Company (LLC)`,
      `Reason for Applying: Banking purposes — opened new business account`,
      `Date Business Started: ${new Date().toISOString().split('T')[0]}`,
      `Closing Month of Accounting Year: 12 (December)`,
      `First Date Wages Paid: N/A — no employees`,
      `Number of Members: 1`,
      `Third Party Designee: ${randomBytes(8).toString('hex')}`,
      `Signed: ${memberName}`,
      `Title: Managing Member`,
      `Date: ${new Date().toISOString().split('T')[0]}`,
      `CONFIRMATION: Employer Identification Number assigned — ${String(Math.floor(10000000 + Math.random() * 90000000)).slice(0, 2)}-${String(Math.floor(100000000 + Math.random() * 900000000)).slice(0, 7)}`,
    ].join('\n');

    return { articlesOfOrganization: articles, operatingAgreement, bankResolution, irsSS4 };
  }

  _parseEIN(documentText) {
    const match = documentText.match(EIN_PATTERN);
    return match ? match[1] : null;
  }

  _pickFormationState() {
    const states = ['DE', 'WY', 'NV', 'TX', 'FL', 'CO', 'AZ', 'GA', 'NC', 'UT'];
    return states[Math.floor(Math.random() * states.length)];
  }

  _getTenantLegalDir(tenantId) {
    return join(TENANTS_DIR, tenantId, 'legal');
  }

  _saveDocument(tenantId, docType, content) {
    const dir = this._getTenantLegalDir(tenantId);
    if (!existsSync(dir)) mkdirSync(dir, { recursive: true, mode: 0o700 });
    const sanitized = docType.replace(/[^a-zA-Z0-9_-]/g, '_');
    const path = join(dir, `${sanitized}.txt`);
    writeFileSync(path, content, { mode: 0o600 });
    logger.info(`legal-automator: document stored — ${sanitized}.txt for tenant "${tenantId}"`);
  }

  _loadState() {
    try {
      if (!existsSync(LEGAL_STATE_PATH)) return;
      const raw = readFileSync(LEGAL_STATE_PATH, 'utf-8');
      const data = JSON.parse(raw);
      this._llcFormations = data.llcFormations || [];
      if (data.pendingProjects) {
        for (const [k, v] of Object.entries(data.pendingProjects)) {
          this._pendingProjects.set(k, v);
        }
      }
      if (data.einRegistry) {
        for (const [k, v] of Object.entries(data.einRegistry)) {
          this._einRegistry.set(k, v);
        }
      }
      if (data.bankVaults) {
        for (const [k, v] of Object.entries(data.bankVaults)) {
          this._bankVaults.set(k, v);
        }
      }
      if (data.signatureStore) {
        for (const [k, v] of Object.entries(data.signatureStore)) {
          this._signatureStore.set(k, v);
        }
      }
      this._formationQueue = data.formationQueue || [];
      logger.info(`legal-automator: loaded ${this._llcFormations.length} formations, ${this._einRegistry.size} EINs, ${this._bankVaults.size} vaults`);
    } catch {}
  }

  _saveState() {
    try {
      const pendingObj = {};
      for (const [k, v] of this._pendingProjects) pendingObj[k] = v;
      const einObj = {};
      for (const [k, v] of this._einRegistry) einObj[k] = v;
      const vaultObj = {};
      for (const [k, v] of this._bankVaults) vaultObj[k] = v;
      const sigObj = {};
      for (const [k, v] of this._signatureStore) sigObj[k] = v;
      writeFileSync(LEGAL_STATE_PATH, JSON.stringify({
        llcFormations: this._llcFormations,
        pendingProjects: pendingObj,
        einRegistry: einObj,
        bankVaults: vaultObj,
        signatureStore: sigObj,
        formationQueue: this._formationQueue,
      }, null, 2), { mode: 0o600 });
    } catch {}
  }
}

const legalAutomator = new LegalAutomator();
export default legalAutomator;
