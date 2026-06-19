export class ProxyManager {
    constructor() {
        this.providers = {
            // Tier 1: Premium datacenter/residential (anti-ban, high success)
            premium: [
                'Bright Data', 'Oxylabs', 'Smartproxy', 'IPRoyal', 'Soax',
                'NetNut', 'Rayobyte', 'Proxy-Cheap', 'Proxy-Seller', 'Storm Proxies',
            ],
            // Tier 2: Scraping API services (no IP management needed)
            scraping: [
                'ScraperAPI', 'ScrapingBee', 'ZenRows', 'Apify', 'ScrapingFish',
                'ScrapingDog', 'ScrapingAnt', 'Crawlbase', 'Scrapestack', 'ScrapingPass',
            ],
            // Tier 3: Stealth/anti-detect browsers (fingerprint rotation)
            stealth: [
                'GoLogin', 'AdsPower', 'Dolphin{anty}', 'Multilogin', 'Indigo',
                'Kameleo', 'Octo Browser', 'Undetectable', 'Maskfog', 'Lalicat',
            ],
            // Tier 4: ISP/residential VPN (high anonymity)
            residential: [
                'AYCD', 'Proxy-Seller Residential', 'IPVanish', 'TorGuard', 'Windscribe',
                'Private Internet Access', 'HideMyAss', 'NordVPN', 'Surfshark', 'VyprVPN',
            ],
            // Tier 5: CDN/edge network (speed + geo-distribution)
            cdn: [
                'Cloudflare Workers', 'Cloudflare WARP', 'Fastly', 'Akamai', 'StackPath',
                'Bunny CDN', 'Google Cloud CDN', 'AWS CloudFront', 'Azure CDN', 'KeyCDN',
            ],
        };
        this.failover = {};
    }

    async getRoute(taskType) {
        const pool = this.providers[taskType] || this.providers.scraping;
        const rotated = this._rotate(taskType, pool);
        this.failover[taskType] = this.failover[taskType] || 0;
        const idx = this.failover[taskType] % pool.length;
        this.failover[taskType]++;
        console.log(`[PROXY] ${taskType} → ${rotated} [${idx + 1}/${pool.length}]`);
        return rotated;
    }

    _rotate(taskType, pool) {
        const base = this.failover[taskType] || 0;
        return pool[base % pool.length];
    }

    getPoolSize() {
        return Object.values(this.providers).reduce((sum, p) => sum + p.length, 0);
    }

    getStatus() {
        const pools = {};
        for (const [tier, list] of Object.entries(this.providers)) {
            pools[tier] = { active: list.length, current: list[this.failover[tier] % list.length] || list[0] };
        }
        return { totalProviders: this.getPoolSize(), pools };
    }
}
