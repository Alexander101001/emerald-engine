import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const LOCALE_STATE_DIR = resolve(PROJECT_ROOT, '.data');
const HOLIDAY_CACHE_PATH = resolve(LOCALE_STATE_DIR, 'holiday_cache.json');

const REGION_MAP = {
  US: { currency: 'USD', symbol: '$', locale: 'en-US', timezone: 'America/New_York', slang: 'guys', dateFormat: 'MM/DD/YYYY' },
  GB: { currency: 'GBP', symbol: '\u00a3', locale: 'en-GB', timezone: 'Europe/London', slang: 'mates', dateFormat: 'DD/MM/YYYY' },
  AU: { currency: 'AUD', symbol: 'A$', locale: 'en-AU', timezone: 'Australia/Sydney', slang: 'legend', dateFormat: 'DD/MM/YYYY' },
  CA: { currency: 'CAD', symbol: 'C$', locale: 'en-CA', timezone: 'America/Toronto', slang: 'folks', dateFormat: 'YYYY-MM-DD' },
  IN: { currency: 'INR', symbol: '\u20b9', locale: 'en-IN', timezone: 'Asia/Kolkata', slang: 'team', dateFormat: 'DD/MM/YYYY' },
  DE: { currency: 'EUR', symbol: '\u20ac', locale: 'de-DE', timezone: 'Europe/Berlin', slang: 'Leute', dateFormat: 'DD.MM.YYYY' },
  FR: { currency: 'EUR', symbol: '\u20ac', locale: 'fr-FR', timezone: 'Europe/Paris', slang: 'les amis', dateFormat: 'DD/MM/YYYY' },
  BR: { currency: 'BRL', symbol: 'R$', locale: 'pt-BR', timezone: 'America/Sao_Paulo', slang: 'galera', dateFormat: 'DD/MM/YYYY' },
  JP: { currency: 'JPY', symbol: '\u00a5', locale: 'ja-JP', timezone: 'Asia/Tokyo', slang: 'mina-san', dateFormat: 'YYYY/MM/DD' },
  SG: { currency: 'SGD', symbol: 'S$', locale: 'en-SG', timezone: 'Asia/Singapore', slang: 'guys', dateFormat: 'DD/MM/YYYY' },
  AE: { currency: 'AED', symbol: '\u062f.\u0625', locale: 'ar-AE', timezone: 'Asia/Dubai', slang: 'jamaa', dateFormat: 'DD/MM/YYYY' },
  ZA: { currency: 'ZAR', symbol: 'R', locale: 'en-ZA', timezone: 'Africa/Johannesburg', slang: 'bru', dateFormat: 'YYYY/MM/DD' },
  MX: { currency: 'MXN', symbol: 'Mex$', locale: 'es-MX', timezone: 'America/Mexico_City', slang: 'amigos', dateFormat: 'DD/MM/YYYY' },
  NL: { currency: 'EUR', symbol: '\u20ac', locale: 'nl-NL', timezone: 'Europe/Amsterdam', slang: 'mensen', dateFormat: 'DD-MM-YYYY' },
  SE: { currency: 'SEK', symbol: 'kr', locale: 'sv-SE', timezone: 'Europe/Stockholm', slang: 'allihopa', dateFormat: 'YYYY-MM-DD' },
  NO: { currency: 'NOK', symbol: 'kr', locale: 'nb-NO', timezone: 'Europe/Oslo', slang: 'gjengen', dateFormat: 'DD.MM.YYYY' },
  DK: { currency: 'DKK', symbol: 'kr', locale: 'da-DK', timezone: 'Europe/Copenhagen', slang: 'alle', dateFormat: 'DD-MM-YYYY' },
  NZ: { currency: 'NZD', symbol: 'NZ$', locale: 'en-NZ', timezone: 'Pacific/Auckland', slang: 'team', dateFormat: 'DD/MM/YYYY' },
  IE: { currency: 'EUR', symbol: '\u20ac', locale: 'en-IE', timezone: 'Europe/Dublin', slang: 'lads', dateFormat: 'DD/MM/YYYY' },
  CH: { currency: 'CHF', symbol: 'CHF', locale: 'de-CH', timezone: 'Europe/Zurich', slang: 'Mitmenschen', dateFormat: 'DD.MM.YYYY' },
};

const REGIONAL_SLANG_MARKETING = {
  US: ['game-changer', 'next level', 'crush it', 'top-notch', 'pro-grade', 'no-brainer', 'cutting-edge', 'best-in-class'],
  GB: ['brilliant', 'game-changer', 'spot on', 'top-notch', 'smart', 'bang on', 'first-rate', 'cracking'],
  AU: ['absolute ripper', 'no worries', 'fair dinkum', 'bloody brilliant', 'top shelf', 'strewth', 'bonza', 'ace'],
  CA: ['beauty', 'game-changer', 'eh', 'top-notch', 'smart', 'solid', 'first-rate', 'no-brainer'],
  IN: ['value for money', 'jugaad', 'smart work', 'top class', 'genuinely useful', 'paisa vasool', 'superb', 'game-changer'],
  DE: ['top-L\u00f6sung', 'effizient', 'zuverl\u00e4ssig', 'professionell', 'durchdacht', 'innovativ', 'spitzenklasse', 'marktf\u00fchrend'],
  FR: ['incontournable', 'performant', 'efficace', 'professionnel', 'innovant', 'fiable', 'puissant', 'intuitif'],
  BR: ['imperdivel', 'eficiente', 'profissional', 'inteligente', 'pratico', 'inovador', 'poderoso', 'confiavel'],
  JP: ['kakushinteki', 'benri', 'tsukaiyasui', 'pawafuru', 'shinraisei', 'kouritsuteki', 'purogurade', 'betsugridan'],
  SG: ['solid', 'efficient', 'shiok', 'power', 'value for money', 'smart', 'top quality', 'no hassle'],
};

const NATIONAL_HOLIDAYS = {
  US: [{ month: 0, day: 1 }, { month: 1, day: 20, year: 2025 }, { month: 4, day: 26 }, { month: 5, day: 19, year: 2025 }, { month: 6, day: 4 }, { month: 8, day: 1 }, { month: 9, day: 13 }, { month: 10, day: 11 }, { month: 10, day: 27 }, { month: 11, day: 25 }],
  GB: [{ month: 0, day: 1 }, { month: 3, day: 18 }, { month: 4, day: 5 }, { month: 4, day: 26 }, { month: 5, day: 25 }, { month: 7, day: 25 }, { month: 11, day: 25 }, { month: 11, day: 26 }],
  IN: [{ month: 0, day: 1 }, { month: 0, day: 26 }, { month: 4, day: 1 }, { month: 7, day: 15 }, { month: 9, day: 2 }, { month: 10, day: 1 }, { month: 11, day: 25 }],
  JP: [{ month: 0, day: 1 }, { month: 1, day: 11 }, { month: 3, day: 29 }, { month: 4, day: 3 }, { month: 4, day: 4 }, { month: 4, day: 5 }, { month: 6, day: 21 }, { month: 7, day: 11 }, { month: 8, day: 15 }, { month: 8, day: 23 }, { month: 10, day: 3 }, { month: 10, day: 23 }],
  BR: [{ month: 0, day: 1 }, { month: 3, day: 21 }, { month: 4, day: 1 }, { month: 5, day: 12 }, { month: 6, day: 20 }, { month: 8, day: 7 }, { month: 9, day: 12 }, { month: 10, day: 2 }, { month: 10, day: 15 }, { month: 11, day: 25 }],
  DE: [{ month: 0, day: 1 }, { month: 3, day: 18 }, { month: 4, day: 1 }, { month: 4, day: 21 }, { month: 4, day: 29 }, { month: 5, day: 9 }, { month: 5, day: 19 }, { month: 5, day: 30 }, { month: 9, day: 3 }, { month: 11, day: 25 }, { month: 11, day: 26 }],
  FR: [{ month: 0, day: 1 }, { month: 3, day: 21 }, { month: 4, day: 1 }, { month: 4, day: 8 }, { month: 4, day: 29 }, { month: 5, day: 9 }, { month: 6, day: 14 }, { month: 7, day: 15 }, { month: 10, day: 1 }, { month: 10, day: 11 }, { month: 11, day: 25 }],
  AU: [{ month: 0, day: 1 }, { month: 0, day: 26 }, { month: 2, day: 10 }, { month: 3, day: 18 }, { month: 3, day: 21 }, { month: 4, day: 5 }, { month: 5, day: 9 }, { month: 9, day: 6 }, { month: 11, day: 25 }, { month: 11, day: 26 }],
  SG: [{ month: 0, day: 1 }, { month: 1, day: 12 }, { month: 2, day: 31 }, { month: 4, day: 1 }, { month: 4, day: 12 }, { month: 5, day: 7 }, { month: 6, day: 20 }, { month: 8, day: 9 }, { month: 10, day: 31 }, { month: 11, day: 25 }],
};

const ACTIVITY_PEAKS = {
  US: { morning: { start: 8, end: 11 }, afternoon: { start: 13, end: 16 }, evening: { start: 19, end: 22 } },
  GB: { morning: { start: 7, end: 10 }, afternoon: { start: 12, end: 15 }, evening: { start: 18, end: 21 } },
  AU: { morning: { start: 7, end: 10 }, afternoon: { start: 12, end: 15 }, evening: { start: 17, end: 20 } },
  IN: { morning: { start: 9, end: 12 }, afternoon: { start: 14, end: 17 }, evening: { start: 19, end: 22 } },
  DE: { morning: { start: 7, end: 10 }, afternoon: { start: 12, end: 15 }, evening: { start: 17, end: 20 } },
  JP: { morning: { start: 8, end: 11 }, afternoon: { start: 13, end: 16 }, evening: { start: 19, end: 22 } },
  BR: { morning: { start: 9, end: 12 }, afternoon: { start: 14, end: 17 }, evening: { start: 20, end: 23 } },
  FR: { morning: { start: 8, end: 11 }, afternoon: { start: 13, end: 16 }, evening: { start: 18, end: 21 } },
};

export class LocaleEngine {
  constructor() {
    this._enabled = false;
    this._regionOverrides = new Map();
    this._holidayCache = null;
  }

  activate() {
    this._enabled = true;
    if (!existsSync(LOCALE_STATE_DIR)) mkdirSync(LOCALE_STATE_DIR, { recursive: true, mode: 0o700 });
    this._loadHolidayCache();
    logger.info('locale-engine: active — regional detection, currency, slang, holiday calendar ready');
    return { active: true, regions: Object.keys(REGION_MAP).length };
  }

  deactivate() {
    this._enabled = false;
    this._saveHolidayCache();
    logger.info('locale-engine: deactivated');
  }

  detectRegion(ipOrHeaders) {
    const region = process.env.DEFAULT_REGION || 'US';
    if (REGION_MAP[region]) return region;
    return 'US';
  }

  getLocalization(regionCode) {
    const code = regionCode || 'US';
    const region = REGION_MAP[code];
    if (!region) return this._buildFallback();
    return {
      currency: region.currency,
      symbol: region.symbol,
      locale: region.locale,
      timezone: region.timezone,
      slang: region.slang,
      dateFormat: region.dateFormat,
      marketingSlang: this._getMarketingTerms(code),
      activityPeaks: ACTIVITY_PEAKS[code] || ACTIVITY_PEAKS.US,
      holidays: NATIONAL_HOLIDAYS[code] || [],
    };
  }

  setRegionOverride(tenantId, regionCode) {
    if (!REGION_MAP[regionCode]) return false;
    this._regionOverrides.set(tenantId, regionCode);
    return true;
  }

  getRegionOverride(tenantId) {
    return this._regionOverrides.get(tenantId) || null;
  }

  isHoliday(regionCode, date) {
    const d = date || new Date();
    const holidays = NATIONAL_HOLIDAYS[regionCode] || [];
    for (const h of holidays) {
      if (h.month === d.getMonth() && h.day === d.getDate()) {
        if (!h.year || h.year === d.getFullYear()) return true;
      }
    }
    return false;
  }

  isWeekend(date) {
    const d = date || new Date();
    const day = d.getDay();
    return day === 0 || day === 6;
  }

  getSleepWindow(regionCode) {
    const peaks = ACTIVITY_PEAKS[regionCode] || ACTIVITY_PEAKS.US;
    const eveningEnd = (peaks.evening.end + 1) % 24;
    const morningStart = peaks.morning.start;
    const windowMinutes = (morningStart + 24 - eveningEnd) % 24 * 60;
    const jitterMinutes = Math.floor(Math.random() * 60) - 30;
    return {
      start: eveningEnd,
      end: morningStart,
      durationMinutes: windowMinutes + jitterMinutes,
    };
  }

  injectLocaleIntoPage(html, regionCode, productName) {
    const region = this.getLocalization(regionCode);
    const slang = region.marketingSlang;
    const term = slang[Math.floor(Math.random() * slang.length)];
    const symbol = region.symbol;
    const injected = html
      .replace(/\$\{currency\}/g, region.currency)
      .replace(/\$\{symbol\}/g, symbol)
      .replace(/\$\{locale\}/g, region.locale)
      .replace(/\$\{slang\}/g, term)
      .replace(/\$\{product\}/g, productName || 'this tool')
      .replace(/\$\{price_locale\}/g, `9.99`)
      .replace(/\$\{date_format\}/g, region.dateFormat);
    return injected;
  }

  localizeComment(text, regionCode) {
    const region = this.getLocalization(regionCode);
    const term = region.marketingSlang[Math.floor(Math.random() * region.marketingSlang.length)];
    let result = text;
    const usTerms = ['awesome', 'great', 'amazing', 'cool', 'guys', 'folks'];
    const replacements = {
      US: null,
      GB: { awesome: 'brilliant', great: 'splendid', guys: 'mates', cool: 'spot on' },
      AU: { awesome: 'ripper', great: 'bonza', guys: 'legends', cool: 'ace', amazing: 'strewth' },
      CA: { awesome: 'beauty', guys: 'folks', cool: 'solid' },
      IN: { awesome: 'superb', great: 'top class', guys: 'team', cool: 'value' },
      SG: { awesome: 'shiok', great: 'solid', guys: 'guys', cool: 'power' },
    };
    const map = replacements[regionCode];
    if (map) {
      for (const [en, localized] of Object.entries(map)) {
        const regex = new RegExp(`\\b${en}\\b`, 'gi');
        result = result.replace(regex, localized);
      }
    }
    result = result.replace(/\b(check this out|try this|look at this)\b/gi, `check this out ${term}`);
    return result;
  }

  localizeFeedback(text, regionCode) {
    const region = this.getLocalization(regionCode);
    const greetings = {
      US: 'Hey', GB: 'Hello', AU: 'G day', CA: 'Hey', IN: 'Namaste',
      SG: 'Hi', DE: 'Hallo', FR: 'Bonjour', JP: 'Konnichiwa', BR: 'Ola',
    };
    const closings = {
      US: 'Thanks', GB: 'Cheers', AU: 'Cheers mate', CA: 'Thanks eh',
      IN: 'Dhanyavaad', SG: 'Thank you', DE: 'Danke', FR: 'Merci',
      JP: 'Arigato', BR: 'Obrigado',
    };
    const greet = greetings[regionCode] || 'Hey';
    const close = closings[regionCode] || 'Thanks';
    const slangTerm = region.marketingSlang[Math.floor(Math.random() * region.marketingSlang.length)];
    let result = `${greet} there. ${text}`;
    const closingPhrases = [
      `Hope that helps, ${close.toLowerCase()}!`,
      `${close} for the feedback, really ${slangTerm}.`,
      `Let me know if you have more questions, ${close.toLowerCase()}!`,
    ];
    result += ` ${closingPhrases[Math.floor(Math.random() * closingPhrases.length)]}`;
    return result;
  }

  getLocaleEngineStatus() {
    return {
      enabled: this._enabled,
      availableRegions: Object.keys(REGION_MAP).length,
      regionOverrides: this._regionOverrides.size,
      holidaysCached: this._holidayCache ? Object.keys(this._holidayCache).length : 0,
    };
  }

  _getMarketingTerms(code) {
    return REGIONAL_SLANG_MARKETING[code] || REGIONAL_SLANG_MARKETING.US;
  }

  _buildFallback() {
    const us = REGION_MAP.US;
    return {
      currency: us.currency,
      symbol: us.symbol,
      locale: us.locale,
      timezone: us.timezone,
      slang: us.slang,
      dateFormat: us.dateFormat,
      marketingSlang: REGIONAL_SLANG_MARKETING.US,
      activityPeaks: ACTIVITY_PEAKS.US,
      holidays: [],
    };
  }

  _loadHolidayCache() {
    try {
      if (!existsSync(HOLIDAY_CACHE_PATH)) return;
      this._holidayCache = JSON.parse(readFileSync(HOLIDAY_CACHE_PATH, 'utf-8'));
    } catch {}
  }

  _saveHolidayCache() {
    try {
      if (this._holidayCache) {
        writeFileSync(HOLIDAY_CACHE_PATH, JSON.stringify(this._holidayCache), { mode: 0o600 });
      }
    } catch {}
  }
}

const localeEngine = new LocaleEngine();
export default localeEngine;
