import { slugify, uid } from '../utils/helpers.js';
import logger from '../utils/logger.js';

export function generateStripeTemplate(productName) {
  const slug = slugify(productName);
  return {
    path: 'src/monetization/stripe-checkout.js',
    content: `import { loadStripe } from '@stripe/stripe-js';

const STRIPE_PUBLISHABLE_KEY = process.env.NEXT_PUBLIC_STRIPE_KEY || 'pk_test_xxxxxxxxxxxx';

const PRICES = [
  { id: 'price_starter', lookup: 'starter', name: 'Starter', amount: 499, interval: 'month', features: ['Core features', 'Email support'] },
  { id: 'price_pro', lookup: 'pro', name: 'Pro', amount: 999, interval: 'month', features: ['All features', 'Priority support', 'API access'] },
  { id: 'price_enterprise', lookup: 'enterprise', name: 'Enterprise', amount: 2999, interval: 'month', features: ['Custom features', 'Dedicated support', 'SLA', 'White-label'] },
];

export async function redirectToCheckout(priceId, userId) {
  try {
    const stripe = await loadStripe(STRIPE_PUBLISHABLE_KEY);
    const res = await fetch('/api/create-checkout-session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ priceId, userId, successUrl: window.location.origin + '/success', cancelUrl: window.location.origin + '/pricing' }),
    });
    const session = await res.json();
    const result = await stripe.redirectToCheckout({ sessionId: session.id });
    if (result.error) throw new Error(result.error.message);
  } catch (e) {
    console.error('Checkout error:', e);
    window.location.href = '/pricing?error=' + encodeURIComponent(e.message);
  }
}

export function renderPricingTable() {
  return PRICES.map(tier => \`
    <div class="pricing-tier">
      <h3>\${tier.name}</h3>
      <div class="price">$\${(tier.amount / 100).toFixed(2)}<span>/\${tier.interval}</span></div>
      <ul>\${tier.features.map(f => '<li>' + f + '</li>').join('')}</ul>
      <button onclick="redirectToCheckout('\${tier.id}', 'guest')" class="btn btn-primary">\${tier.amount === 0 ? 'Get Started' : 'Subscribe'}</button>
    </div>
  \`).join('');
}

export const STRIPE_ENV_TEMPLATE = \`STRIPE_SECRET_KEY=sk_test_your_secret_key
STRIPE_PUBLISHABLE_KEY=pk_test_your_publishable_key
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret\`;
`,
  };
}

export function generateLandingPage(trend) {
  const title = trend?.title || 'SaaS Landing';
  const slug = slugify(title);
  const tagline = trend?.description ? trend.description.slice(0, 100) : 'Built with Emerald AGI';
  const category = trend?.category || 'productivity';
  const year = new Date().getFullYear();

  const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>${title} — SaaS Solution</title>
  <meta name="description" content="${tagline}" />
  <meta property="og:title" content="${title}" />
  <meta property="og:description" content="${tagline}" />
  <meta property="og:type" content="website" />
  <meta name="twitter:card" content="summary_large_image" />
  <link rel="canonical" href="https://${slug}.vercel.app" />
  <script type="application/ld+json">
  {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    "name": "${title}",
    "applicationCategory": "BusinessApplication",
    "description": "${tagline}",
    "offers": {
      "@type": "AggregateOffer",
      "lowPrice": "4.99",
      "highPrice": "29.99",
      "priceCurrency": "USD"
    }
  }
  </script>
  <script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js"></script>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;line-height:1.6;color:#1a1a2e;background:#f8f9fa}
    .hero{text-align:center;padding:5rem 2rem;background:linear-gradient(135deg,#667eea,#764ba2);color:#fff}
    .hero h1{font-size:2.8rem;margin-bottom:0.5rem}
    .hero p{font-size:1.2rem;opacity:0.9;max-width:600px;margin:1rem auto}
    .btn{display:inline-block;padding:0.8rem 2rem;background:#fff;color:#667eea;border-radius:6px;text-decoration:none;font-weight:600;margin-top:1rem}
    .btn:hover{background:#f0f0ff}
    .features{display:grid;grid-template-columns:repeat(auto-fit,minmax(250px,1fr));gap:2rem;padding:4rem 2rem;max-width:1100px;margin:0 auto}
    .feature-card{background:#fff;padding:2rem;border-radius:10px;box-shadow:0 4px 12px rgba(0,0,0,0.08);text-align:center}
    .feature-card h3{color:#667eea;margin-bottom:0.5rem}
    .pricing{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:2rem;padding:4rem 2rem;max-width:1100px;margin:0 auto}
    .pricing-tier{background:#fff;padding:2rem;border-radius:10px;box-shadow:0 4px 12px rgba(0,0,0,0.08);text-align:center}
    .pricing-tier .price{font-size:2rem;font-weight:700;color:#1a1a2e;margin:1rem 0}
    .pricing-tier .price span{font-size:0.9rem;color:#666}
    .pricing-tier ul{list-style:none;padding:0;margin:1rem 0}
    .pricing-tier ul li{padding:0.3rem 0;border-bottom:1px solid #eee}
    .pricing-tier ul li:last-child{border:none}
    .cta{text-align:center;padding:4rem 2rem;background:#1a1a2e;color:#fff}
    .cta h2{font-size:2rem;margin-bottom:1rem}
    .cta .btn{background:#667eea;color:#fff}
    .cta .btn:hover{background:#5a6fd6}
    .ad-container{margin:2rem auto;text-align:center;min-height:90px}
    footer{text-align:center;padding:2rem;color:#999;font-size:0.85rem}
    footer a{color:#667eea;text-decoration:none}
    @media(max-width:600px){.hero h1{font-size:1.8rem}}
  </style>
</head>
<body>
  <section class="hero">
    <h1>${title}</h1>
    <p>${tagline}</p>
    <a href="/pricing" class="btn">Get Started Free</a>
  </section>

  <section class="features">
    <div class="feature-card">
      <h3>Lightning Fast</h3>
      <p>Built for speed with modern architecture</p>
    </div>
    <div class="feature-card">
      <h3>Secure</h3>
      <p>Enterprise-grade security out of the box</p>
    </div>
    <div class="feature-card">
      <h3>Scalable</h3>
      <p>Grows with your business seamlessly</p>
    </div>
  </section>

  <div class="ad-container">
    <ins class="adsbygoogle" style="display:block" data-ad-client="ca-pub-xxxxxxxxxxxx" data-ad-slot="emerald-slot-0" data-ad-format="auto"></ins>
    <script>(adsbygoogle = window.adsbygoogle || []).push({});</script>
  </div>

  <section class="pricing" id="pricing">
    <div class="pricing-tier">
      <h3>Starter</h3>
      <div class="price">$4.99<span>/month</span></div>
      <ul>
        <li>Core features</li>
        <li>Email support</li>
        <li>1 project</li>
      </ul>
      <a href="/api/create-checkout-session?price=price_starter" class="btn">Subscribe</a>
    </div>
    <div class="pricing-tier">
      <h3>Pro</h3>
      <div class="price">$9.99<span>/month</span></div>
      <ul>
        <li>All features</li>
        <li>Priority support</li>
        <li>API access</li>
        <li>Unlimited projects</li>
      </ul>
      <a href="/api/create-checkout-session?price=price_pro" class="btn">Subscribe</a>
    </div>
    <div class="pricing-tier">
      <h3>Enterprise</h3>
      <div class="price">$29.99<span>/month</span></div>
      <ul>
        <li>Custom features</li>
        <li>Dedicated support</li>
        <li>SLA guarantee</li>
        <li>White-label option</li>
      </ul>
      <a href="/api/create-checkout-session?price=price_enterprise" class="btn">Contact Sales</a>
    </div>
  </section>

  <section class="cta">
    <h2>Ready to Get Started?</h2>
    <p style="margin-bottom:1.5rem;opacity:0.9">Join thousands of teams using ${title}</p>
    <a href="/pricing" class="btn">Start Free Trial</a>
  </section>

  <footer>
    <p>&copy; ${year} ${title} — <a href="/privacy">Privacy Policy</a> | <a href="/terms">Terms of Service</a></p>
  </footer>
  <script src="/js/ads.js"></script>
</body>
</html>`;

  return html;
}

export function generateCheckoutAPI() {
  return {
    path: 'api/create-checkout-session.js',
    content: `import Stripe from 'stripe';

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY, { apiVersion: '2023-10-16' });

export default async function handler(req, res) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { priceId, userId, successUrl, cancelUrl } = req.body;

  if (!priceId) {
    return res.status(400).json({ error: 'priceId is required' });
  }

  try {
    const session = await stripe.checkout.sessions.create({
      mode: 'subscription',
      payment_method_types: ['card'],
      line_items: [{ price: priceId, quantity: 1 }],
      client_reference_id: userId || 'anonymous',
      success_url: successUrl || req.headers.origin + '/success?session_id={CHECKOUT_SESSION_ID}',
      cancel_url: cancelUrl || req.headers.origin + '/pricing',
      metadata: { userId: userId || '' },
    });

    return res.status(200).json({ id: session.id, url: session.url });
  } catch (e) {
    return res.status(500).json({ error: e.message });
  }
}`,
  };
}

export function generateSEOConfig(productName, tagline) {
  return {
    path: 'src/config/seo.js',
    content: `export default {
  siteName: '${productName}',
  tagline: '${tagline || 'SaaS built with Emerald'}',
  language: 'en',
  locale: 'en_US',
  twitterHandle: '@emerald_agi',
  defaultImage: '/og-image.png',
  themeColor: '#667eea',
  structuredData: {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: '${productName}',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    offers: {
      '@type': 'AggregateOffer',
      lowPrice: '4.99',
      highPrice: '29.99',
      priceCurrency: 'USD',
    },
  },
};`,
  };
}

export function generatePrivacyPage(productName) {
  const year = new Date().getFullYear();
  const slug = slugify(productName);
  return {
    path: 'public/privacy/index.html',
    content: `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Privacy Policy — ${productName}</title>
  <meta name="robots" content="noindex" />
</head>
<body style="max-width:800px;margin:0 auto;padding:2rem;font-family:sans-serif;line-height:1.6">
  <h1>Privacy Policy</h1>
  <p>Last updated: ${new Date().toISOString().split('T')[0]}</p>
  <h2>Data Collection</h2>
  <p>We collect only the data necessary to provide our service: email address, payment information (processed securely by Stripe), and usage analytics.</p>
  <h2>Data Usage</h2>
  <p>Your data is used solely to operate and improve our service. We never sell your personal information to third parties.</p>
  <h2>Third-Party Services</h2>
  <p>We use Stripe for payment processing. Stripe's privacy policy applies to payment data. We use Google AdSense for advertising; their privacy policy applies to ad personalization.</p>
  <h2>Contact</h2>
  <p>Email: privacy@${slug}.com</p>
  <p>&copy; ${year} ${productName}</p>
</body>
</html>`,
  };
}

export function generateTermsPage(productName) {
  const year = new Date().getFullYear();
  return {
    path: 'public/terms/index.html',
    content: `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Terms of Service — ${productName}</title>
  <meta name="robots" content="noindex" />
</head>
<body style="max-width:800px;margin:0 auto;padding:2rem;font-family:sans-serif;line-height:1.6">
  <h1>Terms of Service</h1>
  <p>Last updated: ${new Date().toISOString().split('T')[0]}</p>
  <h2>Use of Service</h2>
  <p>By using ${productName}, you agree to these terms. You must be at least 18 years old to use this service.</p>
  <h2>Subscriptions</h2>
  <p>Subscriptions auto-renew monthly. You can cancel at any time from your account dashboard. Refunds are handled per our refund policy.</p>
  <h2>Limitation of Liability</h2>
  <p>${productName} is provided "as is" without warranty of any kind. We are not liable for any damages arising from use of this service.</p>
  <p>&copy; ${year} ${productName}</p>
</body>
</html>`,
  };
}

export default {
  generateStripeTemplate,
  generateLandingPage,
  generateCheckoutAPI,
  generateSEOConfig,
  generatePrivacyPage,
  generateTermsPage,
};
