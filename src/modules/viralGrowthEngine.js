import logger from '../utils/logger.js';
import { generateArticles, generateTwitterThread, generateFacebookPost, generateInstagramPost, generateVideoScript } from './contentFactory.js';
import publisher from './multiChannelPublisher.js';
import social from './socialMediaAutomation.js';
import traffic from './trafficDriver.js';
import seo from './seoOptimizer.js';

export class ViralGrowthEngine {
  constructor() {
    this._campaigns = [];
    this._totalReach = 0;
    this._active = false;
  }

  async executeCampaign(product) {
    logger.info(`viral: starting Viral-First campaign for "${product.productName}"`);
    const startTime = Date.now();

    const articles = generateArticles(product);
    logger.info(`viral: generated ${articles.length} SEO articles`);

    const keywords = seo.getLongTailKeywords(product.category);
    const seoMeta = seo.generateSEOMeta(product, keywords);
    const youtubeMeta = seo.generateYouTubeMetadata(product);

    for (const article of articles) {
      article.seoMeta = seoMeta;
      article.schemaOrg = seo.generateArticleSchema(article, product);
    }

    const publishResults = await publisher.publishAll(articles, product);
    const publishedCount = publishResults.filter(r => r.success).length;

    const threads = generateTwitterThread(product, articles);
    const fbPosts = articles.map(a => generateFacebookPost(product, a));
    const igPosts = articles.map(a => generateInstagramPost(product, a));
    const videoScripts = articles.map(a => generateVideoScript(product, a, 'youtube'));
    const shortScripts = articles.map(a => generateVideoScript(product, a, 'tiktok'));

    const socialResults = await social.publishAllSocial(threads, fbPosts, igPosts, [...videoScripts, ...shortScripts], product);
    const socialPublished = socialResults.filter(r => r.success).length;

    const trafficResults = await traffic.driveAllTraffic(articles, product);
    const trafficPublished = trafficResults.filter(r => r.success).length;

    const total = publishedCount + socialPublished + trafficPublished;
    const duration = Date.now() - startTime;

    const campaign = {
      product: product.productName,
      articlesGenerated: articles.length,
      articlesPublished: publishedCount,
      socialPosts: socialPublished,
      trafficDrives: trafficPublished,
      totalPublications: total,
      keywords: keywords.slice(0, 5),
      seoScore: seoMeta.title.length > 10 && seoMeta.description.length > 50 ? 85 : 60,
      durationMs: duration,
      timestamp: new Date().toISOString(),
      publishResults,
      socialResults,
      trafficResults,
    };

    this._campaigns.push(campaign);
    this._totalReach += total * 500;
    this._active = true;

    logger.info(`viral: campaign complete — ${total} publications in ${(duration / 1000).toFixed(1)}s for "${product.productName}"`);
    return campaign;
  }

  async executeAllCampaigns(products) {
    const results = [];
    for (const product of products) {
      const campaign = await this.executeCampaign(product);
      results.push(campaign);
    }
    return results;
  }

  generateReport(campaign) {
    const lines = [
      `Viral Growth Campaign Report`,
      `Product: ${campaign.product}`,
      `Generated: ${campaign.articlesGenerated} SEO articles`,
      `Published: ${campaign.totalPublications} total pieces`,
      `  - Articles published: ${campaign.articlesPublished}`,
      `  - Social media posts: ${campaign.socialPosts}`,
      `  - Traffic drives: ${campaign.trafficDrives}`,
      `Target Keywords: ${campaign.keywords.join(', ')}`,
      `SEO Score: ${campaign.seoScore}/100`,
      `Duration: ${(campaign.durationMs / 1000).toFixed(1)}s`,
    ];

    if (campaign.publishResults) {
      const platforms = {};
      for (const r of campaign.publishResults) {
        if (r.success) {
          platforms[r.platform] = (platforms[r.platform] || 0) + 1;
        }
      }
      lines.push(`Platform breakdown:`);
      for (const [p, c] of Object.entries(platforms)) {
        lines.push(`  - ${p}: ${c} posts`);
      }
    }

    return lines.join('\n');
  }

  getSummary() {
    const totalPubs = this._campaigns.reduce((s, c) => s + c.totalPublications, 0);
    return {
      active: this._active,
      campaignsRun: this._campaigns.length,
      totalPublications: totalPubs,
      estimatedReach: this._totalReach,
      lastCampaign: this._campaigns[this._campaigns.length - 1] || null,
    };
  }
}

const viral = new ViralGrowthEngine();
export default viral;
