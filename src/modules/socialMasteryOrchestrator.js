import logger from '../utils/logger.js';
import SentimentAnalyzer from './sentimentAnalyzer.js';
import PersonaEngine, { PERSONAS } from './personaEngine.js';
import SocialEvolutionLog from './socialEvolutionLog.js';
import ConversionStrategist from './conversionStrategist.js';
import CharismaRefiner from './charismaRefiner.js';

export class SocialMasteryOrchestrator {
  constructor() {
    this._initialized = false;
    this._persona = PersonaEngine;
    this._evolution = SocialEvolutionLog;
    this._conversion = ConversionStrategist;
    this._refiner = CharismaRefiner;
    this._stats = {
      interactionsProcessed: 0,
      conversionsAttempted: 0,
      optimalTagged: 0,
      refinementsRun: 0,
    };
  }

  initialize() {
    this._evolution.load();
    this._persona.setPersona('founder');
    this._initialized = true;
    logger.info('social-mastery: orchestrator initialized with founder persona');
    return { status: 'ok', persona: 'founder', interactionsLoaded: this._evolution.getStats().totalInteractions };
  }

  process(message, context = {}) {
    if (!this._initialized) {
      return { error: 'not_initialized', text: 'Orchestrator not initialized.' };
    }

    const sentiment = SentimentAnalyzer.analyzeSentiment(message);
    const userIntent = SentimentAnalyzer.classifyUserIntent(message);

    const baseResponse = this._buildBaseResponse(message, sentiment, context);
    const responseText = this._persona.craftResponse(baseResponse, {
      sentiment: sentiment.primary,
      convert: sentiment.readyToConvert || userIntent === 'sales',
    });

    const interaction = this._evolution.recordInteraction({
      messageText: message.slice(0, 500),
      responseText: responseText.slice(0, 500),
      sentiment: sentiment.primary,
      platform: context.platform || 'telegram',
      userId: context.userId,
      userIntent,
    });

    this._stats.interactionsProcessed++;

    if (sentiment.readyToConvert || (userIntent === 'sales' && sentiment.confidence > 0.4)) {
      this._stats.conversionsAttempted++;
    }

    return {
      text: responseText,
      sentiment,
      interactionId: interaction.id,
      intent: userIntent,
      requiresEmpathy: sentiment.requiresEmpathy,
      needsReassurance: sentiment.needsReassurance,
      readyToConvert: sentiment.readyToConvert,
    };
  }

  tagOptimal(interactionId, conversionType) {
    const result = this._evolution.tagOptimal(interactionId, conversionType);
    if (result) {
      this._stats.optimalTagged++;
    }
    return result;
  }

  runRefinement() {
    const result = this._refiner.refine();
    if (result.refined) {
      this._stats.refinementsRun++;
    }
    return result;
  }

  getInsights() {
    const stats = this._evolution.getStats();
    const insights = this._refiner.generateInsights();
    return {
      ...insights,
      orchestratorStats: this._stats,
      persona: this._persona.getPersona().name,
      charismaProfile: this._persona.getCharismaProfile(),
      empathyStrategy: this._persona.getEmpathyStrategy(),
    };
  }

  getPersonaList() {
    return Object.keys(PERSONAS || { founder: 1, supporter: 1 });
  }

  setPersona(name) {
    return this._persona.setPersona(name);
  }
}

const orchestrator = new SocialMasteryOrchestrator();
export default orchestrator;
