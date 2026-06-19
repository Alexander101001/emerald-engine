import { Router } from 'express';

const router = Router();

const PLANS = [
  { name: 'Starter', price: 9, features: ['100 affiliate links/mo', 'Basic analytics', 'Email support'] },
  { name: 'Pro', price: 19, features: ['Unlimited links', 'Real-time dashboard', 'AI recommendations', 'Priority support'] },
  { name: 'Agency', price: 49, features: ['Everything in Pro', 'White-label reports', 'Multi-account', 'API access'] },
];

router.get('/affiliate', (req, res) => {
  res.send(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>AffiliateEngine — Auto Affiliate Dashboard</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0a0a0a; color: #fff; }
    .hero { text-align: center; padding: 100px 20px 60px; background: linear-gradient(135deg, #00c853 0%, #00e676 100%); }
    .hero h1 { font-size: 3em; margin-bottom: 20px; }
    .hero p { font-size: 1.2em; opacity: 0.9; max-width: 600px; margin: 0 auto 30px; }
    .cta { display: inline-block; padding: 15px 40px; background: #000; color: #00e676; text-decoration: none; border-radius: 30px; font-weight: bold; font-size: 1.1em; }
    .plans { display: flex; justify-content: center; gap: 20px; padding: 60px 20px; flex-wrap: wrap; }
    .plan { background: #1a1a1a; border-radius: 16px; padding: 30px; width: 280px; border: 1px solid #2a2a2a; }
    .plan h2 { font-size: 1.5em; margin-bottom: 10px; }
    .plan .price { font-size: 2.5em; font-weight: bold; color: #00e676; margin: 15px 0; }
    .plan .price span { font-size: 0.4em; color: #888; }
    .plan ul { list-style: none; margin: 20px 0; }
    .plan li { padding: 8px 0; border-bottom: 1px solid #2a2a2a; }
    .plan li::before { content: '✓ '; color: #00e676; }
    .plan .btn { display: block; text-align: center; padding: 12px; background: #00e676; color: #000; border-radius: 8px; text-decoration: none; font-weight: bold; margin-top: 20px; }
    .plan.featured { border-color: #00e676; transform: scale(1.05); }
    .features { max-width: 800px; margin: 0 auto; padding: 60px 20px; }
    .features h2 { text-align: center; margin-bottom: 40px; font-size: 2em; }
    .feature-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; }
    .feature-card { background: #1a1a1a; padding: 25px; border-radius: 12px; border: 1px solid #2a2a2a; }
    .feature-card h3 { color: #00e676; margin-bottom: 10px; }
    .footer { text-align: center; padding: 40px; color: #666; }
    .demo-link { display: inline-block; margin-top: 15px; color: #00e676; }
  </style>
</head>
<body>
  <div class="hero">
    <h1>AffiliateEngine</h1>
    <p>AI-powered affiliate marketing that works 24/7. Auto-generate links, track conversions, and optimize revenue across 50+ proxy networks.</p>
    <a class="cta" href="#plans">Start Free Trial →</a>
  </div>

  <div class="plans" id="plans">
    ${PLANS.map((p, i) => `<div class="plan${i === 1 ? ' featured' : ''}">
      <h2>${p.name}</h2>
      <div class="price">$${p.price}<span>/mo</span></div>
      <ul>${p.features.map(f => `<li>${f}</li>`).join('')}</ul>
      <a class="btn" href="/api/checkout?plan=${p.name.toLowerCase()}">Get Started</a>
    </div>`).join('')}
  </div>

  <div class="features">
    <h2>Powered by Emerald Engine</h2>
    <div class="feature-grid">
      <div class="feature-card"><h3>50+ Proxy Providers</h3><p>Rotate through 5 tiers of residential, datacenter, and stealth proxies.</p></div>
      <div class="feature-card"><h3>AI Product Matching</h3><p>Groq/Together/OpenRouter auto-match content to high-converting products.</p></div>
      <div class="feature-card"><h3>Auto-Sync Every 6h</h3><p>New affiliate opportunities discovered and deployed automatically.</p></div>
      <div class="feature-card"><h3>Multi-Network</h3><p>Amazon, ShareASale, ClickBank — all from one dashboard.</p></div>
      <div class="feature-card"><h3>Real-time Analytics</h3><p>Clicks, conversions, revenue — updated every cycle.</p></div>
      <div class="feature-card"><h3>Self-Evolving</h3><p>The engine learns which products convert and doubles down.</p></div>
    </div>
  </div>

  <div class="footer">
    <p>Emerald Engine — Autonomous Monetization Platform</p>
    <a class="demo-link" href="/api/status">→ System Status</a>
  </div>
</body>
</html>`);
});

export default router;
