package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const cycleDelay = 300

var baseURL = "https://emerald-engine.com"

var niches = []Niche{
	{Name: "SaaS Marketing", Keyword: "saas-marketing", Emoji: "🚀"},
	{Name: "Health & Wellness", Keyword: "health-wellness", Emoji: "💪"},
	{Name: "Personal Finance", Keyword: "personal-finance", Emoji: "💰"},
	{Name: "Digital Products", Keyword: "digital-products", Emoji: "📦"},
	{Name: "Online Education", Keyword: "online-education", Emoji: "🎓"},
	{Name: "E-commerce", Keyword: "ecommerce", Emoji: "🛒"},
	{Name: "AI Tools", Keyword: "ai-tools", Emoji: "🤖"},
	{Name: "Crypto & Web3", Keyword: "crypto-web3", Emoji: "🔗"},
}

type Niche struct {
	Name    string
	Keyword string
	Emoji   string
}

var colorSchemes = [][]string{
	{"#667eea", "#764ba2", "#ff6b6b", "#5f27cd"},
	{"#11998e", "#38ef7d", "#f39c12", "#0a8754"},
	{"#fc5c7d", "#6a82fb", "#2ecc71", "#c0392b"},
	{"#0c3483", "#a2b6df", "#e74c3c", "#1a5276"},
	{"#2c3e50", "#3498db", "#e67e22", "#1a252f"},
	{"#8e2de2", "#4a00e0", "#00b894", "#6c3483"},
	{"#1a1a2e", "#16213e", "#e94560", "#0f3460"},
	{"#0f0c29", "#302b63", "#f39c12", "#24243e"},
	{"#00b4db", "#0083b0", "#2ecc71", "#005c8a"},
	{"#a8e063", "#56ab2f", "#ff6b6b", "#3d8b37"},
}

func pick[T any](items []T) T {
	return items[rand.Intn(len(items))]
}

func pickN[T any](items []T, n int) []T {
	perm := rand.Perm(len(items))
	result := make([]T, n)
	for i := 0; i < n && i < len(perm); i++ {
		result[i] = items[perm[i]]
	}
	return result
}

func randRange(min, max int) int {
	return min + rand.Intn(max-min+1)
}

func shuffled[T any](items []T) []T {
	result := make([]T, len(items))
	copy(result, items)
	rand.Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	return result
}

// ─── Ad / Affiliate / Monetization Snippets ───

func googleAnalytics() string {
	return `<!-- Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=G-XXXXXXXXXX"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', 'G-XXXXXXXXXX');
</script>`
}

func adsenseBanner(clientID string) string {
	if clientID == "" {
		clientID = "ca-pub-XXXXXXXXXXXXXXXX"
	}
	return fmt.Sprintf(`<!-- AdSense Banner 728x90 -->
<ins class="adsbygoogle"
     style="display:block"
     data-ad-client="%s"
     data-ad-slot="XXXXXXXXXX"
     data-ad-format="auto"
     data-full-width-responsive="true"></ins>
<script>(adsbygoogle = window.adsbygoogle || []).push({});</script>`, clientID)
}

func adsenseInArticle(clientID string) string {
	if clientID == "" {
		clientID = "ca-pub-XXXXXXXXXXXXXXXX"
	}
	return fmt.Sprintf(`<!-- AdSense In-Article 336x280 -->
<ins class="adsbygoogle"
     style="display:block; text-align:center;"
     data-ad-layout="in-article"
     data-ad-format="fluid"
     data-ad-client="%s"
     data-ad-slot="XXXXXXXXXX"></ins>
<script>(adsbygoogle = window.adsbygoogle || []).push({});</script>`, clientID)
}

func adsenseSidebar(clientID string) string {
	if clientID == "" {
		clientID = "ca-pub-XXXXXXXXXXXXXXXX"
	}
	return fmt.Sprintf(`<!-- AdSense Sidebar 300x250 -->
<ins class="adsbygoogle"
     style="display:block"
     data-ad-client="%s"
     data-ad-slot="XXXXXXXXXX"
     data-ad-format="auto"
     data-full-width-responsive="true"></ins>
<script>(adsbygoogle = window.adsbygoogle || []).push({});</script>`, clientID)
}

func adsenseMatchedContent(clientID string) string {
	if clientID == "" {
		clientID = "ca-pub-XXXXXXXXXXXXXXXX"
	}
	return fmt.Sprintf(`<!-- AdSense Matched Content -->
<ins class="adsbygoogle"
     style="display:block"
     data-ad-format="autorelaxed"
     data-ad-client="%s"
     data-ad-slot="XXXXXXXXXX"></ins>
<script>(adsbygoogle = window.adsbygoogle || []).push({});</script>`, clientID)
}

func amazonAffiliateLinks() []string {
	tag := "emeraldeng0e-20"
	return []string{
		fmt.Sprintf(`<a href="https://www.amazon.com/dp/B0BXYZ?tag=%s" target="_blank" rel="nofollow sponsored">Amazon Product #1</a>`, tag),
		fmt.Sprintf(`<a href="https://www.amazon.com/dp/B0CABC?tag=%s" target="_blank" rel="nofollow sponsored">Amazon Product #2</a>`, tag),
		fmt.Sprintf(`<a href="https://www.amazon.com/dp/B0DDEF?tag=%s" target="_blank" rel="nofollow sponsored">Amazon Product #3</a>`, tag),
		fmt.Sprintf(`<img src="https://ir-na.amazon-adsystem.com/e/ir?t=%s&l=am2&o=1" width="1" height="1" border="0" alt="" style="border:none !important; margin:0px !important;" />`, tag),
	}
}

func amazonNativeAd() string {
	tag := "emeraldeng0e-20"
	return fmt.Sprintf(`<!-- Amazon Native Ad -->
<iframe src="https://rcm-na.amazon-adsystem.com/e/cm?o=1&p=48&l=ur1&category=amzn_main&banner=0P3Y7BVGKXBQ5CXKNW02&f=ifr&linkID=amzn-main&t=%s" width="728" height="90" scrolling="no" border="0" marginwidth="0" style="border:none;" frameborder="0"></iframe>`, tag)
}

func clickbankLink() string {
	return `<a href="https://HOPLINK.hop.clickbank.net" target="_blank" rel="nofollow sponsored">ClickBank Offer</a>`
}

func sharesaleLink() string {
	return `<a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" target="_blank" rel="nofollow sponsored">ShareASale Deal</a>`
}

func digistoreLink() string {
	return `<a href="https://www.digistore24.com/redir/PRODUCT_ID" target="_blank" rel="nofollow sponsored">DigiStore24 Product</a>`
}

func jvzooLink() string {
	return `<a href="https://www.jvzoo.com/buy/PRODUCT_ID" target="_blank" rel="nofollow sponsored">JVZoo Offer</a>`
}

func warriorplusLink() string {
	return `<a href="https://warriorplus.com/buy/PRODUCT_ID" target="_blank" rel="nofollow sponsored">WarriorPlus Deal</a>`
}

func stripeBuyButton(productName string, price string) string {
	return fmt.Sprintf(`<!-- Stripe Buy Button -->
<form action="https://buy.stripe.com/TEST_SESSION_ID" method="GET" target="_blank">
  <button type="submit" style="background: #6772E5; color: white; padding: 14px 40px; border: none; border-radius: 50px; font-size: 1.1em; font-weight: bold; cursor: pointer; transition: transform 0.2s;">
    Buy %s - $%s
  </button>
</form>`, productName, price)
}

func cryptoDonationBTC() string {
	return `<div class="crypto-donation">
  <h3>Support Us with Crypto</h3>
  <p><strong>Bitcoin (BTC):</strong> <code>bc1qxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx</code></p>
  <p><strong>Ethereum (ETH):</strong> <code>0xXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX</code></p>
  <p><strong>USDT (ERC-20):</strong> <code>0xYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY</code></p>
</div>`
}

func emailCaptureForm() string {
	return `<div class="email-capture">
  <h3>Get Exclusive Updates & Free Resources</h3>
  <form action="https://YOUR_MAILCHIMP_URL.us18.list-manage.com/subscribe/post?u=USER_ID&amp;id=LIST_ID" method="post" target="_blank">
    <input type="email" name="EMAIL" placeholder="Enter your email" required style="padding:14px 20px;border:2px solid #ddd;border-radius:50px;width:280px;font-size:1em;">
    <button type="submit" style="background:#667eea;color:white;padding:14px 30px;border:none;border-radius:50px;font-size:1em;font-weight:bold;cursor:pointer;margin-left:10px;">Subscribe Free</button>
  </form>
</div>`
}

func popunderAd() string {
	return `<!-- Popunder Ad (PropellerAds) -->
<script type="text/javascript" src="https://PROPELLER_ADS_ZONE.script" data-cfasync="false"></script>`
}

func outbrainWidget() string {
	return `<!-- Outbrain / Taboola Native Widget -->
<div class="OUTBRAIN" data-src="https://emerald-engine.com" data-widget-id="AR_1"></div>
<script type="text/javascript" async="async" src="https://widgets.outbrain.com/outbrain.js"></script>`
}

// ─── Page Templates ───

func commonHead(title, desc string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <meta name="description" content="%s">
    <meta name="robots" content="index, follow">
    <link rel="canonical" href="%s">
    %s
    <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; min-height: 100vh; display: flex; flex-direction: column; align-items: center; }
    .container { max-width: 1200px; width: 100%%; padding: 20px; }
    .ad-container { background: rgba(0,0,0,0.03); padding: 16px; border-radius: 10px; text-align: center; margin: 20px 0; min-height: 90px; display: flex; align-items: center; justify-content: center; }
    .features { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 30px; margin: 40px 0; }
    footer { color: #999; padding: 40px 0; text-align: center; width: 100%%; font-size: 0.9em; border-top: 1px solid #eee; margin-top: 60px; }
    a { color: #667eea; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .btn { display: inline-block; padding: 14px 40px; border: none; border-radius: 50px; font-size: 1.1em; font-weight: bold; cursor: pointer; transition: all 0.3s; text-decoration: none; }
    .btn-primary { background: #667eea; color: white; }
    .btn-primary:hover { transform: scale(1.05); text-decoration: none; }
    .btn-success { background: #2ecc71; color: white; }
    .btn-warning { background: #f39c12; color: white; }
    .btn-danger { background: #e74c3c; color: white; }
    .email-capture { background: linear-gradient(135deg, #667eea, #764ba2); color: white; padding: 40px; border-radius: 15px; text-align: center; margin: 40px 0; }
    .email-capture h3 { margin-bottom: 20px; font-size: 1.5em; }
    .email-capture input[type="email"] { padding: 14px 20px; border: none; border-radius: 50px 0 0 50px; width: 300px; font-size: 1em; }
    .email-capture button { padding: 14px 30px; border: none; border-radius: 0 50px 50px 0; font-size: 1em; font-weight: bold; cursor: pointer; background: #ff6b6b; color: white; }
    .crypto-donation { background: #f8f9fa; padding: 30px; border-radius: 15px; margin: 40px 0; text-align: center; }
    .crypto-donation code { background: #e9ecef; padding: 4px 8px; border-radius: 4px; font-size: 0.85em; word-break: break-all; }
    .affiliate-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin: 30px 0; }
    .affiliate-card { border: 1px solid #eee; padding: 25px; border-radius: 12px; transition: box-shadow 0.3s; }
    .affiliate-card:hover { box-shadow: 0 5px 20px rgba(0,0,0,0.1); }
    .affiliate-card h4 { margin-bottom: 10px; }
    .affiliate-card .price { font-size: 1.3em; color: #2ecc71; font-weight: bold; }
    .affiliate-card .btn { margin-top: 15px; }
    .review-box { background: white; border: 1px solid #e0e0e0; border-radius: 16px; padding: 35px; margin: 25px 0; box-shadow: 0 4px 15px rgba(0,0,0,0.05); }
    .review-box h3 { color: #333; margin-bottom: 15px; }
    .rating { color: #f39c12; font-size: 1.3em; }
    .pros-cons { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin: 20px 0; }
    .pros { background: #e8f8f5; padding: 20px; border-radius: 10px; }
    .cons { background: #fdf2f2; padding: 20px; border-radius: 10px; }
    .badge { display: inline-block; background: #667eea; color: white; padding: 4px 14px; border-radius: 20px; font-size: 0.8em; margin-bottom: 12px; }
    @media (max-width: 768px) {
      .email-capture input[type="email"] { width: 100%%; border-radius: 50px; margin-bottom: 10px; }
      .email-capture button { width: 100%%; border-radius: 50px; }
      .pros-cons { grid-template-columns: 1fr; }
    }
    </style>
</head>
<body>
<div class="container">`,
		title, desc, baseURL, googleAnalytics())
}

func commonFooter() string {
	return fmt.Sprintf(`</div>
<footer>
  <p>&copy; %d Emerald Engine. All rights reserved.</p>
  <p style="margin-top:8px;font-size:0.8em;opacity:0.6;">We participate in affiliate programs. As an Amazon Associate we earn from qualifying purchases.</p>
  <p style="margin-top:4px;"><a href="/">Home</a> &middot; <a href="/compare.html">Compare</a> &middot; <a href="/review.html">Reviews</a> &middot; <a href="/resources.html">Resources</a> &middot; <a href="/blog/">Blog</a></p>
</footer>
</body>
</html>`, time.Now().Year())
}

// ─── Page Generators ───

func generateIndexPage(niche Niche) string {
	scheme := pick(colorSchemes)
	title := fmt.Sprintf("%s - AI-Powered Solutions", niche.Name)
	desc := fmt.Sprintf("Discover cutting-edge %s solutions. Boost your results with AI-driven tools and strategies.", niche.Name)

	features := []string{
		"AI-Powered Automation",
		"Real-Time Analytics Dashboard",
		"Smart Campaign Optimization",
	}
	descs := []string{
		"Automate repetitive tasks and focus on what matters most. Our AI handles the heavy lifting.",
		"Track every metric that matters with beautiful, actionable dashboards powered by machine learning.",
		"Continuous A/B testing and optimization across all channels with zero manual effort.",
	}

	featureHTML := ""
	for i := 0; i < 3; i++ {
		featureHTML += fmt.Sprintf(`
<div class="feature-card" style="background:white;padding:35px;border-radius:15px;box-shadow:0 8px 25px rgba(0,0,0,0.08);">
  <h3 style="color:#333;margin-bottom:12px;font-size:1.4em;">%s</h3>
  <p style="color:#666;line-height:1.7;">%s</p>
</div>`, features[i], descs[i])
	}

	affiliateLinks := amazonAffiliateLinks()

	return fmt.Sprintf(`%s
<header style="text-align:center;padding:60px 0 30px;background:linear-gradient(135deg,%s 0%%,%s 100%%);border-radius:0 0 30px 30px;color:white;margin:-20px -20px 0;">
  <div class="badge" style="background:rgba(255,255,255,0.2);">%s %s</div>
  <h1 style="font-size:3em;margin-bottom:15px;">%s</h1>
  <p style="font-size:1.2em;opacity:0.9;max-width:700px;margin:0 auto;">%s</p>
</header>

<div class="ad-container">%s</div>

<div class="features">%s</div>

<div style="text-align:center;margin:40px 0;">
  <a href="#" class="btn btn-primary" style="background:%s;">Get Started Free</a>
</div>

<div class="ad-container">%s</div>

<div class="email-capture">
  <h3>Get Exclusive %s Tips & Resources</h3>
  <form action="https://YOUR_MAILCHIMP_URL.us18.list-manage.com/subscribe/post?u=USER_ID&amp;id=LIST_ID" method="post" target="_blank">
    <input type="email" name="EMAIL" placeholder="Enter your email" required>
    <button type="submit">Subscribe Free</button>
  </form>
</div>

<div class="affiliate-grid">
  <div class="affiliate-card"><h4>Recommended Tool #1</h4><p>Top-rated solution for professionals</p><div class="price">$29/mo</div>%s</div>
  <div class="affiliate-card"><h4>Recommended Tool #2</h4><p>Trusted by 10,000+ businesses</p><div class="price">$49/mo</div>%s</div>
  <div class="affiliate-card"><h4>Recommended Tool #3</h4><p>Best value for teams</p><div class="price">$99/mo</div>%s</div>
</div>

<div class="ad-container">%s</div>
%s
%s`, commonHead(title, desc),
		scheme[0], scheme[1],
		niche.Emoji, niche.Name,
		title, desc,
		adsenseBanner(""),
		featureHTML,
		scheme[2],
		adsenseInArticle(""),
		niche.Name,
		affiliateLinks[0], affiliateLinks[1], affiliateLinks[2],
		adsenseMatchedContent(""),
		cryptoDonationBTC(),
		commonFooter())
}

func generateComparePage() string {
	title := "Best AI Marketing Tools Compared 2026 - Side by Side Review"
	desc := "Compare the top AI marketing platforms. Detailed feature comparison, pricing, and real user reviews to help you choose the best tool."

	products := []struct {
		Name    string
		Rating  string
		Price   string
		Pros    []string
		Cons    []string
		Upsell  string
		AffTag  string
		Link    string
		Stripe  string
	}{
		{
			"ToolAlpha Pro", "★★★★★", "$29/mo",
			[]string{"Best AI automation", "Intuitive interface", "24/7 support"},
			[]string{"Limited integrations", "No mobile app"},
			"AI Automation Suite", "amzn", "https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20", "$29",
		},
		{
			"MarketGenius 360", "★★★★☆", "$49/mo",
			[]string{"Advanced analytics", "Multi-channel", "API access"},
			[]string{"Steep learning curve", "Higher price"},
			"Analytics Mastery Pack", "cb", "https://HOPLINK.hop.clickbank.net", "$49",
		},
		{
			"ConvertFlow AI", "★★★★★", "$19/mo",
			[]string{"Best value", "Easy setup", "Great templates"},
			[]string{"Fewer features", "Basic reporting"},
			"Conversion Booster Kit", "ss", "https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID", "$19",
		},
	}

	rows := ""
	for _, p := range products {
		pros := strings.Join(p.Pros, "; ")
		cons := strings.Join(p.Cons, "; ")
		rows += fmt.Sprintf(`
<div class="review-box">
  <div class="rating">%s</div>
  <h3>%s</h3>
  <div class="price" style="font-size:1.5em;color:#2ecc71;font-weight:bold;margin:10px 0;">%s</div>
  <div class="pros-cons">
    <div class="pros"><strong>✓ Pros:</strong><br>%s</div>
    <div class="cons"><strong>✗ Cons:</strong><br>%s</div>
  </div>
  <div style="display:flex;gap:15px;flex-wrap:wrap;margin-top:15px;">
    <a href="%s" class="btn btn-primary" target="_blank" rel="nofollow sponsored">Buy via Affiliate →</a>
    %s
  </div>
  <p style="margin-top:12px;font-size:0.85em;color:#999;">⭐ Bonus: <strong>%s</strong> included with purchase</p>
</div>`, p.Rating, p.Name, p.Price, pros, cons,
			p.Link,
			stripeBuyButton(p.Name, p.Stripe),
			p.Upsell)
	}

	return fmt.Sprintf(`%s
<header style="text-align:center;padding:50px 0 30px;">
  <h1 style="font-size:2.5em;color:#333;margin-bottom:12px;">%s</h1>
  <p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto;">%s</p>
</header>

<div class="ad-container">%s</div>

%s

<div class="ad-container">%s</div>

%s

<div class="ad-container">%s</div>

<p style="text-align:center;color:#999;margin:20px 0;">We compare prices and features as of %s. Prices may vary. Affiliate links may earn us a commission.</p>

%s
%s`, commonHead(title, desc),
		title, desc,
		adsenseBanner(""),
		rows,
		adsenseInArticle(""),
		emailCaptureForm(),
		adsenseSidebar(""),
		time.Now().Format("January 2006"),
		amazonNativeAd(),
		commonFooter())
}

func generateReviewPage() string {
	title := "Comprehensive Review: Top AI Marketing Platform 2026"
	desc := "In-depth review of the leading AI marketing platform. Features, pricing, pros & cons, and real performance data. Read before you buy."

	reviewTitle := pick([]string{
		"AI Marketing Hub Pro",
		"SmartCampaign 360",
		"ConvertBot Elite",
		"MarketForge AI",
		"GrowthEngine X",
	})
	rating := pick([]string{"★★★★★", "★★★★☆", "★★★★★"})
	pros := pickN([]string{
		"Incredible AI-powered automation saves 20+ hours/week",
		"Beautiful, intuitive dashboard",
		"Excellent customer support (24/7 live chat)",
		"Seamless integration with 500+ tools",
		"Real-time performance tracking",
		"Smart A/B testing built-in",
		"No coding required",
		"Regular feature updates",
		"Great value for enterprise features",
		"30-day money-back guarantee",
	}, 5)
	cons := pickN([]string{
		"Monthly subscription can be pricey for small teams",
		"Advanced features have a learning curve",
		"Some integrations still in beta",
		"Mobile app could be better",
		"Limited customization on basic plan",
	}, 3)

	return fmt.Sprintf(`%s
<header style="text-align:center;padding:50px 0 20px;">
  <span class="badge">⭐ Editor's Pick 2026</span>
  <h1 style="font-size:2.5em;color:#333;margin:10px 0;">%s Review</h1>
  <div class="rating" style="font-size:2em;">%s</div>
  <p style="color:#666;margin-top:10px;">Our in-depth analysis after 90 days of testing</p>
</header>

<div class="ad-container">%s</div>

<div class="review-box">
  <h2>What Is %s?</h2>
  <p style="line-height:1.8;color:#444;">%s is a comprehensive AI-powered marketing platform designed to automate and optimize your entire marketing funnel. From lead generation to conversion tracking, it uses machine learning to continuously improve campaign performance across all channels.</p>
  <p style="line-height:1.8;color:#444;margin-top:12px;">After testing %s extensively for 90 days across multiple campaigns, we can confidently say it's one of the most powerful tools available for serious marketers.</p>
</div>

<div class="ad-container">%s</div>

<div class="review-box">
  <h2>Pros & Cons</h2>
  <div class="pros-cons">
    <div class="pros"><strong>✅ Pros</strong><br>%s</div>
    <div class="cons"><strong>❌ Cons</strong><br>%s</div>
  </div>
</div>

<div style="text-align:center;margin:30px 0;">
  <a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-primary" target="_blank" rel="nofollow sponsored" style="font-size:1.2em;padding:18px 60px;">Check Price on Amazon →</a>
  <p style="margin-top:10px;color:#999;font-size:0.85em;">Prices start at $29/mo</p>
</div>

<div class="ad-container">%s</div>

%s

<div class="review-box">
  <h2>Key Features</h2>
  <div class="features">
    <div class="feature-card" style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06);">
      <h3 style="color:#333;margin-bottom:8px;">🤖 AI Automation</h3>
      <p style="color:#666;">Automate repetitive tasks, smart scheduling, and predictive campaign adjustments.</p>
    </div>
    <div class="feature-card" style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06);">
      <h3 style="color:#333;margin-bottom:8px;">📊 Analytics</h3>
      <p style="color:#666;">Real-time dashboards with AI-powered insights and recommendations.</p>
    </div>
    <div class="feature-card" style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06);">
      <h3 style="color:#333;margin-bottom:8px;">🔗 Integrations</h3>
      <p style="color:#666;">Connect with 500+ tools including CRM, email, social, and analytics platforms.</p>
    </div>
  </div>
</div>

<div class="ad-container">%s</div>

%s

<div style="text-align:center;margin:40px 0;">
  <h3 style="margin-bottom:20px;">Ready to Try %s?</h3>
  <div style="display:flex;gap:15px;justify-content:center;flex-wrap:wrap;">
    <a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-primary" target="_blank" rel="nofollow sponsored">Buy on Amazon</a>
    <a href="https://HOPLINK.hop.clickbank.net" class="btn btn-success" target="_blank" rel="nofollow sponsored">Official Website</a>
    <a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" class="btn btn-warning" target="_blank" rel="nofollow sponsored">Special Deal</a>
  </div>
</div>

%s
%s`, commonHead(title, desc),
		reviewTitle, rating,
		adsenseBanner(""),
		reviewTitle, reviewTitle, reviewTitle,
		adsenseInArticle(""),
		strings.Join(pros, "<br>"), strings.Join(cons, "<br>"),
		adsenseMatchedContent(""),
		emailCaptureForm(),
		adsenseSidebar(""),
		cryptoDonationBTC(),
		reviewTitle,
		amazonNativeAd(),
		commonFooter())
}

func generateResourcesPage() string {
	title := "Ultimate Marketing Resource Directory 2026 - Curated Tools & Guides"
	desc := "Hand-picked collection of the best marketing tools, resources, and guides. Save time and money with our curated recommendations."

	resources := []struct {
		Name string
		Desc string
		Link string
	}{
		{"AI Marketing Platform", "All-in-one AI-powered marketing automation suite", "https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20"},
		{"Email Marketing Tool", "Best email automation platform for campaigns", "https://HOPLINK.hop.clickbank.net"},
		{"SEO Analyzer", "Comprehensive SEO audit and optimization tool", "https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID"},
		{"Social Media Manager", "Schedule, analyze, and optimize social content", "https://www.digistore24.com/redir/PRODUCT_ID"},
		{"Analytics Dashboard", "Real-time marketing analytics and reporting", "https://www.jvzoo.com/buy/PRODUCT_ID"},
		{"Landing Page Builder", "Create high-converting landing pages easily", "https://warriorplus.com/buy/PRODUCT_ID"},
	}

	resourceCards := ""
	for _, r := range resources {
		resourceCards += fmt.Sprintf(`
<div class="affiliate-card">
  <h4>%s</h4>
  <p>%s</p>
  <a href="%s" class="btn btn-primary" target="_blank" rel="nofollow sponsored" style="font-size:0.9em;padding:10px 24px;">Learn More →</a>
</div>`, r.Name, r.Desc, r.Link)
	}

	return fmt.Sprintf(`%s
<header style="text-align:center;padding:50px 0 30px;">
  <h1 style="font-size:2.5em;color:#333;margin-bottom:12px;">%s</h1>
  <p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto;">%s</p>
</header>

<div class="ad-container">%s</div>

<p style="margin:20px 0 10px;font-size:0.9em;color:#999;">Disclosure: Some links on this page are affiliate links. We may earn a commission at no extra cost to you.</p>

<div class="affiliate-grid">%s</div>

<div class="ad-container">%s</div>

%s

<div class="ad-container">%s</div>

<div class="ad-container">%s</div>

%s
%s`, commonHead(title, desc),
		title, desc,
		adsenseBanner(""),
		resourceCards,
		adsenseInArticle(""),
		emailCaptureForm(),
		adsenseSidebar(""),
		adsenseMatchedContent(""),
		cryptoDonationBTC(),
		commonFooter())
}

func generateBlogPost() string {
	niche := pick(niches)
	title := fmt.Sprintf("How to %s in 2026 - Complete Guide", pick([]string{
		"Boost Your Marketing ROI with AI",
		"Generate More Leads Automatically",
		"Double Your Conversion Rate",
		"Build a Profitable Online Business",
		fmt.Sprintf("Master %s Automation", niche.Name),
		fmt.Sprintf("Scale Your %s Strategy", niche.Name),
	}))
	desc := fmt.Sprintf("Complete step-by-step guide to %s. Learn proven strategies used by top marketers. Includes expert tips, tools, and actionable checklists.", title)

	headings := []string{
		"Why Traditional Marketing Is Failing You",
		"The AI Revolution in Digital Marketing",
		"Step 1: Automate Your Lead Generation",
		"Step 2: Optimize Your Conversion Funnel",
		"Step 3: Scale with Smart Analytics",
		"Step 4: Continuous A/B Testing",
		"Real Results from Our Testing",
		"Getting Started Today",
	}
	paragraphs := []string{
		"In today's fast-paced digital landscape, traditional marketing methods are no longer enough. With consumers being bombarded by thousands of messages daily, you need intelligent automation to cut through the noise and reach your target audience effectively.",
		fmt.Sprintf("Artificial intelligence has transformed how we approach marketing. According to recent studies, businesses using AI-powered marketing tools see an average of 40%% increase in efficiency and 25%% higher conversion rates. The %s industry has been particularly disrupted.", niche.Name),
		"The first step to marketing automation is setting up intelligent lead capture systems. Modern AI tools can identify high-intent visitors, score leads automatically, and trigger personalized follow-up sequences without any manual intervention.",
		"Once you have leads flowing in, your conversion funnel needs to be optimized for maximum results. AI-powered analytics can identify exactly where prospects drop off and automatically adjust your messaging, timing, and offers to improve conversion rates.",
		"Scaling your marketing efforts requires sophisticated analytics. Today's AI platforms can process millions of data points in real-time, identifying patterns and opportunities that human analysts would miss. This allows for rapid scaling without proportional increases in cost or effort.",
		"A/B testing has been revolutionized by AI. Instead of manually testing one variable at a time, modern platforms can run multivariate tests across hundreds of combinations simultaneously, automatically allocating traffic to winning variations and maximizing your ROI.",
		"We tested these strategies across multiple campaigns in various niches. The results were consistent: automated AI-powered campaigns outperformed manual campaigns by an average of 3.2x in conversion rate and 4.7x in revenue per visitor.",
		"Ready to transform your marketing? Start with a comprehensive AI marketing platform that integrates all these capabilities. The best part is that most platforms offer free trials, so you can test drive before committing.",
	}

	content := ""
	for i := 0; i < len(headings); i++ {
		content += fmt.Sprintf(`
<h2 style="margin:35px 0 15px;color:#333;">%s</h2>
<p style="line-height:1.8;color:#444;margin-bottom:15px;">%s</p>`, headings[i], paragraphs[i])
		if i == 2 || i == 5 {
			content += fmt.Sprintf(`<div class="ad-container">%s</div>`, adsenseInArticle(""))
		}
	}

	return fmt.Sprintf(`%s
<article>
<header style="text-align:center;padding:40px 0 20px;">
  <div class="badge">📝 Blog Post</div>
  <h1 style="font-size:2.2em;color:#333;margin:10px 0;line-height:1.3;">%s</h1>
  <p style="color:#999;font-size:0.9em;">Published: %s &middot; Category: %s &middot; Reading time: 8 min</p>
</header>

<div class="ad-container">%s</div>

%s

<div class="ad-container">%s</div>

%s

<div class="ad-container">%s</div>

<div style="background:#f8f9fa;padding:40px;border-radius:15px;margin:40px 0;text-align:center;">
  <h3 style="margin-bottom:15px;">🚀 Ready to Get Started?</h3>
  <p style="margin-bottom:20px;color:#666;">Try the same tools we use and recommend. Start your free trial today.</p>
  <div style="display:flex;gap:15px;justify-content:center;flex-wrap:wrap;">
    <a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-primary" target="_blank" rel="nofollow sponsored">Shop on Amazon</a>
    <a href="https://HOPLINK.hop.clickbank.net" class="btn btn-success" target="_blank" rel="nofollow sponsored">ClickBank Deal</a>
    <a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" class="btn btn-warning" target="_blank" rel="nofollow sponsored">ShareASale Offers</a>
  </div>
</div>

%s
</article>
%s`, commonHead(title, desc),
		title,
		time.Now().Format("January 2, 2006"),
		niche.Name,
		adsenseBanner(""),
		content,
		adsenseSidebar(""),
		emailCaptureForm(),
		adsenseMatchedContent(""),
		outbrainWidget(),
		commonFooter())
}

// ─── Engine ───

func ensureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

func generateSitemap(pages []string) string {
	urls := ""
	for _, p := range pages {
		urls += fmt.Sprintf(`  <url><loc>%s/%s</loc><changefreq>daily</changefreq><priority>0.8</priority></url>`, baseURL, p)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
%s
</urlset>`, urls)
}

func generateRobots() string {
	return fmt.Sprintf(`User-agent: *
Allow: /
Sitemap: %s/sitemap.xml

# Disallow admin paths
Disallow: /admin/
Disallow: /private/
`, baseURL)
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitCommitAndPush(outputFiles []string) error {
	args := append([]string{"add"}, outputFiles...)
	if err := runCmd("git", args...); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	ts := time.Now().Format(time.RFC3339)
	msg := fmt.Sprintf("auto: generate monetized pages [%s]", ts)
	if err := runCmd("git", "commit", "--allow-empty", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	remotes := []string{"origin", "huggingface"}
	for _, r := range remotes {
		if err := runCmd("git", "push", r, "main"); err != nil {
			fmt.Fprintf(os.Stderr, "[ENGINE] push to %s failed: %v\n", r, err)
		} else {
			fmt.Printf("[ENGINE] pushed to %s\n", r)
		}
	}
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("[ENGINE] Starting Emerald Engine v3.0 (Multi-Page Monetization)")
	fmt.Printf("[ENGINE] Cycle interval: %ds\n", cycleDelay)

	outputFiles := []string{
		"public/index.html",
		"public/compare.html",
		"public/review.html",
		"public/resources.html",
		"public/blog/getting-started.html",
		"public/sitemap.xml",
		"public/robots.txt",
	}

	for i := 1; ; i++ {
		fmt.Printf("\n[ENGINE] === Cycle %d ===\n", i)
		start := time.Now()
		totalBytes := 0

		niche := pick(niches)
		fmt.Printf("[ENGINE] Niche: %s %s\n", niche.Emoji, niche.Name)

		pages := map[string]func() string{
			"public/index.html":                func() string { return generateIndexPage(niche) },
			"public/compare.html":              generateComparePage,
			"public/review.html":               generateReviewPage,
			"public/resources.html":            generateResourcesPage,
			"public/blog/getting-started.html": generateBlogPost,
		}

		for path, gen := range pages {
			if err := ensureDir(path); err != nil {
				fmt.Fprintf(os.Stderr, "[ENGINE] mkdir %s: %v\n", path, err)
				continue
			}
			html := gen()
			if err := os.WriteFile(path, []byte(html), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "[ENGINE] write %s: %v\n", path, err)
				continue
			}
			totalBytes += len(html)
			fmt.Printf("[ENGINE] ✓ %s (%d bytes)\n", path, len(html))
		}

		allPages := []string{
			"index.html", "compare.html", "review.html",
			"resources.html", "blog/getting-started.html",
		}
		sitemap := generateSitemap(allPages)
		os.WriteFile("public/sitemap.xml", []byte(sitemap), 0644)
		totalBytes += len(sitemap)
		fmt.Printf("[ENGINE] ✓ sitemap.xml (%d bytes)\n", len(sitemap))

		robots := generateRobots()
		os.WriteFile("public/robots.txt", []byte(robots), 0644)
		totalBytes += len(robots)
		fmt.Printf("[ENGINE] ✓ robots.txt (%d bytes)\n", len(robots))

		fmt.Printf("[ENGINE] Total: %d bytes across %d files\n", totalBytes, len(outputFiles))

		if err := gitCommitAndPush(outputFiles); err != nil {
			fmt.Fprintf(os.Stderr, "[ENGINE] git error: %v\n", err)
		}

		elapsed := time.Since(start)
		sleep := cycleDelay*time.Second - elapsed
		if sleep > 0 {
			fmt.Printf("[ENGINE] Cycle done in %v. Next in %v\n", elapsed, sleep)
			time.Sleep(sleep)
		} else {
			fmt.Printf("[ENGINE] Cycle took %v (exceeded interval)\n", elapsed)
		}
	}
}
