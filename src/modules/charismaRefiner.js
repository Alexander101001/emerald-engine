import logger from '../utils/logger.js';
import evolutionLog from './socialEvolutionLog.js';
import persona from './personaEngine.js';

const CHARISMA_DIMENSIONS = ['warmth', 'expertise', 'enthusiasm', 'humility', 'directness'];
const EMPATHY_DIMENSIONS = ['acknowledge', 'validate', 'offerSolution', 'personalTouch'];

class CharismaRefiner {
  constructor() {
    this._lastRefinedAt = null;
    this._refinementHistory = [];
  }

  refine() {
    logger.info('charisma: starting weekly refinement cycle...');

    const stats = evolutionLog.getStats();
    const optimalResponses = evolutionLog.getOptimalResponses();
    const patterns = evolutionLog.getOptimalPatterns();

    if (optimalResponses.length < 3) {
      logger.info(`charisma: insufficient data (${optimalResponses.length} optimal responses) — skipping refinement`);
      return { refined: false, reason: 'insufficient_data', optimalCount: optimalResponses.length };
    }

    const profile = persona.getCharismaProfile();
    const strategy = persona.getEmpathyStrategy();
    const updates = {};
    const strategyUpdates = {};

    for (const dim of CHARISMA_DIMENSIONS) {
      const dimInteractions = optimalResponses.filter(i => {
        const text = (i.responseText || '').toLowerCase();
        return text.includes(dim);
      });
      const adjustment = dimInteractions.length / Math.max(optimalResponses.length, 1);
      if (adjustment > 0.1) {
        const delta = (adjustment - 0.1) * 0.2;
        updates[dim] = Math.min(1, Math.max(0.3, (profile[dim] || 0.7) + delta));
      }
    }

    for (const dim of EMPATHY_DIMENSIONS) {
      const dimInteractions = optimalResponses.filter(i => {
        const text = (i.responseText || '').toLowerCase();
        return text.includes(dim) || text.includes('understand') || text.includes('appreciate') || text.includes('hear');
      });
      const adjustment = dimInteractions.length / Math.max(optimalResponses.length, 1);
      if (adjustment > 0.1) {
        const delta = (adjustment - 0.1) * 0.15;
        if (typeof strategy[dim] === 'number') {
          strategyUpdates[dim] = Math.min(1, Math.max(0.3, (strategy[dim] || 0.6) + delta));
        } else {
          strategyUpdates[dim] = adjustment > 0.2;
        }
      }
    }

    if (Object.keys(updates).length > 0) {
      persona.updateCharismaProfile(updates);
    }
    if (Object.keys(strategyUpdates).length > 0) {
      persona.updateEmpathyStrategy(strategyUpdates);
    }

    const refinedProfile = persona.getCharismaProfile();
    const refinedStrategy = persona.getEmpathyStrategy();

    const result = {
      refined: true,
      optimalCount: optimalResponses.length,
      profileChanges: updates,
      strategyChanges: strategyUpdates,
      newProfile: refinedProfile,
      newStrategy: refinedStrategy,
      timestamp: new Date().toISOString(),
    };

    this._refinementHistory.push(result);
    this._lastRefinedAt = new Date().toISOString();

    logger.info(`charisma: refinement complete — warmth=${refinedProfile.warmth.toFixed(2)} expertise=${refinedProfile.expertise.toFixed(2)} from ${optimalResponses.length} optimal patterns`);
    return result;
  }

  generateInsights() {
    const stats = evolutionLog.getStats();
    const profile = persona.getCharismaProfile();
    const patterns = evolutionLog.getOptimalPatterns();

    return {
      interactionCount: stats.totalInteractions,
      optimalRate: stats.optimalRate,
      charismaProfile: profile,
      topKeywords: patterns.keywords.slice(0, 20),
      successfulConversions: patterns.conversions.length,
      lastRefinedAt: this._lastRefinedAt,
    };
  }

  getRefinementHistory() {
    return this._refinementHistory;
  }
}

const refiner = new CharismaRefiner();
export default refiner;
