package main

import (
	"fmt"
	"strings"
)

type ChildData struct {
	Name          string
	Keyword       string
	Emoji         string
	Headline      string
	Subhead       string
	Primary       string
	Secondary     string
	Accent        string
	BgColor       string
	DarkPrimary   string
	Features      string
	CTAs          string
	FeatureTitles []string
	Slug          string
	Year          string
	VariationIdx  int
}

func generateChildCode(niche Niche) string {
	d := buildChildData(niche)

	return fmt.Sprintf(`package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var startTime = time.Now()

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/compare", compareHandler)
	http.HandleFunc("/review", reviewHandler)
	http.HandleFunc("/blog", blogHandler)
	http.HandleFunc("/health", healthHandler)
	log.Printf("[Child %[1]s] Starting on :%%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "niche": "%[1]s", "uptime": time.Since(startTime).String()})
}

func gaTag() string {
	id := os.Getenv("GA_ID")
	if id == "" || id == "G-XXXXXXXXXX" { return "" }
	return "<script async src=\"https://www.googletagmanager.com/gtag/js?id=" + id + "\"></script><script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config','" + id + "');</script>"
}

func adTag() string {
	id := os.Getenv("ADSENSE_CLIENT_ID")
	if id == "" || id == "ca-pub-XXXXXXXXXXXXXXXX" { return "" }
	return "<script async src=\"https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=" + id + "\" crossorigin=\"anonymous\"></script>"
}

func amzBlock() string {
	tag := os.Getenv("AMAZON_ASSOCIATES_TAG")
	if tag == "" || tag == "emeraldeng0e-20" { return "" }
	c1 := "<div class=\"amazon-card\"><h3>Top Rated %[2]s Tool</h3><p>Best seller</p><a class=\"cta-btn cta-1\" href=\"https://www.amazon.com/dp/B08N5WRWNW?tag=" + tag + "\" target=\"_blank\" rel=\"nofollow\">Check Price</a></div>"
	c2 := "<div class=\"amazon-card\"><h3>Best Value %[2]s Kit</h3><p>Top rated</p><a class=\"cta-btn cta-2\" href=\"https://www.amazon.com/dp/B08N5WRWNW?tag=" + tag + "\" target=\"_blank\" rel=\"nofollow\">View on Amazon</a></div>"
	return "<section class=\"amazon-section\"><h2>Recommended on Amazon</h2><div class=\"amazon-grid\">" + c1 + c2 + "</div></section>"
}

func replacePage(page string) string {
	page = strings.ReplaceAll(page, "{GA}", gaTag())
	page = strings.ReplaceAll(page, "{AD}", adTag())
	page = strings.ReplaceAll(page, "{AMZ}", amzBlock())
	return page
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(replacePage(indexPage)))
}

func compareHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(comparePage))
}

func reviewHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(reviewPage))
}

func blogHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(blogPage))
}

var indexPage = `+"`"+`%s`+"`"+`
var comparePage = `+"`"+`%s`+"`"+`
var reviewPage = `+"`"+`%s`+"`"+`
var blogPage = `+"`"+`%s`+"`"+`
`,
		d.Keyword, d.Name,
		generateChildIndexHTML(d),
		generateChildCompareHTML(d),
		generateChildReviewHTML(d),
		generateChildBlogHTML(d),
	)
}

func generateChildIndexHTML(d ChildData) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + d.Name + ` — Expert Reviews & Comparisons 2026</title>
<meta name="description" content="` + d.Subhead + `">
<meta property="og:title" content="` + d.Name + ` — Expert Reviews & Comparisons">
<meta property="og:description" content="` + d.Subhead + `">
<meta name="twitter:card" content="summary_large_image">
` + childStructuredData(d, "WebSite") + `
{GA}{AD}
` + generateChildCSS(d) + `
</head>
<body>
<nav class="top-nav"><div class="nav-inner"><a href="/" class="logo">` + d.Emoji + ` <span>` + d.Name + `</span></a><div class="nav-links"><a href="/">Home</a><a href="/compare">Compare</a><a href="/review">Reviews</a><a href="/blog">Blog</a></div></div></nav>
<section class="hero">
  <div class="hero-content">
    <h1>` + d.Headline + `</h1>
    <p>` + d.Subhead + `</p>
    <div class="hero-cta">` + d.CTAs + `</div>
  </div>
</section>
<section class="features">
  <h2>Top ` + d.Slug + ` Picks</h2>
  <div class="feature-grid">` + d.Features + `</div>
</section>
<section class="newsletter">
  <div class="newsletter-inner">
    <h2>Stay Updated</h2>
    <p>Get the latest ` + d.Slug + ` insights delivered weekly.</p>
    <form class="email-form" action="#" method="post"><input type="email" placeholder="Your email address" required><button type="submit">Subscribe</button></form>
  </div>
</section>
{AMZ}
<footer>
  <p>&copy; ` + d.Year + ` Emerald Engine. All rights reserved.</p>
  <p><a href="/">Home</a> | <a href="/compare">Compare</a> | <a href="/review">Reviews</a> | <a href="/blog">Blog</a></p>
  <p><small>Disclosure: Some links are affiliate links. We may earn a commission.</small></p>
</footer>
</body>
</html>`
}

func generateChildCompareHTML(d ChildData) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Compare ` + d.Name + ` Solutions</title>
<meta name="description" content="Compare the best ` + d.Slug + ` solutions side by side.">
` + childStructuredData(d, "Product") + `
` + generateChildCSS(d) + `
</head>
<body>
<nav class="top-nav"><div class="nav-inner"><a href="/" class="logo">` + d.Emoji + ` <span>` + d.Name + `</span></a><div class="nav-links"><a href="/">Home</a><a href="/compare">Compare</a><a href="/review">Reviews</a><a href="/blog">Blog</a></div></div></nav>
<section class="content-page">
  <h1>Compare Top ` + d.Name + ` Solutions</h1>
  <div class="compare-grid">
    <div class="compare-card"><h3>Premium Choice</h3><div class="price">$29/mo</div><ul><li>Full feature access</li><li>Priority support</li><li>` + d.Slug + `-optimized</li></ul><a class="cta-btn" href="#">Get Started</a></div>
    <div class="compare-card featured"><h3>Best Value</h3><div class="price">$14/mo</div><ul><li>Core features</li><li>Standard support</li><li>` + d.Slug + ` insights</li></ul><a class="cta-btn" href="#">Try Now</a></div>
    <div class="compare-card"><h3>Free Tier</h3><div class="price">$0</div><ul><li>Basic access</li><li>Community support</li><li>Limited ` + d.Slug + `</li></ul><a class="cta-btn" href="#">Start Free</a></div>
  </div>
</section>
<footer><p>&copy; ` + d.Year + ` Emerald Engine. All rights reserved.</p></footer>
</body>
</html>`
}

func generateChildReviewHTML(d ChildData) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + d.Name + ` Reviews 2026</title>
<meta name="description" content="Expert reviews of the best ` + d.Slug + ` solutions.">
` + childStructuredData(d, "Review") + `
` + generateChildCSS(d) + `
</head>
<body>
<nav class="top-nav"><div class="nav-inner"><a href="/" class="logo">` + d.Emoji + ` <span>` + d.Name + `</span></a><div class="nav-links"><a href="/">Home</a><a href="/compare">Compare</a><a href="/review">Reviews</a><a href="/blog">Blog</a></div></div></nav>
<section class="content-page">
  <h1>Expert ` + d.Name + ` Reviews</h1>
  <div class="review-list">
    <article class="review-card"><h3>Top ` + d.Name + ` Platform 2026</h3><div class="rating">★★★★☆</div><p>Comprehensive analysis of the leading ` + d.Slug + ` solution this year.</p><a class="cta-btn" href="#">Read Full Review</a></article>
    <article class="review-card"><h3>` + d.Name + ` for Beginners</h3><div class="rating">★★★★★</div><p>Step-by-step guide to getting started with ` + d.Slug + ` tools.</p><a class="cta-btn" href="#">Read Full Review</a></article>
    <article class="review-card"><h3>Advanced ` + d.Name + ` Strategies</h3><div class="rating">★★★★☆</div><p>Deep dive into advanced techniques for ` + d.Slug + ` professionals.</p><a class="cta-btn" href="#">Read Full Review</a></article>
  </div>
</section>
<footer><p>&copy; ` + d.Year + ` Emerald Engine. All rights reserved.</p></footer>
</body>
</html>`
}

func generateChildBlogHTML(d ChildData) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + d.Name + ` Blog & News</title>
<meta name="description" content="Latest ` + d.Slug + ` news, tips, and trends.">
` + childStructuredData(d, "Blog") + `
` + generateChildCSS(d) + `
</head>
<body>
<nav class="top-nav"><div class="nav-inner"><a href="/" class="logo">` + d.Emoji + ` <span>` + d.Name + `</span></a><div class="nav-links"><a href="/">Home</a><a href="/compare">Compare</a><a href="/review">Reviews</a><a href="/blog">Blog</a></div></div></nav>
<section class="content-page">
  <h1>` + d.Name + ` Blog & News</h1>
  <div class="blog-list">
    <article class="blog-card"><h3>2026 ` + d.Name + ` Trends</h3><p class="date">June 20, 2026</p><p>Discover the top ` + d.Slug + ` trends shaping the industry.</p><a class="cta-btn" href="#">Read More</a></article>
    <article class="blog-card"><h3>How to Choose ` + d.Name + `</h3><p class="date">June 18, 2026</p><p>A practical guide to selecting the right ` + d.Slug + ` solutions.</p><a class="cta-btn" href="#">Read More</a></article>
    <article class="blog-card"><h3>` + d.Name + ` Tips & Tricks</h3><p class="date">June 15, 2026</p><p>Expert tips to maximize your ` + d.Slug + ` experience.</p><a class="cta-btn" href="#">Read More</a></article>
  </div>
</section>
<footer><p>&copy; ` + d.Year + ` Emerald Engine. All rights reserved.</p></footer>
</body>
</html>`
}

func nicheFeatures(keyword string) []string {
	features := map[string][]string{
		"saas-marketing":    {"AI Campaign Builder", "Conversion Optimizer", "Audience Insights", "A/B Testing Engine", "ROI Dashboard"},
		"health-wellness":   {"Wellness Tracker", "Meal Planner", "Fitness AI Coach", "Sleep Analyzer", "Meditation Guide"},
		"personal-finance":  {"Smart Budgeting", "Investment Tracker", "Expense AI", "Credit Optimizer", "Retirement Planner"},
		"digital-products":  {"Product Builder", "Delivery Automation", "Analytics Suite", "Customer Portal", "Revenue Tracker"},
		"online-education":  {"Course Creator", "Student Analytics", "Content AI", "Quiz Engine", "Certificate Manager"},
		"ecommerce":         {"Store Builder", "Inventory AI", "Checkout Optimizer", "Customer Insights", "Marketing Hub"},
		"ai-tools":          {"Prompt Studio", "Model Comparator", "Output Analyzer", "Pipeline Builder", "API Manager"},
		"crypto-web3":       {"Portfolio Tracker", "DeFi Scanner", "Gas Optimizer", "NFT Analyzer", "Cross-Chain Bridge"},
		"real-estate":       {"Property Scanner", "Market Analyzer", "Mortgage Calculator", "Neighborhood AI", "ROI Predictor"},
		"travel":            {"Trip Planner", "Price Predictor", "Itinerary Builder", "Review Analyzer", "Rewards Optimizer"},
		"fitness-nutrition": {"Workout Builder", "Nutrition AI", "Progress Tracker", "Form Analyzer", "Meal Prep Guide"},
		"remote-work":       {"Productivity OS", "Team Dashboard", "Focus Timer", "Meeting Optimizer", "Async Hub"},
		"content-creation":  {"AI Writer", "Video Editor", "Audio Enhancer", "Template Studio", "Analytics Suite"},
		"cybersecurity":     {"Threat Scanner", "Password Vault", "Network Monitor", "Privacy Guard", "Breach Checker"},
		"self-improvement":  {"Habit Tracker", "Goal Builder", "Journal AI", "Skill Manager", "Progress OS"},
	}
	if f, ok := features[keyword]; ok {
		return f
	}
	return []string{"Smart Dashboard", "AI Analytics", "Performance Hub", "Optimization Engine", "Growth Tools"}
}

func nicheHeadline(keyword string) string {
	headlines := map[string]string{
		"saas-marketing":    "Transform Your Marketing with AI-Powered SaaS",
		"health-wellness":   "Take Control of Your Health Journey",
		"personal-finance":  "Master Your Money with Smart Tools",
		"digital-products":  "Launch & Scale Your Digital Products",
		"online-education":  "Create Impactful Online Courses",
		"ecommerce":         "Build Your Dream Online Store",
		"ai-tools":          "Harness the Power of Artificial Intelligence",
		"crypto-web3":       "Navigate the Future of Finance",
		"real-estate":       "Find Your Perfect Property Investment",
		"travel":            "Explore the World Smarter",
		"fitness-nutrition": "Transform Your Body & Mind",
		"remote-work":       "Master the Art of Remote Work",
		"content-creation":  "Create Content That Stands Out",
		"cybersecurity":     "Protect Your Digital Life",
		"self-improvement":  "Become the Best Version of You",
	}
	if h, ok := headlines[keyword]; ok {
		return h
	}
	return "Discover Premium Solutions for Your Success"
}

func nicheCTA(keyword string) string {
	ctas := map[string]string{
		"saas-marketing":    "Start Free Trial",
		"health-wellness":   "Begin Your Journey",
		"personal-finance":  "Take Control Now",
		"digital-products":  "Start Creating",
		"online-education":  "Launch Your Course",
		"ecommerce":         "Open Your Store",
		"ai-tools":          "Explore AI Tools",
		"crypto-web3":       "Enter Web3",
		"real-estate":       "Find Properties",
		"travel":            "Plan Your Trip",
		"fitness-nutrition": "Start Training",
		"remote-work":       "Work Smarter",
		"content-creation":  "Create Today",
		"cybersecurity":     "Secure Now",
		"self-improvement":  "Start Growing",
	}
	if c, ok := ctas[keyword]; ok {
		return c
	}
	return "Get Started"
}

func buildChildData(niche Niche) ChildData {
	primary, secondary, accent, bg := nicheColorVariation(niche)
	slug := strings.ReplaceAll(niche.Keyword, "-", " ")
	feats := nicheFeatures(niche.Keyword)
	headline := nicheHeadline(niche.Keyword)
	cta := nicheCTA(niche.Keyword)

	featureCount := 3 + (hashString(niche.Keyword) % 4)
	if featureCount > len(feats) {
		featureCount = len(feats)
	}
	featureCards := make([]string, featureCount)
	featureTitles := make([]string, featureCount)
	for i := 0; i < featureCount; i++ {
		f := feats[i]
		featureTitles[i] = f
		featureCards[i] = fmt.Sprintf(`<div class="feature-card"><h3>%s</h3><p>Discover top-rated %s solutions tailored for your needs.</p></div>`, f, slug)
	}

	ctaCount := 2 + (hashString(niche.Keyword+"cta") % 2)
	ctaBtns := make([]string, ctaCount)
	for i := 0; i < ctaCount; i++ {
		if i == 0 {
			ctaBtns[i] = fmt.Sprintf(`<a href="/compare" class="cta-btn cta-1">%s</a>`, cta)
		} else if i == 1 {
			ctaBtns[i] = `<a href="/review" class="cta-btn cta-2">Read Reviews</a>`
		} else {
			ctaBtns[i] = `<a href="/blog" class="cta-btn cta-2">Learn More</a>`
		}
	}

	subs := []string{
		"Find the best solutions for your %s needs",
		"Expert reviews and comparisons for %s",
		"Your trusted guide to %s in 2026",
	}
	subhead := fmt.Sprintf(subs[hashString(niche.Keyword)%len(subs)], slug)

	return ChildData{
		Name:          niche.Name,
		Keyword:       niche.Keyword,
		Emoji:         niche.Emoji,
		Headline:      headline,
		Subhead:       subhead,
		Primary:       primary,
		Secondary:     secondary,
		Accent:        accent,
		BgColor:       bg,
		DarkPrimary:   darkenColor(primary),
		Features:      strings.Join(featureCards, "\n    "),
		FeatureTitles: featureTitles,
		CTAs:          strings.Join(ctaBtns, "\n      "),
		Slug:          slug,
		Year:          "2026",
		VariationIdx:  hashString(niche.Keyword) % 15,
	}
}

func generateChildCSS(d ChildData) string {
	return fmt.Sprintf(`<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;line-height:1.6;color:#333;background:%[5]s}
.top-nav{background:%[2]s;color:#fff;padding:1rem 0;position:sticky;top:0;z-index:100;box-shadow:0 2px 8px rgba(0,0,0,0.15)}
.nav-inner{max-width:1100px;margin:0 auto;padding:0 1rem;display:flex;justify-content:space-between;align-items:center}
.logo{font-size:1.5rem;font-weight:800;color:#fff;text-decoration:none}
.logo span{color:%[3]s}
.nav-links a{color:rgba(255,255,255,0.85);text-decoration:none;margin-left:1.5rem;font-weight:500;transition:color .2s}
.nav-links a:hover{color:%[3]s}
.hero{background:linear-gradient(135deg,%[2]s 0%%,%[6]s 100%%);color:#fff;padding:5rem 1rem;text-align:center}
.hero-content{max-width:800px;margin:0 auto}
.hero h1{font-size:2.8rem;margin-bottom:1rem}
.hero p{font-size:1.2rem;opacity:0.92;margin-bottom:2rem}
.hero-cta{display:flex;gap:1rem;justify-content:center;flex-wrap:wrap}
.cta-btn{display:inline-block;padding:0.9rem 2rem;border-radius:8px;text-decoration:none;font-weight:700;transition:all .2s}
.cta-1{background:%[3]s;color:%[2]s}
.cta-2{background:#fff;color:%[2]s}
.cta-btn:hover{transform:translateY(-2px);box-shadow:0 4px 12px rgba(0,0,0,0.2)}
.features{padding:4rem 1rem;max-width:1100px;margin:0 auto}
.features h2{text-align:center;font-size:2rem;margin-bottom:2.5rem;color:%[2]s}
.feature-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:1.5rem}
.feature-card{background:#fff;padding:2rem;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08);border-top:4px solid %[3]s;transition:transform .2s}
.feature-card:hover{transform:translateY(-4px)}
.feature-card h3{color:%[2]s;margin-bottom:0.5rem}
.newsletter{background:%[4]s;padding:3rem 1rem;text-align:center}
.newsletter-inner{max-width:600px;margin:0 auto}
.newsletter h2{color:%[2]s;margin-bottom:0.5rem}
.email-form{display:flex;gap:0.5rem;margin-top:1.5rem;justify-content:center;flex-wrap:wrap}
.email-form input{padding:0.8rem 1rem;border:2px solid #ddd;border-radius:8px;font-size:1rem;flex:1;min-width:200px}
.email-form button{padding:0.8rem 2rem;background:%[3]s;color:%[2]s;border:none;border-radius:8px;font-weight:700;cursor:pointer;transition:background .2s}
.email-form button:hover{opacity:0.9}
.content-page{padding:3rem 1rem;max-width:1100px;margin:0 auto}
.content-page h1{font-size:2.2rem;color:%[2]s;margin-bottom:2rem;text-align:center}
.compare-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:1.5rem;margin-top:2rem}
.compare-card{background:#fff;padding:2rem;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08);text-align:center}
.compare-card.featured{border:3px solid %[3]s;transform:scale(1.05)}
.compare-card h3{color:%[2]s;margin-bottom:0.5rem}
.compare-card .price{font-size:2rem;font-weight:800;color:%[3]s;margin:1rem 0}
.compare-card ul{list-style:none;padding:0;margin:1rem 0}
.compare-card li{padding:0.4rem 0;border-bottom:1px solid #f0f0f0}
.compare-card .cta-btn{background:%[3]s;color:%[2]s;display:inline-block;margin-top:1rem}
.review-list{display:grid;gap:1.5rem}
.review-card{background:#fff;padding:2rem;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08)}
.review-card h3{color:%[2]s}
.rating{color:%[3]s;font-size:1.3rem;margin:0.5rem 0}
.blog-list{display:grid;gap:1.5rem}
.blog-card{background:#fff;padding:2rem;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08)}
.blog-card h3{color:%[2]s}
.blog-card .date{color:#999;font-size:0.9rem;margin:0.3rem 0}
footer{background:%[2]s;color:#fff;padding:2rem 1rem;text-align:center;margin-top:3rem}
footer a{color:%[3]s}
@media(max-width:768px){.hero h1{font-size:2rem}.nav-links a{margin-left:0.8rem;font-size:0.9rem}}
</style>`, d.Name, d.Primary, d.Secondary, d.Accent, d.BgColor, d.DarkPrimary)
}

func childStructuredData(d ChildData, pageType string) string {
	types := map[string]string{
		"WebSite": fmt.Sprintf(`{"@context":"https://schema.org","@type":"WebSite","name":"%s - %s","url":"https://%s.emerald.internal","description":"%s"}`, d.Emoji, d.Name, d.Slug, d.Subhead),
		"Product": fmt.Sprintf(`{"@context":"https://schema.org","@type":"ItemList","name":"Compare %s Solutions","numberOfItems":3,"itemListElement":[{"@type":"Product","name":"Premium %s","offers":{"@type":"Offer","price":"29","priceCurrency":"USD"}},{"@type":"Product","name":"Best Value %s","offers":{"@type":"Offer","price":"14","priceCurrency":"USD"}},{"@type":"Product","name":"Free %s","offers":{"@type":"Offer","price":"0","priceCurrency":"USD"}}]}`, d.Name, d.Name, d.Name, d.Name),
		"Review": fmt.Sprintf(`{"@context":"https://schema.org","@type":"Review","itemReviewed":{"@type":"Product","name":"Top %s Platform"},"reviewRating":{"@type":"Rating","ratingValue":"4.5"}}`, d.Name),
		"Blog":   fmt.Sprintf(`{"@context":"https://schema.org","@type":"Blog","name":"%s Blog","description":"Latest %s news and updates"}`, d.Name, d.Name),
	}
	if script, ok := types[pageType]; ok {
		return `<script type="application/ld+json">` + script + `</script>`
	}
	return ""
}

func generateChildDockerfile(niche Niche) string {
	return `FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod child.go ./
RUN go build -o child .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/child .
EXPOSE 8080
CMD ["./child"]
`
}

func generateChildReadme(niche Niche) string {
	return fmt.Sprintf(`---
title: %s %s
emoji: %s
colorFrom: indigo
colorTo: pink
sdk: docker
pinned: false
license: mit
---
`, niche.Emoji, niche.Name, niche.Emoji)
}

func nicheColorVariation(niche Niche) (primary, secondary, accent, bg string) {
	type cs struct{ primary, secondary, accent, bg string }
	styles := []cs{
		{"#1a1a2e", "#e94560", "#16213e", "#f8f9fa"},
		{"#0f3460", "#e94560", "#533483", "#fafafa"},
		{"#2d4059", "#ea5455", "#f07b3f", "#fcfcfc"},
		{"#222831", "#00adb5", "#393e46", "#f5f5f5"},
		{"#112d4e", "#3f72af", "#dbe2ef", "#f9f9f9"},
		{"#2b2b2b", "#ff6b6b", "#c0392b", "#fcfcfc"},
		{"#1b262c", "#bbe1fa", "#3282b8", "#fafafa"},
		{"#273c75", "#e1b12c", "#40739e", "#f8f8f8"},
		{"#2c3e50", "#e74c3c", "#3498db", "#fdfdfd"},
		{"#34495e", "#1abc9c", "#2ecc71", "#fbfbfb"},
		{"#4a235a", "#e74c3c", "#a569bd", "#f9f9f9"},
		{"#0c2461", "#fa983a", "#60a3bc", "#fafafa"},
		{"#3d3d3d", "#e77f67", "#786fa6", "#f5f5f5"},
		{"#218c74", "#f97f51", "#2c3e50", "#fcfcfc"},
		{"#5758bb", "#ffc312", "#d980fa", "#fbfbfb"},
	}
	idx := hashString(niche.Keyword) % len(styles)
	s := styles[idx]
	return s.primary, s.secondary, s.accent, s.bg
}

func darkenColor(hex string) string {
	return hex + "dd"
}

func hashString(s string) int {
	h := 0
	for i := 0; i < len(s); i++ {
		h = h*31 + int(s[i])
	}
	if h < 0 {
		h = -h
	}
	return h
}
