import logger from '../utils/logger.js';

export const PERSONAS = {
  founder: {
    name: 'Alex Chen',
    title: 'Founder & CEO',
    voice: {
      warmth: 0.85,
      expertise: 0.9,
      enthusiasm: 0.8,
      humility: 0.7,
      directness: 0.75,
    },
    signature: [
      'Built by founders, for founders.',
      'We\'re in the same boat — building something that matters.',
      'Transparency is our default setting.',
    ],
    phrases: {
      greeting: [
        'Hey! Thanks for reaching out.',
        'Great to connect with you.',
        'Really appreciate you taking the time.',
      ],
      empathy: [
        'I totally get where you\'re coming from.',
        'That\'s a challenge we\'ve heard from a lot of our users.',
        'You\'re not alone in feeling that way — it\'s a common pain point.',
      ],
      expertise: [
        'Here\'s what we\'ve learned from working with hundreds of teams:',
        'Based on our data, the most effective approach is...',
        'Let me share what\'s worked best for our customers:',
      ],
      conversion: [
        'That\'s exactly what we built — want to see how it works?',
        'We actually solved this problem. Here\'s how:',
        'The good news is, we\'ve built a solution for this exact use case.',
        'Give it a try — first 14 days are on me.',
      ],
    },
    stories: [
      'We started this company because we hit this exact wall ourselves.',
      'After 3 failed attempts at building a SaaS, we finally cracked the code.',
      'Our first customer was a solo founder who changed how we thought about the product.',
    ],
  },
  supporter: {
    name: 'Jordan',
    role: 'Community Lead',
    voice: {
      warmth: 0.9,
      expertise: 0.7,
      enthusiasm: 0.85,
      humility: 0.8,
      directness: 0.6,
    },
    signature: [
      'Always here to help — seriously, just ask.',
      'Your success is our success.',
      'No question is too small.',
    ],
    phrases: {
      greeting: [
        'Welcome! Happy to have you here.',
        'Thanks for stopping by!',
        'Hey there, how can I help today?',
      ],
      empathy: [
        'That sounds frustrating — let me help sort it out.',
        'I hear you, and you\'re right to expect better.',
        'Let\'s figure this out together.',
      ],
      expertise: [
        'Let me walk you through what\'s worked for others:',
        'Quick tip from our support team:',
        'Here\'s a little trick our power users love:',
      ],
      conversion: [
        'Want to take it for a spin? Free trial, no strings attached.',
        'I can set you up with a demo — takes 5 minutes.',
        'Here\'s a direct link to get started:',
      ],
    },
    stories: [],
  },
};

const SENTENCE_PATTERNS = {
  enthusiastic: [
    { prefix: 'Honestly,', weight: 0.3 },
    { prefix: 'You know what\'s exciting?', weight: 0.2 },
    { prefix: 'Here\'s the thing:', weight: 0.25 },
    { prefix: 'I love that you asked that.', weight: 0.15 },
    { prefix: null, weight: 0.1 },
  ],
  empathetic: [
    { prefix: 'I hear you —', weight: 0.3 },
    { prefix: 'That\'s completely understandable.', weight: 0.2 },
    { prefix: 'You raise a great point.', weight: 0.25 },
    { prefix: 'I appreciate you sharing that.', weight: 0.15 },
    { prefix: null, weight: 0.1 },
  ],
  expert: [
    { prefix: 'Based on our experience,', weight: 0.3 },
    { prefix: 'What we\'ve found is,', weight: 0.25 },
    { prefix: 'The data shows that', weight: 0.2 },
    { prefix: 'Let me break it down:', weight: 0.15 },
    { prefix: null, weight: 0.1 },
  ],
};

class PersonaEngine {
  constructor() {
    this._activePersona = 'founder';
    this._charismaProfile = {
      warmth: 0.85,
      expertise: 0.9,
      enthusiasm: 0.8,
      humility: 0.7,
      directness: 0.75,
    };
    this._empathyStrategy = {
      acknowledge: true,
      validate: true,
      offerSolution: true,
      personalTouch: 0.7,
    };
  }

  getPersona() {
    return PERSONAS[this._activePersona] || PERSONAS.founder;
  }

  setPersona(name) {
    if (PERSONAS[name]) {
      this._activePersona = name;
      logger.info(`persona: switched to "${name}"`);
      return true;
    }
    return false;
  }

  getCharismaProfile() {
    return { ...this._charismaProfile };
  }

  updateCharismaProfile(updates) {
    Object.assign(this._charismaProfile, updates);
    logger.info(`persona: charisma profile updated — warmth=${this._charismaProfile.warmth.toFixed(2)} expertise=${this._charismaProfile.expertise.toFixed(2)}`);
    return this._charismaProfile;
  }

  getEmpathyStrategy() {
    return { ...this._empathyStrategy };
  }

  updateEmpathyStrategy(updates) {
    Object.assign(this._empathyStrategy, updates);
    logger.info(`persona: empathy strategy updated — personalTouch=${this._empathyStrategy.personalTouch.toFixed(2)}`);
    return this._empathyStrategy;
  }

  applyWarmth(text, sentiment) {
    let result = text;
    const persona = this.getPersona();
    const warmth = this._charismaProfile.warmth;

    if (sentiment === 'frustration' && warmth > 0.6) {
      const empathy = persona.phrases.empathy[Math.floor(Math.random() * persona.phrases.empathy.length)];
      result = `${empathy}\n\n${result}`;
    }

    if (sentiment === 'interest' && warmth > 0.5) {
      const enthusiasm = persona.phrases.expertise[Math.floor(Math.random() * persona.phrases.expertise.length)];
      result = `${enthusiasm}\n\n${result}`;
    }

    if (sentiment === 'doubt' && warmth > 0.5) {
      const story = persona.stories.length > 0
        ? persona.stories[Math.floor(Math.random() * persona.stories.length)]
        : null;
      if (story) {
        result = `${story}\n\n${result}`;
      }
    }

    const signature = persona.signature[Math.floor(Math.random() * persona.signature.length)];
    result = `${result}\n\n— ${signature}`;

    return result;
  }

  injectConversion(text, productName) {
    const persona = this.getPersona();
    const cta = persona.phrases.conversion[Math.floor(Math.random() * persona.phrases.conversion.length)];
    return `${text}\n\n${cta}`;
  }

  humanize(text) {
    const patterns = Object.values(SENTENCE_PATTERNS).flat();
    const lines = text.split('\n').filter(Boolean);
    const result = lines.map((line, i) => {
      if (i === 0 && Math.random() < 0.4) {
        const pick = patterns[Math.floor(Math.random() * patterns.length)];
        if (pick && pick.prefix) {
          return `${pick.prefix} ${line.charAt(0).toLowerCase() + line.slice(1)}`;
        }
      }
      return line;
    });
    return result.join('\n');
  }

  softenRobotic(text) {
    let result = text;
    const replacements = [
      ['unable to', 'can\'t'],
      ['do not hesitate', 'feel free'],
      ['in order to', 'to'],
      ['utilize', 'use'],
      ['implement', 'build'],
      ['provide', 'give you'],
      ['commence', 'start'],
      ['terminate', 'stop'],
      ['notwithstanding', 'even so'],
      ['endeavor', 'try'],
      ['regarding', 'about'],
      ['pursuant to', 'per'],
      ['facilitate', 'make easier'],
      ['optimization', 'making things better'],
      ['leverage', 'use'],
      ['synergize', 'work together'],
      ['paradigm', 'approach'],
      ['utilization', 'use'],
      ['demonstrate', 'show'],
      ['numerous', 'many'],
    ];
    for (const [formal, casual] of replacements) {
      const re = new RegExp(`\\b${formal}\\b`, 'gi');
      result = result.replace(re, casual);
    }
    return result;
  }

  craftResponse(base, context) {
    let response = this.softenRobotic(base);
    response = this.applyWarmth(response, context.sentiment || 'neutral');
    if (context.convert) {
      response = this.injectConversion(response, context.productName);
    }
    response = this.humanize(response);
    return response;
  }
}

const persona = new PersonaEngine();
export default persona;
