import logger from '../utils/logger.js';

const STRIPE_API_BASE = 'https://api.stripe.com/v1';

function getSecret() {
  return process.env.STRIPE_SECRET_KEY || '';
}

function getPubKey() {
  return process.env.STRIPE_PUBLISHABLE_KEY || 'pk_test_xxxxxxxxxxxx';
}

async function stripeReq(method, path, body) {
  const key = getSecret();
  if (!key) {
    logger.warn('stripe: no STRIPE_SECRET_KEY, using mock mode');
    return mockStripeResponse(method, path, body);
  }

  const params = body ? new URLSearchParams(body).toString() : undefined;
  const res = await fetch(`${STRIPE_API_BASE}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${key}`,
      'Content-Type': 'application/x-www-form-urlencoded',
      'Stripe-Version': '2023-10-16',
    },
    body: params,
  });

  if (!res.ok) {
    const err = await res.text();
    throw new Error(`Stripe ${res.status}: ${err.slice(0, 200)}`);
  }
  return res.json();
}

export async function createCheckoutSession({ priceId, userId, successUrl, cancelUrl, email }) {
  const session = await stripeReq('POST', '/checkout/sessions', {
    mode: 'subscription',
    payment_method_types: ['card'],
    line_items: [{ price: priceId, quantity: 1 }],
    client_reference_id: userId,
    customer_email: email || undefined,
    success_url: successUrl || 'https://emerald.app/success?session_id={CHECKOUT_SESSION_ID}',
    cancel_url: cancelUrl || 'https://emerald.app/pricing',
    metadata: { userId: userId || '' },
  });

  logger.info(`stripe: checkout session created ${session.id} for user ${userId}`);
  return { id: session.id, url: session.url, publishableKey: getPubKey() };
}

export async function createPortalSession(customerId, returnUrl) {
  const session = await stripeReq('POST', '/billing_portal/sessions', {
    customer: customerId,
    return_url: returnUrl || 'https://emerald.app/account',
  });
  return { url: session.url };
}

export async function getSubscription(subscriptionId) {
  return stripeReq('GET', `/subscriptions/${subscriptionId}`);
}

export async function listPrices() {
  return stripeReq('GET', '/prices?active=true&type=recurring');
}

export async function createProduct(name, amountCents, interval = 'month') {
  const product = await stripeReq('POST', '/products', { name });
  const price = await stripeReq('POST', '/prices', {
    product: product.id,
    unit_amount: String(amountCents),
    currency: 'usd',
    recurring: { interval },
  });
  return { product, price };
}

export async function handleWebhook(rawBody, signature, endpointSecret) {
  const key = getSecret();
  if (!key) {
    logger.warn('stripe: webhook mock mode');
    return { event: 'mock', data: { object: { status: 'active' } } };
  }

  const secret = endpointSecret || process.env.STRIPE_WEBHOOK_SECRET;
  if (!secret) throw new Error('STRIPE_WEBHOOK_SECRET not configured');

  const crypto = await import('crypto');
  const payload = typeof rawBody === 'string' ? rawBody : JSON.stringify(rawBody);
  const expectedSig = crypto.createHmac('sha256', secret).update(payload).digest('hex');

  const parts = signature.split(',').reduce((acc, p) => {
    const [k, v] = p.trim().split('=');
    if (k === 'v1') acc.v1 = v;
    return acc;
  }, {});

  if (!parts.v1 || !crypto.timingSafeEqual(Buffer.from(parts.v1), Buffer.from(expectedSig))) {
    throw new Error('Stripe webhook signature verification failed');
  }

  return JSON.parse(payload);
}

export async function cancelSubscription(subscriptionId) {
  return stripeReq('DELETE', `/subscriptions/${subscriptionId}`);
}

export async function getCustomer(email) {
  const list = await stripeReq('GET', `/customers?email=${encodeURIComponent(email)}`);
  return list.data?.[0] || null;
}

export function getPublishableKey() {
  return getPubKey();
}

const PRICES = [
  { id: 'price_free', lookup: 'free', name: 'Free', amount: 0, interval: 'month' },
  { id: 'price_starter', lookup: 'starter', name: 'Starter', amount: 499, interval: 'month' },
  { id: 'price_pro', lookup: 'pro', name: 'Pro', amount: 999, interval: 'month' },
  { id: 'price_lifetime', lookup: 'lifetime', name: 'Lifetime', amount: 9900, interval: 'once' },
];

export function getLocalPrices() {
  return PRICES;
}

function mockStripeResponse(method, path, body) {
  if (path === '/checkout/sessions' && method === 'POST') {
    return {
      id: `cs_mock_${Date.now()}`,
      url: `https://checkout.stripe.com/mock?userId=${body?.client_reference_id || ''}`,
      status: 'mock',
    };
  }
  if (path.startsWith('/prices')) {
    return { data: PRICES };
  }
  if (path === '/billing_portal/sessions' && method === 'POST') {
    return { url: 'https://billing.stripe.com/mock/portal' };
  }
  return { id: 'mock_default', status: 'mock' };
}

export default {
  createCheckoutSession, createPortalSession, getSubscription,
  listPrices, createProduct, handleWebhook, cancelSubscription,
  getCustomer, getLocalPrices, getPublishableKey,
};
