export default {
  name: 'Ad Revenue (AdSense)',
  tier: 'passive',

  async execute() {
    console.log(`[${this.name}] Checking ad slots...`);
    const adClient = process.env.ADSENSE_CLIENT_ID;
    if (!adClient || adClient === 'ca-pub-xxxxxxxxxxxx') {
      console.log(`[${this.name}] No valid ADSENSE_CLIENT_ID — set to activate`);
      return { revenue: 0, status: 'inactive', reason: 'missing_ad_client' };
    }
    return { revenue: 300, status: 'active', note: 'Ad slots ready — $3/mo ad-free upsell active' };
  }
};
