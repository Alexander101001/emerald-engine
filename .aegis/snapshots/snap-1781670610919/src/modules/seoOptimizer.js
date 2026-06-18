import logger from '../utils/logger.js';

const LONG_TAIL_KEYWORDS = {
  productivity: [
    'best productivity tools for remote teams',
    'how to increase team productivity',
    'productivity software for small business',
    'affordable project management solution',
    'team collaboration tools comparison',
    'workflow automation software guide',
    'task management app for startups',
  ],
  marketing: [
    'digital marketing tools for small business',
    'best email marketing platform 2026',
    'social media management for startups',
    'content marketing automation tools',
    'seo tools for small business owners',
    'lead generation software comparison',
    'marketing analytics platform review',
  ],
  devtools: [
    'best developer tools for startups',
    'api management platform comparison',
    'cloud deployment tools for developers',
    'ci cd pipeline tools 2026',
    'monitoring tools for microservices',
    'developer productivity tools review',
    'open source devops tools guide',
  ],
  finance: [
    'best accounting software for small business',
    'financial planning tools for startups',
    'invoice software for freelancers',
    'expense management app review',
    'budgeting software for entrepreneurs',
    'payment processing tools comparison',
    'business analytics dashboard tools',
  ],
  health: [
    'wellness app for busy professionals',
    'health tracking software 2026',
    'telehealth platform comparison',
    'fitness app for remote workers',
    'mental health tools for startups',
    'nutrition tracking app review',
    'healthcare management software',
  ],
  education: [
    'online learning platform comparison',
    'best lms for small business',
    'course creation tools for educators',
    'virtual classroom software review',
    'student management system 2026',
    'elearning tools for corporate training',
    'educational technology trends 2026',
  ],
  ecommerce: [
    'best ecommerce platform for small business',
    'shopify alternative for startups',
    'dropshipping tools 2026 guide',
    'ecommerce seo optimization tips',
    'shopping cart software comparison',
    'inventory management for online stores',
    'conversion rate optimization tools',
  ],
  design: [
    'ui ux design tools for startups',
    'prototyping software comparison 2026',
    'design collaboration tools for teams',
    'graphic design software for non-designers',
    'brand identity tools for entrepreneurs',
    'wireframe tools for web developers',
    'design system management platforms',
  ],
  analytics: [
    'business intelligence tools for startups',
    'data analytics platform comparison',
    'real time analytics dashboard tools',
    'customer analytics software review',
    'web analytics tools for marketers',
    'product analytics platforms 2026',
    'reporting tools for small business',
  ],
  automation: [
    'workflow automation tools 2026',
    'business process automation software',
    'robotic process automation for smb',
    'email automation tools comparison',
    'marketing automation platforms review',
    'automated reporting tools for teams',
    'low code automation platforms',
  ],
};

const YOUTUBE_KEYWORDS = {
  productivity: ['productivity tips', 'team management', 'workflow automation'],
  marketing: ['digital marketing strategy', 'social media tips', 'email marketing'],
  devtools: ['developer tools', 'coding tips', 'devops tutorial'],
  finance: ['business finance', 'accounting tips', 'financial planning'],
  health: ['wellness tips', 'healthy habits', 'mental health'],
  education: ['online learning', 'study tips', 'course creation'],
  ecommerce: ['ecommerce tips', 'online store', 'dropshipping'],
  design: ['design tips', 'ui ux tutorial', 'graphic design'],
  analytics: ['data analytics', 'business intelligence', 'reporting'],
  automation: ['automation tips', 'workflow', 'productivity hacks'],
};

export class SEOOptimizer {
  constructor() {
    this._keywordCache = new Map();
  }

  getLongTailKeywords(category) {
    const cat = (category || 'productivity').toLowerCase();
    const cached = this._keywordCache.get(cat);
    if (cached) return cached;

    const keywords = LONG_TAIL_KEYWORDS[cat] || LONG_TAIL_KEYWORDS.productivity;
    this._keywordCache.set(cat, keywords);
    return keywords;
  }

  getYouTubeTags(category) {
    const cat = (category || 'productivity').toLowerCase();
    return YOUTUBE_KEYWORDS[cat] || YOUTUBE_KEYWORDS.productivity;
  }

  generateSEOMeta(product, keywords) {
    const primary = keywords.slice(0, 3);
    const secondary = keywords.slice(3, 7);
    const title = `${product.productName}: ${primary[0] || 'Complete Guide'} (${new Date().getFullYear()})`;

    return {
      title: title.slice(0, 60),
      description: `Discover ${product.productName}, the leading ${product.category || 'SaaS'} solution. ${primary.join(', ')}. Comprehensive review, features, pricing, and alternatives for ${new Date().getFullYear()}.`.slice(0, 160),
      keywords: [...primary, ...secondary, product.productName, product.category || 'saas'].filter(Boolean).join(', '),
      ogTitle: title.slice(0, 60),
      ogDescription: `${product.productName} — ${product.tagline || 'Built with Emerald AGI'}`,
      twitterTitle: title.slice(0, 60),
      structuredData: {
        '@context': 'https://schema.org',
        '@type': 'SoftwareApplication',
        name: product.productName,
        applicationCategory: 'BusinessApplication',
        operatingSystem: 'Web',
        description: product.tagline || '',
        offers: {
          '@type': 'AggregateOffer',
          lowPrice: '4.99',
          highPrice: '29.99',
          priceCurrency: 'USD',
        },
        review: {
          '@type': 'Review',
          reviewRating: {
            '@type': 'Rating',
            ratingValue: '4.8',
            bestRating: '5',
          },
          author: {
            '@type': 'Organization',
            name: 'Emerald AGI',
          },
        },
      },
    };
  }

  generateYouTubeMetadata(product) {
    const tags = this.getYouTubeTags(product.category);
    const longTail = this.getLongTailKeywords(product.category);

    return {
      title: `${product.productName} Review: ${longTail[0] || 'Complete Guide'} (${new Date().getFullYear()})`,
      description: [
        `Learn everything about ${product.productName} in this comprehensive guide.`,
        ``,
        `In this video:`,
        `- What is ${product.productName} and how it works`,
        `- Key features and benefits`,
        `- Pricing and plans`,
        `- Real use cases and examples`,
        `- Comparison with alternatives`,
        ``,
        `#${product.category || 'SaaS'} #${product.productName.replace(/\s+/g, '')} #Productivity`,
        ``,
        `Resources:`,
        `${product.url || ''}`,
      ].join('\n'),
      tags: [...tags, product.productName, `${product.productName} review`, `${product.category || 'SaaS'} tools`, 'software review', 'productivity'].slice(0, 30),
      categoryId: '28',
      privacyStatus: 'public',
      madeForKids: false,
    };
  }

  generateArticleSchema(article, product) {
    return {
      '@context': 'https://schema.org',
      '@type': 'Article',
      headline: article.title,
      description: article.description,
      author: {
        '@type': 'Organization',
        name: 'Emerald AGI',
      },
      publisher: {
        '@type': 'Organization',
        name: product.productName,
      },
      datePublished: article.generatedAt || new Date().toISOString(),
      mainEntityOfPage: {
        '@type': 'WebPage',
        '@id': product.url || '',
      },
    };
  }
}

const seo = new SEOOptimizer();
export default seo;
