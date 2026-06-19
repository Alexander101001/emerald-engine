import { Router } from 'express';

const router = Router();

router.get('/api/checkout', (req, res) => {
  const plan = req.query.plan || 'starter';
  const prices = { starter: 900, pro: 1900, agency: 4900 };
  const amount = prices[plan] || 900;

  res.send(`<!DOCTYPE html>
<html>
<head><title>Checkout — AffiliateEngine</title>
<style>
  body { font-family: sans-serif; background: #0a0a0a; color: #fff; display: flex; justify-content: center; align-items: center; height: 100vh; }
  .card { background: #1a1a1a; padding: 40px; border-radius: 16px; border: 1px solid #2a2a2a; text-align: center; max-width: 400px; }
  .price { font-size: 3em; color: #00e676; font-weight: bold; }
  .btn { display: block; padding: 15px; background: #00e676; color: #000; border-radius: 8px; text-decoration: none; font-weight: bold; margin: 20px 0; font-size: 1.1em; }
  .note { color: #666; font-size: 0.8em; }
</style></head>
<body>
  <div class="card">
    <h2>${plan.charAt(0).toUpperCase() + plan.slice(1)} Plan</h2>
    <div class="price">$${amount / 100}<span style="font-size:0.3em;color:#666">/mo</span></div>
    <p style="color:#888;margin:10px 0;">Stripe integration ready. Add your Stripe key to activate payments.</p>
    <a class="btn" href="/affiliate">← Back</a>
    <p class="note">API keys encrypted via AES-256. No data stored on our servers.</p>
  </div>
</body>
</html>`);
});

export default router;
