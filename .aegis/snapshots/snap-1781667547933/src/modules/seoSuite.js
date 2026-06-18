import { sanitizeHtml, cache } from '../utils/helpers.js';
import logger from '../utils/logger.js';

const seoCache = cache(86400000);

export function analyzeSEO(html, url) {
  const cached = seoCache.get(url || html.slice(0, 100));
  if (cached) return cached;

  const report = {
    score: 100,
    issues: [],
    suggestions: [],
    meta: {},
  };

  const titleMatch = html.match(/<title>([^<]*)<\/title>/i);
  const descMatch = html.match(/<meta\s+name="description"\s+content="([^"]*)"/i);
  const h1s = (html.match(/<h1[^>]*>/gi) || []).length;
  const imgs = (html.match(/<img[^>]*>/gi) || []).length;
  const imgsWithAlt = (html.match(/<img[^>]*\salt=[^>]*>/gi) || []).length;
  const words = html.replace(/<[^>]*>/g, '').trim().split(/\s+/).length;
  const hasViewport = /name="viewport"/i.test(html);
  const hasOG = /property="og:/i.test(html);
  const hasCanonical = /rel="canonical"/i.test(html);
  const hasLang = /lang=["']\w+["']/i.test(html);
  const hasFavicon = /favicon/i.test(html);
  const hasSchema = /application\/ld\+json/i.test(html);

  const title = titleMatch ? titleMatch[1] : '';
  const desc = descMatch ? descMatch[1] : '';

  report.meta = { title, description: desc, wordCount: words, h1Count: h1s };

  if (!title) { report.issues.push('Missing <title>'); report.score -= 15; }
  else if (title.length < 10) { report.issues.push('Title too short'); report.score -= 5; }
  else if (title.length > 60) { report.suggestions.push('Title exceeds 60 characters'); report.score -= 3; }

  if (!desc) { report.issues.push('Missing meta description'); report.score -= 10; }
  else if (desc.length < 50) { report.suggestions.push('Description too short (<50 chars)'); report.score -= 3; }

  if (h1s === 0) { report.issues.push('No H1 heading'); report.score -= 8; }
  else if (h1s > 1) { report.suggestions.push('Multiple H1 tags found'); report.score -= 3; }

  if (!hasViewport) { report.issues.push('Missing viewport meta'); report.score -= 10; }
  if (!hasOG) { report.suggestions.push('Missing Open Graph tags'); report.score -= 3; }
  if (!hasCanonical) { report.suggestions.push('Missing canonical URL'); report.score -= 2; }
  if (!hasLang) { report.suggestions.push('Missing lang attribute on <html>'); report.score -= 2; }
  if (!hasFavicon) { report.suggestions.push('Missing favicon'); report.score -= 2; }
  if (!hasSchema) { report.suggestions.push('Consider adding structured data (JSON-LD)'); report.score -= 2; }

  if (words < 300) { report.suggestions.push(`Low word count (${words}, min 300 recommended)`); report.score -= 5; }

  const missingAlt = imgs - imgsWithAlt;
  if (missingAlt > 0) { report.issues.push(`${missingAlt} image(s) missing alt text`); report.score -= missingAlt * 3; }

  report.score = Math.max(0, report.score);
  seoCache.set(url || html.slice(0, 100), report);
  return report;
}

export function generateMetaTags({ title, description, canonical, ogImage, keywords }) {
  const safeTitle = sanitizeHtml(title || '');
  const safeDesc = sanitizeHtml(description || '');
  const safeUrl = sanitizeHtml(canonical || '');
  const safeImg = sanitizeHtml(ogImage || '');

  return [
    `<title>${safeTitle}</title>`,
    `<meta name="description" content="${safeDesc}" />`,
    `<meta name="keywords" content="${sanitizeHtml(keywords || '')}" />`,
    `<meta property="og:title" content="${safeTitle}" />`,
    `<meta property="og:description" content="${safeDesc}" />`,
    safeUrl && `<meta property="og:url" content="${safeUrl}" />`,
    safeImg && `<meta property="og:image" content="${safeImg}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<meta name="twitter:title" content="${safeTitle}" />`,
    `<meta name="twitter:description" content="${safeDesc}" />`,
    safeUrl && `<link rel="canonical" href="${safeUrl}" />`,
    `<script type="application/ld+json">${JSON.stringify({
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: safeTitle,
      description: safeDesc,
      url: safeUrl || undefined,
    })}</script>`,
  ].filter(Boolean).join('\n  ');
}

export function injectSEO(html, tags) {
  return html.replace('</head>', `  ${tags}\n</head>`);
}

export default { analyzeSEO, generateMetaTags, injectSEO };
