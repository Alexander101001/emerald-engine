import { slugify } from '../utils/helpers.js';
import logger from '../utils/logger.js';

const ARTICLE_TEMPLATES = {
  tutorial: {
    title: (p) => `How to ${p.productName} — A Complete Step-by-Step Guide for ${new Date().getFullYear()}`,
    sections: [
      (p) => `## Why ${p.productName} Matters`,
      (p) => `## Getting Started with ${p.productName}`,
      (p) => `## Advanced ${p.productName} Techniques`,
      (p) => `## Common Mistakes to Avoid`,
      (p) => `## Conclusion: Is ${p.productName} Right for You?`,
    ],
    tags: (p) => [p.productName, 'tutorial', 'guide', 'how-to', p.category || 'saas'].filter(Boolean),
    wordCount: 1200,
  },
  listicle: {
    title: (p) => `10 Reasons Why ${p.productName} Is Transforming ${p.category || 'the Industry'} in ${new Date().getFullYear()}`,
    sections: [
      (p) => `## 1. ${p.productName} Solves a Critical Pain Point`,
      (p) => `## 2. Built for Modern Teams`,
      (p) => `## 3. Cost-Effective Alternative to Legacy Tools`,
      (p) => `## 4. Seamless Integration Ecosystem`,
      (p) => `## 5. Enterprise-Grade Security`,
      (p) => `## 6. Scalable Architecture`,
      (p) => `## 7. Stellar Customer Support`,
      (p) => `## 8. Data-Driven Insights`,
      (p) => `## 9. Continuous Innovation`,
      (p) => `## 10. Proven ROI`,
      (p) => `## Final Verdict`,
    ],
    tags: (p) => [p.productName, p.category || 'saas', 'features', 'benefits', 'review'].filter(Boolean),
    wordCount: 1500,
  },
  comparison: {
    title: (p) => `${p.productName} vs. Competitors: Which ${p.category || 'Solution'} Wins in ${new Date().getFullYear()}?`,
    sections: [
      (p) => `## Overview: The ${p.category || 'SaaS'} Landscape`,
      (p) => `## Feature Comparison Table`,
      (p) => `## Pricing: ${p.productName} vs. The Market`,
      (p) => `## Ease of Use`,
      (p) => `## Integration Capabilities`,
      (p) => `## Customer Support & Community`,
      (p) => `## Scalability & Performance`,
      (p) => `## Verdict: Why ${p.productName} Comes Out Ahead`,
    ],
    tags: (p) => [p.productName, 'comparison', 'vs', 'alternatives', p.category || 'saas'].filter(Boolean),
    wordCount: 1800,
  },
};

function generateArticleBody(template, product) {
  const body = template.sections.map((sectionFn) => {
    const heading = sectionFn(product);
    const content = generateSectionContent(heading, product);
    return `${heading}\n\n${content}`;
  });
  return body.join('\n\n');
}

function generateSectionContent(heading, product) {
  const templates = [
    `When it comes to ${product.category || 'modern software solutions'}, ${product.productName} stands out as a game-changer. Teams across the globe are adopting this approach to streamline their workflows and drive measurable results.`,
    `${product.productName} addresses the core challenges that businesses face today. With its intuitive interface and powerful backend, it enables organizations to focus on what matters most: growth.`,
    `Industry experts have consistently highlighted ${product.productName} as a top contender in the ${product.category || 'SaaS'} space. The platform's commitment to innovation and user experience sets it apart from traditional alternatives.`,
    `One of the key advantages of ${product.productName} is its flexibility. Whether you're a small startup or a large enterprise, the platform scales to meet your unique requirements without compromising performance.`,
    `The ${product.productName} ecosystem includes robust APIs, seamless third-party integrations, and a thriving community of developers and power users who continuously contribute to its evolution.`,
    `Security is paramount in today's digital landscape. ${product.productName} employs enterprise-grade encryption, regular security audits, and compliance with industry standards including SOC 2 and GDPR.`,
    `Getting started with ${product.productName} is straightforward. The platform offers comprehensive documentation, video tutorials, and a dedicated support team to ensure a smooth onboarding experience.`,
    `Analytics and reporting capabilities in ${product.productName} provide actionable insights that drive strategic decision-making. Real-time dashboards and custom reports keep your team aligned and informed.`,
    `Pricing for ${product.productName} is transparent and competitive. With flexible plans ranging from free tiers to enterprise options, there's a solution for every budget and requirement.`,
    `The future of ${product.productName} looks promising, with a roadmap full of exciting features and improvements. Early adopters and enterprise customers get exclusive access to beta features and priority support.`,
  ];
  const idx = Math.abs(heading.length + product.productName.length) % templates.length;
  return templates[idx];
}

function generateMetaDescription(product, template) {
  const descs = [
    `Discover how ${product.productName} can transform your ${product.category || 'business'}. Complete guide with features, pricing, and best practices for ${new Date().getFullYear()}.`,
    `Learn everything about ${product.productName}: features, pricing, integrations, and more. The definitive ${product.category || 'guide'} for teams looking to scale.`,
    `${product.productName} review: comprehensive analysis of features, pricing, and alternatives. Find out why leading companies choose ${product.productName} for their ${product.category || 'workflow'} needs.`,
  ];
  return descs[Math.abs(product.productName.length) % descs.length];
}

export function generateArticles(product) {
  const templates = Object.values(ARTICLE_TEMPLATES);
  const articles = templates.map((template, idx) => {
    const title = template.title(product);
    const slug = slugify(title);
    const body = generateArticleBody(template, product);
    const tags = template.tags(product);
    const description = generateMetaDescription(product, template);
    const date = new Date().toISOString().split('T')[0];

    const content = `---
title: "${title}"
description: "${description}"
date: "${date}"
tags: [${tags.map(t => `"${t}"`).join(', ')}]
slug: "${slug}"
category: "${product.category || 'saas'}"
author: "Emerald AGI Content Engine"
---

# ${title}

${body}

---

*This article was generated by the Emerald AGI Content Engine. ${product.productName} is a ${product.category || 'SaaS'} solution designed to help teams achieve more.*`;

    const mediumContent = `${title}\n\n${description}\n\n${body.replace(/^## /gm, '\n\n## ')}\n\n---\n\n*Originally published on [${product.productName}](${product.url || 'https://' + slugify(product.productName) + '.vercel.app'})*`;

    const devtoContent = `---\ntitle: "${title}"\npublished: true\ndescription: "${description}"\ntags: ${tags.length > 4 ? tags.slice(0, 4).map(t => t.toLowerCase().replace(/[^a-z0-9-]/g, '')).join(', ') : tags.map(t => t.toLowerCase().replace(/[^a-z0-9-]/g, '')).join(', ')}\ncanonical_url: ${product.url || 'https://' + slugify(product.productName) + '.vercel.app'}\n---\n\n${body}`;

    return {
      type: ['tutorial', 'listicle', 'comparison'][idx],
      title,
      slug,
      description,
      tags,
      wordCount: template.wordCount,
      content,
      mediumContent,
      devtoContent,
      linkedinContent: `${title}\n\n${description}\n\n${body.replace(/^## /gm, '\n\n').slice(0, 3000)}...\n\nRead the full article: ${product.url || 'https://' + slugify(product.productName) + '.vercel.app'}`,
      generatedAt: new Date().toISOString(),
    };
  });

  logger.info(`contentFactory: generated ${articles.length} articles for "${product.productName}"`);
  return articles;
}

export function generateTwitterThread(product, articles) {
  const threads = articles.map((article, idx) => {
    const tweets = [
      `🧵 ${idx + 1}/${articles.length}: ${article.title}\n\nA thread 🧵👇`,
      `1/${article.tags.slice(0, 3).map(t => `#${t.replace(/\s+/g, '')}`).join(' ')}`,
      `2/ Why we built ${product.productName}: ${product.tagline || 'To solve real problems for real teams.'}`,
      `3/ Key features that set us apart:\n• Enterprise-grade security\n• Seamless integrations\n• Scalable architecture\n• 24/7 support`,
      `4/ ${product.productName} in numbers:\n• Used by teams worldwide\n• 99.9% uptime SLA\n• SOC 2 compliant\n• 14-day free trial`,
      `5/ Ready to get started? 👇\n${product.url || 'https://' + slugify(product.productName) + '.vercel.app'}`,
      `6/ Found this useful? Follow us for more ${product.category || 'SaaS'} insights and product updates.`,
    ];
    return { articleTitle: article.title, tweets, platform: 'twitter' };
  });
  return threads;
}

export function generateFacebookPost(product, article) {
  return {
    platform: 'facebook',
    content: `💡 ${article.title}\n\n${article.description}\n\nWe're excited to share how ${product.productName} is helping teams achieve more. Whether you're a startup or enterprise, our platform scales with you.\n\n👉 Learn more: ${product.url || 'https://' + slugify(product.productName) + '.vercel.app'}\n\n#${product.category || 'SaaS'} #Productivity #Innovation`,
    imageUrl: null,
    callToAction: 'Learn More',
  };
}

export function generateInstagramPost(product, article) {
  return {
    platform: 'instagram',
    content: `${article.title}\n\n${article.description.slice(0, 150)}\n\nLink in bio to learn more about ${product.productName} ✨\n\n.#${product.category || 'saas'} #productivity #startup #tech #innovation #business #growth #${slugify(product.productName)}`,
    imagePrompt: `Modern ${product.productName} dashboard on laptop with analytics charts`,
    callToAction: 'Link in bio',
  };
}

export function generateVideoScript(product, article, platform) {
  const isShort = platform === 'tiktok' || platform === 'youtube-shorts';
  const duration = isShort ? '30-60 seconds' : '3-5 minutes';
  const hook = isShort
    ? `Did you know ${product.productName} can save your team 10+ hours per week?`
    : `In this video, we'll explore ${article.title} and show you exactly how ${product.productName} can transform your workflow.`;

  return {
    platform,
    title: isShort ? `${product.productName}: The ${product.category || 'SaaS'} Game Changer` : article.title,
    duration,
    hook,
    script: isShort
      ? [
          `[OPENING - ${duration}]`,
          hook,
          `Here's what makes ${product.productName} different:`,
          `✅ Enterprise-grade security`,
          `✅ Seamless integrations`,
          `✅ Scalable for any team size`,
          `✅ 14-day free trial, no credit card required`,
          `[CALL TO ACTION]`,
          `Try ${product.productName} today at ${product.url || slugify(product.productName) + '.vercel.app'}`,
          `#${product.category || 'saas'} #productivity`,
        ].join('\n\n')
      : [
          `[INTRO - 30s]`,
          hook,
          ``,
          `[SECTION 1 - ${product.productName} Overview - 60s]`,
          `${article.description}`,
          ``,
          `[SECTION 2 - Key Features - 90s]`,
          `Let's dive into the core features that make ${product.productName} stand out:`,
          `• Enterprise-grade security with end-to-end encryption`,
          `• Seamless integrations with your favorite tools`,
          `• Scalable architecture that grows with you`,
          `• 24/7 customer support with real humans`,
          ``,
          `[SECTION 3 - Use Cases - 60s]`,
          `Real teams are using ${product.productName} to:`,
          `• Streamline their workflows`,
          `• Reduce operational costs`,
          `• Improve team collaboration`,
          `• Drive data-informed decisions`,
          ``,
          `[OUTRO - 30s]`,
          `Ready to see ${product.productName} in action?`,
          `Start your free trial today: ${product.url || slugify(product.productName) + '.vercel.app'}`,
          ``,
          `Don't forget to like, subscribe, and hit the bell for more ${product.category || 'tech'} content!`,
        ].join('\n'),
    tags: [`#${product.category || 'SaaS'}`, `#${slugify(product.productName)}`, '#Productivity', '#Tech', '#Innovation'],
    thumbnailSuggestion: `Split screen: ${product.productName} logo on left, dashboard screenshot on right with "10x Your Productivity" overlay`,
  };
}

export default {
  generateArticles,
  generateTwitterThread,
  generateFacebookPost,
  generateInstagramPost,
  generateVideoScript,
};
