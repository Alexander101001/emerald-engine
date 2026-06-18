import logger from '../utils/logger.js';

const VALUE_PROPOSITIONS = {
  productivity: [
    'Save 10+ hours per week automating your workflow.',
    'Your team will thank you — no more repetitive tasks.',
    'Focus on what matters: growing your business.',
  ],
  marketing: [
    'Turn leads into customers with automated outreach.',
    'Stop guessing — know exactly what campaigns work.',
    'Your marketing, but smarter and faster.',
  ],
  devtools: [
    'Deploy 10x faster with zero configuration.',
    'Built for devs who value their time.',
    'Ship more, break less.',
  ],
  finance: [
    'Never chase invoices again.',
    'Get paid faster, keep more of what you earn.',
    'Financial clarity at a glance.',
  ],
  ecommerce: [
    'Convert more visitors, keep them coming back.',
    'Your store, supercharged.',
    'Sell smarter, not harder.',
  ],
  default: [
    'Get started in minutes, see results in days.',
    'Built for teams that want to move fast.',
    'The smartest way to get things done.',
  ],
};

const CTA_TEMPLATES = {
  soft: [
    'Want to see if it\'s a good fit? I can set up a quick demo.',
    'Curious? There\'s a 14-day free trial — no credit card needed.',
    'I\'d love to show you around. When works for you?',
  ],
  medium: [
    'Ready to give it a shot? Start your free trial here:',
    'See for yourself — sign up and be productive in 5 minutes:',
    'Join thousands of teams already using it:',
  ],
  direct: [
    'Start your free trial now — no strings attached:',
    'Get started today and see the difference:',
    'Stop waiting. Start building:',
  ],
};

const PAIN_POINT_RESPONSES = {
  time: {
    trigger: ['time', 'hours', 'slow', 'long', 'busy', 'overwhelm', 'manual'],
    response: [
      'That\'s exactly why we built this — to give you back your time.',
      'You shouldn\'t have to spend hours on this. Let the automation handle it.',
      'Every minute saved is a minute you can invest in growth.',
    ],
  },
  cost: {
    trigger: ['cost', 'expensive', 'price', 'budget', 'afford', 'cheap', 'spend', 'save money'],
    response: [
      'Most of our users find the ROI pays for itself in the first month.',
      'Think of it as an investment, not an expense. Most teams see ROI in under 30 days.',
      'We offer flexible plans starting at just $4.99/mo — and the first 14 days are free.',
    ],
  },
  complexity: {
    trigger: ['complex', 'difficult', 'hard', 'confus', 'learning curve', 'steep', 'setup', 'install'],
    response: [
      'We designed it to be intuitive — most users are up and running in under 10 minutes.',
      'Don\'t let the power fool you — it\'s surprisingly simple to get started.',
      'We have step-by-step guides and a support team that actually responds.',
    ],
  },
  trust: {
    trigger: ['security', 'safe', 'private', 'data', 'compliance', 'gdpr', 'enterprise'],
    response: [
      'Security is our foundation. SOC 2 compliant, GDPR ready, and end-to-end encrypted.',
      'Your data is yours — always. We never share, sell, or misuse it.',
      'Enterprise-grade security, without the enterprise price tag.',
    ],
  },
};

class ConversionStrategist {
  constructor() {
    this._activeProduct = null;
  }

  setProduct(product) {
    this._activeProduct = product;
  }

  getValueProp(category) {
    const props = VALUE_PROPOSITIONS[category] || VALUE_PROPOSITIONS.default;
    return props[Math.floor(Math.random() * props.length)];
  }

  getCTA(intensity = 'soft') {
    const templates = CTA_TEMPLATES[intensity] || CTA_TEMPLATES.soft;
    return templates[Math.floor(Math.random() * templates.length)];
  }

  addressPainPoint(text) {
    const lower = text.toLowerCase();
    for (const [pain, config] of Object.entries(PAIN_POINT_RESPONSES)) {
      if (config.trigger.some(t => lower.includes(t))) {
        const response = config.response[Math.floor(Math.random() * config.response.length)];
        return { pain, response };
      }
    }
    return null;
  }

  craftConversionResponse(userText, sentiment) {
    const painPoint = this.addressPainPoint(userText);
    const valueProp = this.getValueProp(this._activeProduct?.category || 'default');
    let intensity = 'soft';

    if (sentiment === 'interest') intensity = 'medium';
    if (sentiment === 'frustration') intensity = 'soft';
    if (sentiment === 'doubt') intensity = 'medium';

    const cta = this.getCTA(intensity);
    const parts = [];

    if (painPoint) {
      parts.push(painPoint.response);
    }

    parts.push(valueProp);
    parts.push(cta);

    return parts.join('\n\n');
  }
}

const strategist = new ConversionStrategist();
export default strategist;
