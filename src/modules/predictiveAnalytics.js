import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const ANALYTICS_STATE_DIR = resolve(PROJECT_ROOT, '.data');
const ANALYTICS_STATE_PATH = resolve(ANALYTICS_STATE_DIR, 'predictive_analytics.json');

const REVENUE_THRESHOLD = 5000;
const FORECAST_WINDOW_DAYS = 30;
const DECAY_FACTOR = 0.85;

export class PredictiveAnalytics {
  constructor() {
    this._enabled = false;
    this._weeklyReports = [];
    this._snapshots = [];
    this._lastSundayReport = 0;
  }

  activate() {
    this._enabled = true;
    if (!existsSync(ANALYTICS_STATE_DIR)) mkdirSync(ANALYTICS_STATE_DIR, { recursive: true, mode: 0o700 });
    this._loadState();
    logger.info('predictive-analytics: active — revenue forecasting, traffic/sentiment/Stripe ingestion');
    return { active: true };
  }

  deactivate() {
    this._enabled = false;
    this._saveState();
    logger.info('predictive-analytics: deactivated');
  }

  ingestSnapshot(metrics) {
    if (!this._enabled) return;
    const snapshot = {
      timestamp: Date.now(),
      ...metrics,
    };
    this._snapshots.push(snapshot);
    if (this._snapshots.length > 1000) this._snapshots = this._snapshots.slice(-500);
    this._saveState();
  }

  generateForecast() {
    if (!this._enabled) return { error: 'not_active' };
    const recent = this._snapshots.slice(-50);
    if (recent.length < 3) {
      return this._emptyForecast('insufficient_data');
    }
    const projects = this._aggregateByProject(recent);
    const forecasts = projects.map(p => this._forecastProject(p));
    const totalProjected = forecasts.reduce((a, f) => a + f.projectedRevenue, 0);
    const exceeding = forecasts.filter(f => f.projectedRevenue >= REVENUE_THRESHOLD);
    return {
      generatedAt: Date.now(),
      totalProjects: forecasts.length,
      totalProjectedRevenue: totalProjected,
      projectsExceedingThreshold: exceeding.length,
      threshold: REVENUE_THRESHOLD,
      projects: forecasts,
      topProjects: forecasts
        .filter(f => f.projectedRevenue >= REVENUE_THRESHOLD)
        .sort((a, b) => b.projectedRevenue - a.projectedRevenue),
      trends: this._extractTrends(recent),
      confidence: this._calculateConfidence(recent.length),
    };
  }

  formatReport(forecast) {
    if (!forecast || forecast.error) {
      return 'Predictive Analytics: Insufficient data for forecast this week.';
    }
    const lines = [
      'Predictive Revenue Forecast',
      '',
      `Projects Analyzed: ${forecast.totalProjects}`,
      `Total Projected Revenue: $${forecast.totalProjectedRevenue.toFixed(2)}`,
      `Projects Exceeding $${REVENUE_THRESHOLD}: ${forecast.projectsExceedingThreshold}`,
      `Confidence: ${(forecast.confidence * 100).toFixed(1)}%`,
      '',
    ];
    if (forecast.topProjects.length > 0) {
      lines.push('High-Value Projects (over $5K):');
      for (const p of forecast.topProjects) {
        const rev = `$${p.projectedRevenue.toFixed(2)}`;
        lines.push(`  ${p.name} — ${rev} (${p.growth}% projected growth, ${p.confidence}% confidence)`);
      }
      lines.push('');
    }
    const allSorted = [...forecast.projects].sort((a, b) => b.projectedRevenue - a.projectedRevenue);
    lines.push('All Projects (ranked):');
    for (const p of allSorted.slice(0, 10)) {
      const rev = `$${p.projectedRevenue.toFixed(2)}`;
      const flag = p.projectedRevenue >= REVENUE_THRESHOLD ? ' [OVER $5K]' : '';
      lines.push(`  ${p.name} — ${rev}${flag}`);
    }
    if (forecast.trends.length > 0) {
      lines.push('');
      lines.push('Trending Insights:');
      for (const t of forecast.trends.slice(0, 3)) {
        lines.push(`  ${t.label}: ${t.value}`);
      }
    }
    return lines.join('\n');
  }

  checkSundayReportDue() {
    const now = new Date();
    if (now.getDay() !== 0) return false;
    const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    return this._lastSundayReport < todayStart;
  }

  markSundayReportSent() {
    this._lastSundayReport = Date.now();
    this._saveState();
  }

  getAnalyticsStatus() {
    return {
      enabled: this._enabled,
      snapshotsCollected: this._snapshots.length,
      weeklyReportsGenerated: this._weeklyReports.length,
      lastSundayReport: this._lastSundayReport ? new Date(this._lastSundayReport).toISOString() : null,
    };
  }

  _emptyForecast(reason) {
    return {
      generatedAt: Date.now(),
      error: reason,
      totalProjects: 0,
      totalProjectedRevenue: 0,
      projectsExceedingThreshold: 0,
      threshold: REVENUE_THRESHOLD,
      projects: [],
      topProjects: [],
      trends: [],
      confidence: 0,
    };
  }

  _aggregateByProject(snapshots) {
    const projectMap = new Map();
    for (const s of snapshots) {
      if (!s.projectName) continue;
      if (!projectMap.has(s.projectName)) {
        projectMap.set(s.projectName, { name: s.projectName, category: s.category || 'general', data: [] });
      }
      projectMap.get(s.projectName).data.push(s);
    }
    return Array.from(projectMap.values());
  }

  _forecastProject(project) {
    const data = project.data;
    if (data.length < 2) {
      return {
        name: project.name,
        category: project.category,
        projectedRevenue: 0,
        currentRevenue: 0,
        growth: 0,
        confidence: 0,
        signals: ['insufficient_data'],
      };
    }
    const sorted = data.sort((a, b) => a.timestamp - b.timestamp);
    const latest = sorted[sorted.length - 1];
    const first = sorted[0];
    const timeSpanDays = (latest.timestamp - first.timestamp) / 86400000;
    const revenueValues = sorted.map(s => s.revenue || s.conversionValue || 0);
    const viewValues = sorted.map(s => s.views || 0);
    const sentimentValues = sorted.filter(s => s.sentimentScore !== undefined).map(s => s.sentimentScore);
    const currentRevenue = revenueValues[revenueValues.length - 1] || 0;
    const revenueGrowth = revenueValues.length > 2
      ? ((revenueValues[revenueValues.length - 1] - revenueValues[0]) / Math.max(1, revenueValues[0])) * 100
      : 0;
    const avgSentiment = sentimentValues.length > 0
      ? sentimentValues.reduce((a, v) => a + v, 0) / sentimentValues.length
      : 0.5;
    const viewVelocity = viewValues.length > 2
      ? (viewValues[viewValues.length - 1] - viewValues[0]) / Math.max(1, timeSpanDays)
      : 0;
    const projectedDaily = currentRevenue * 0.15 + viewVelocity * 0.003 + (avgSentiment - 0.5) * currentRevenue * 0.1;
    const projectedRevenue = currentRevenue + projectedDaily * FORECAST_WINDOW_DAYS * DECAY_FACTOR;
    const dataQuality = Math.min(1, data.length / 20);
    const confidence = Math.round(dataQuality * 100);
    const signals = [];
    if (avgSentiment > 0.7) signals.push('strong_positive_sentiment');
    if (viewVelocity > 100) signals.push('high_view_velocity');
    if (revenueGrowth > 50) signals.push('rapid_revenue_growth');
    if (revenueGrowth < -20) signals.push('revenue_declining');
    if (currentRevenue > 1000) signals.push('established_revenue_base');
    return {
      name: project.name,
      category: project.category,
      projectedRevenue: Math.max(0, projectedRevenue),
      currentRevenue,
      growth: Math.round(revenueGrowth * 100) / 100,
      confidence,
      signals,
      avgSentiment: Math.round(avgSentiment * 100) / 100,
      viewVelocity: Math.round(viewVelocity * 100) / 100,
      dataPoints: data.length,
    };
  }

  _extractTrends(snapshots) {
    const trends = [];
    const totalViews = snapshots.reduce((a, s) => a + (s.views || 0), 0);
    const totalRevenue = snapshots.reduce((a, s) => a + (s.revenue || s.conversionValue || 0), 0);
    const totalConversions = snapshots.reduce((a, s) => a + (s.conversions || 0), 0);
    const avgSentiment = snapshots.filter(s => s.sentimentScore !== undefined).reduce((a, s) => a + s.sentimentScore, 0) / Math.max(1, snapshots.filter(s => s.sentimentScore !== undefined).length);
    if (totalViews > 0) trends.push({ label: 'Total Views', value: totalViews.toLocaleString() });
    if (totalRevenue > 0) trends.push({ label: 'Total Tracked Revenue', value: `$${totalRevenue.toFixed(2)}` });
    if (totalConversions > 0) trends.push({ label: 'Total Conversions', value: totalConversions.toString() });
    if (avgSentiment > 0) trends.push({ label: 'Avg Sentiment Score', value: (avgSentiment * 100).toFixed(1) + '%' });
    return trends;
  }

  _calculateConfidence(dataPoints) {
    if (dataPoints < 5) return 0.1;
    if (dataPoints < 20) return 0.3;
    if (dataPoints < 50) return 0.6;
    return Math.min(0.95, 0.6 + dataPoints * 0.005);
  }

  _loadState() {
    try {
      if (!existsSync(ANALYTICS_STATE_PATH)) return;
      const raw = readFileSync(ANALYTICS_STATE_PATH, 'utf-8');
      const data = JSON.parse(raw);
      this._snapshots = data.snapshots || [];
      this._weeklyReports = data.reports || [];
      this._lastSundayReport = data.lastSundayReport || 0;
    } catch {}
  }

  _saveState() {
    try {
      writeFileSync(ANALYTICS_STATE_PATH, JSON.stringify({
        snapshots: this._snapshots.slice(-500),
        reports: this._weeklyReports.slice(-52),
        lastSundayReport: this._lastSundayReport,
      }, null, 2), { mode: 0o600 });
    } catch {}
  }
}

const predictiveAnalytics = new PredictiveAnalytics();
export default predictiveAnalytics;
