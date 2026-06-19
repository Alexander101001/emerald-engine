package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const cycleDelay = 300

var baseURL = "https://emerald-engine.com"

type Vault map[string]string

var vault Vault

func loadVault() Vault {
	cmd := exec.Command("node", "src/engine/vault.js")
	cmd.Env = append(os.Environ(), "MASTER_KEY="+os.Getenv("MASTER_KEY"))
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[VAULT] decrypt failed: %v\n", err)
		return Vault{}
	}
	var v Vault
	if err := json.Unmarshal(out, &v); err != nil {
		fmt.Fprintf(os.Stderr, "[VAULT] parse failed: %v\n", err)
		return Vault{}
	}
	fmt.Printf("[VAULT] %d tokens loaded\n", len(v))
	return v
}

func vaultGet(key, fallback string) string {
	if v, ok := vault[key]; ok && v != "" {
		return v
	}
	return fallback
}

// ─── Telegram ───

func telegramSend(msg string) {
	botToken := vaultGet("TELEGRAM_BOT_TOKEN", "")
	chatID := vaultGet("TELEGRAM_USER_ID", "")
	if botToken == "" || chatID == "" {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	body, _ := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    msg,
		"parse_mode": "HTML",
	})
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[TELEGRAM] send failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[TELEGRAM] notification sent\n")
}

// ─── LLM Content Generation ───

type LLMProvider struct {
	Name string
	URL  string
	Key  string
	Model string
}

var llmProviders = []LLMProvider{}

func initLLMProviders() {
	llmProviders = []LLMProvider{}
	if k := vaultGet("GROQ_API_KEY", ""); k != "" {
		llmProviders = append(llmProviders, LLMProvider{"Groq", "https://api.groq.com/openai/v1/chat/completions", k, "mixtral-8x7b-32768"})
	}
	if k := vaultGet("TOGETHER_API_KEY", ""); k != "" {
		llmProviders = append(llmProviders, LLMProvider{"Together", "https://api.together.xyz/v1/chat/completions", k, "mistralai/Mixtral-8x7B-Instruct-v0.1"})
	}
	if k := vaultGet("OPENROUTER_API_KEY", ""); k != "" {
		llmProviders = append(llmProviders, LLMProvider{"OpenRouter", "https://openrouter.ai/api/v1/chat/completions", k, "mistralai/mixtral-8x7b-instruct"})
	}
	if k := vaultGet("OPENAI_API_KEY", ""); k != "" {
		llmProviders = append(llmProviders, LLMProvider{"OpenAI", "https://api.openai.com/v1/chat/completions", k, "gpt-4o-mini"})
	}
	if k := vaultGet("CLAUDE_API_KEY", ""); k != "" {
		llmProviders = append(llmProviders, LLMProvider{"Claude", "https://api.anthropic.com/v1/messages", k, "claude-3-5-haiku-20241022"})
	}
}

func llmGenerate(prompt string) string {
	if len(llmProviders) == 0 {
		return ""
	}
	prov := llmProviders[rand.Intn(len(llmProviders))]
	payload := map[string]interface{}{
		"model": prov.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a professional copywriter and marketer. Generate high-quality, SEO-optimized content in plain text without markdown formatting."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.8,
		"max_tokens":  1024,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", prov.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+prov.Key)
	if prov.Name == "Claude" {
		req.Header.Set("x-api-key", prov.Key)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[LLM] %s: %v\n", prov.Name, err)
		return ""
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					fmt.Printf("[LLM] %s generated %d chars\n", prov.Name, len(content))
					return content
				}
			}
		}
	}
	if prov.Name == "Claude" {
		if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
			if block, ok := content[0].(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					return text
				}
			}
		}
	}
	return ""
}

func llmGenerateOrTemplate(prompt string, template string) string {
	result := llmGenerate(prompt)
	if result == "" {
		return template
	}
	return result
}

// ─── Niches ───

type Niche struct {
	Name    string
	Keyword string
	Emoji   string
}

var niches = []Niche{
	{"SaaS Marketing", "saas-marketing", "🚀"},
	{"Health & Wellness", "health-wellness", "💪"},
	{"Personal Finance", "personal-finance", "💰"},
	{"Digital Products", "digital-products", "📦"},
	{"Online Education", "online-education", "🎓"},
	{"E-commerce", "ecommerce", "🛒"},
	{"AI Tools", "ai-tools", "🤖"},
	{"Crypto & Web3", "crypto-web3", "🔗"},
	{"Real Estate", "real-estate", "🏠"},
	{"Travel & Hospitality", "travel", "✈️"},
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
	{"#d4145a", "#3bbdc3", "#ffc312", "#c31432"},
	{"#02aab0", "#00cdac", "#f39c12", "#008080"},
}

func pick[T any](items []T) T { return items[rand.Intn(len(items))] }

func pickN[T any](items []T, n int) []T {
	perm := rand.Perm(len(items))
	result := make([]T, n)
	for i := 0; i < n && i < len(perm); i++ {
		result[i] = items[perm[i]]
	}
	return result
}

func shuffled[T any](items []T) []T {
	result := make([]T, len(items))
	copy(result, items)
	rand.Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	return result
}

// ─── Monetization Snippets ───

func gaTag() string   { return `<!-- GA4 --><script async src="https://www.googletagmanager.com/gtag/js?id=G-XXXXXXXXXX"></script><script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config','G-XXXXXXXXXX');</script>` }

func adBanner() string  { return `<!-- AS --><ins class="adsbygoogle" style="display:block" data-ad-client="ca-pub-XXXXXXXXXXXXXXXX" data-ad-slot="XXXXXXXXXX" data-ad-format="auto" data-full-width-responsive="true"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({});</script>` }
func adInArticle() string { return `<!-- AS --><ins class="adsbygoogle" style="display:block;text-align:center" data-ad-layout="in-article" data-ad-format="fluid" data-ad-client="ca-pub-XXXXXXXXXXXXXXXX" data-ad-slot="XXXXXXXXXX"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({});</script>` }
func adSidebar() string { return `<!-- AS --><ins class="adsbygoogle" style="display:block" data-ad-client="ca-pub-XXXXXXXXXXXXXXXX" data-ad-slot="XXXXXXXXXX" data-ad-format="auto" data-full-width-responsive="true"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({});</script>` }
func adMatched() string { return `<!-- AS --><ins class="adsbygoogle" style="display:block" data-ad-format="autorelaxed" data-ad-client="ca-pub-XXXXXXXXXXXXXXXX" data-ad-slot="XXXXXXXXXX"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({});</script>` }

func amazonLinks() string {
	return `<a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" rel="nofollow sponsored" target="_blank">Amazon #1</a>
<a href="https://www.amazon.com/dp/B0CABC?tag=emeraldeng0e-20" rel="nofollow sponsored" target="_blank">Amazon #2</a>
<a href="https://www.amazon.com/dp/B0DDEF?tag=emeraldeng0e-20" rel="nofollow sponsored" target="_blank">Amazon #3</a>
<img src="https://ir-na.amazon-adsystem.com/e/ir?t=emeraldeng0e-20&l=am2&o=1" width="1" height="1" border="0" alt="" style="border:none!important;margin:0!important"/>`
}

func amazonAd() string { return `<iframe src="https://rcm-na.amazon-adsystem.com/e/cm?o=1&p=48&l=ur1&category=amzn_main&banner=0P3Y7BVGKXBQ5CXKNW02&f=ifr&linkID=amzn-main&t=emeraldeng0e-20" width="728" height="90" scrolling="no" border="0" marginwidth="0" style="border:none" frameborder="0"></iframe>` }

func affiliateLinks() string {
	return `<a href="https://HOPLINK.hop.clickbank.net" rel="nofollow sponsored" target="_blank">ClickBank Offer</a>
<a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" rel="nofollow sponsored" target="_blank">ShareASale Deal</a>
<a href="https://www.digistore24.com/redir/PRODUCT_ID" rel="nofollow sponsored" target="_blank">DigiStore24 Product</a>
<a href="https://www.jvzoo.com/buy/PRODUCT_ID" rel="nofollow sponsored" target="_blank">JVZoo Offer</a>
<a href="https://warriorplus.com/buy/PRODUCT_ID" rel="nofollow sponsored" target="_blank">WarriorPlus Deal</a>`
}

func stripeBtn(product, price string) string {
	return fmt.Sprintf(`<form action="https://buy.stripe.com/TEST_SESSION_ID" method="GET" target="_blank"><button type="submit" style="background:#6772E5;color:white;padding:14px 40px;border:none;border-radius:50px;font-size:1.1em;font-weight:bold;cursor:pointer">Buy %s - $%s</button></form>`, product, price)
}

func cryptoDonate() string {
	return `<div class="cd"><h3>Support with Crypto</h3><p><strong>BTC:</strong> <code>bc1qxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx</code></p><p><strong>ETH:</strong> <code>0xXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX</code></p><p><strong>USDT:</strong> <code>0xYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY</code></p></div>`
}

func emailForm() string {
	return `<div class="ec"><h3>Get Exclusive Updates & Free Resources</h3><form action="https://YOUR_MAILCHIMP_URL.us18.list-manage.com/subscribe/post?u=USER_ID&amp;id=LIST_ID" method="post" target="_blank"><input type="email" name="EMAIL" placeholder="Your best email" required style="padding:14px 20px;border:2px solid #ddd;border-radius:50px;width:280px;font-size:1em"><button type="submit" style="background:#667eea;color:white;padding:14px 30px;border:none;border-radius:50px;font-size:1em;font-weight:bold;cursor:pointer;margin-left:10px">Subscribe Free</button></form></div>`
}

func popunder() string { return `<!-- PP --><script type="text/javascript" src="https://PROPELLER_ADS_ZONE.script" data-cfasync="false"></script>` }

func outbrain() string { return `<div class="OUTBRAIN" data-src="https://emerald-engine.com" data-widget-id="AR_1"></div><script async src="https://widgets.outbrain.com/outbrain.js"></script>` }

// ─── CSS ───

var commonCSS = `
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;min-height:100vh;display:flex;flex-direction:column;align-items:center}
.container{max-width:1200px;width:100%;padding:20px}
.ac{background:rgba(0,0,0,0.03);padding:16px;border-radius:10px;text-align:center;margin:20px 0;min-height:90px;display:flex;align-items:center;justify-content:center}
.g{display:grid;grid-template-columns:repeat(auto-fit,minmax(300px,1fr));gap:30px;margin:40px 0}
footer{color:#999;padding:40px 0;text-align:center;width:100%;font-size:0.9em;border-top:1px solid #eee;margin-top:60px}
a{color:#667eea;text-decoration:none}a:hover{text-decoration:underline}
.btn{display:inline-block;padding:14px 40px;border:none;border-radius:50px;font-size:1.1em;font-weight:bold;cursor:pointer;transition:all 0.3s;text-decoration:none}
.btn-p{background:#667eea;color:white}.btn-p:hover{transform:scale(1.05);text-decoration:none}
.btn-s{background:#2ecc71;color:white}.btn-w{background:#f39c12;color:white}.btn-d{background:#e74c3c;color:white}
.ec{background:linear-gradient(135deg,#667eea,#764ba2);color:white;padding:40px;border-radius:15px;text-align:center;margin:40px 0}
.ec h3{margin-bottom:20px;font-size:1.5em}
.ec input[type=email]{padding:14px 20px;border:none;border-radius:50px 0 0 50px;width:300px;font-size:1em}
.ec button{padding:14px 30px;border:none;border-radius:0 50px 50px 0;font-size:1em;font-weight:bold;cursor:pointer;background:#ff6b6b;color:white}
.cd{background:#f8f9fa;padding:30px;border-radius:15px;margin:40px 0;text-align:center}
.cd code{background:#e9ecef;padding:4px 8px;border-radius:4px;font-size:0.85em;word-break:break-all}
.ag{display:grid;grid-template-columns:repeat(auto-fit,minmax(250px,1fr));gap:20px;margin:30px 0}
.acd{border:1px solid #eee;padding:25px;border-radius:12px;transition:box-shadow 0.3s}
.acd:hover{box-shadow:0 5px 20px rgba(0,0,0,0.1)}
.acd h4{margin-bottom:10px}.acd .pr{font-size:1.3em;color:#2ecc71;font-weight:bold}
.rb{background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)}
.rb h3{color:#333;margin-bottom:15px}
.rt{color:#f39c12;font-size:1.3em}
.pc{display:grid;grid-template-columns:1fr 1fr;gap:20px;margin:20px 0}
.prs{background:#e8f8f5;padding:20px;border-radius:10px}
.cns{background:#fdf2f2;padding:20px;border-radius:10px}
.bg{display:inline-block;background:#667eea;color:white;padding:4px 14px;border-radius:20px;font-size:0.8em;margin-bottom:12px}
@media(max-width:768px){.ec input[type=email]{width:100%;border-radius:50px;margin-bottom:10px}.ec button{width:100%;border-radius:50px}.pc{grid-template-columns:1fr}}
`

func commonHead(title, desc string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>%s</title><meta name="description" content="%s"><meta name="robots" content="index,follow"><link rel="canonical" href="%s">%s<style>%s</style></head><body><div class="container">`, title, desc, baseURL, gaTag(), commonCSS)
}

func foot() string {
	return fmt.Sprintf(`</div><footer><p>&copy; %d Emerald Engine. All rights reserved.</p><p style="margin-top:8px;font-size:0.8em;opacity:0.6;">As an Amazon Associate we earn from qualifying purchases. Affiliate links may earn us commission.</p><p style="margin-top:4px"><a href="/">Home</a> · <a href="/compare.html">Compare</a> · <a href="/review.html">Reviews</a> · <a href="/resources.html">Resources</a> · <a href="/blog/">Blog</a></p></footer></body></html>`, time.Now().Year())
}

// ─── Page Generators ───

func genIndex(niche Niche) string {
	scheme := pick(colorSchemes)
	title := niche.Name + " - AI-Powered Solutions"
	desc := fmt.Sprintf("Discover cutting-edge %s solutions. Boost your results with AI-driven tools.", niche.Name)
	afs := amazonLinks()
	llmBlurb := llmGenerateOrTemplate(
		fmt.Sprintf("Write 2 sentences promoting a %s landing page. Make it compelling and benefit-driven.", niche.Name),
		fmt.Sprintf("Automate your %s workflow with AI. Get more done in less time with our intelligent platform.", niche.Name),
	)
	return fmt.Sprintf(`%s<header style="text-align:center;padding:60px 0 30px;background:linear-gradient(135deg,%s 0%%,%s 100%%);border-radius:0 0 30px 30px;color:white;margin:-20px -20px 0"><div class="bg" style="background:rgba(255,255,255,0.2)">%s %s</div><h1 style="font-size:3em;margin-bottom:15px">%s</h1><p style="font-size:1.2em;opacity:0.9;max-width:700px;margin:0 auto">%s</p></header><div class="ac">%s</div><p style="line-height:1.8;color:#444;font-size:1.1em;margin:30px 0;text-align:center">%s</p><div class="g"><div class="acd" style="background:white;padding:35px;border-radius:15px;box-shadow:0 8px 25px rgba(0,0,0,0.08)"><h3 style="color:#333;margin-bottom:12px;font-size:1.4em">AI Automation</h3><p style="color:#666;line-height:1.7">Automate repetitive tasks and focus on growth. AI handles the heavy lifting.</p></div><div class="acd" style="background:white;padding:35px;border-radius:15px;box-shadow:0 8px 25px rgba(0,0,0,0.08)"><h3 style="color:#333;margin-bottom:12px;font-size:1.4em">Smart Analytics</h3><p style="color:#666;line-height:1.7">Real-time dashboards with ML-powered insights and predictions.</p></div><div class="acd" style="background:white;padding:35px;border-radius:15px;box-shadow:0 8px 25px rgba(0,0,0,0.08)"><h3 style="color:#333;margin-bottom:12px;font-size:1.4em">Auto-Optimization</h3><p style="color:#666;line-height:1.7">Continuous testing and optimization across all channels.</p></div></div><div style="text-align:center;margin:40px 0"><a href="#" class="btn btn-p" style="background:%s">Get Started Free</a></div><div class="ac">%s</div>%s<div class="ag"><div class="acd"><h4>Tool #1</h4><p>Top-rated solution</p><div class="pr">$29/mo</div>%s</div><div class="acd"><h4>Tool #2</h4><p>Trusted by 10K+</p><div class="pr">$49/mo</div>%s</div><div class="acd"><h4>Tool #3</h4><p>Best value</p><div class="pr">$99/mo</div>%s</div></div><div class="ac">%s</div>%s%s%s`,
		commonHead(title, desc),
		scheme[0], scheme[1], niche.Emoji, niche.Name, title, desc,
		adBanner(), llmBlurb,
		scheme[2],
		adInArticle(),
		emailForm(),
		strings.Split(afs, "\n")[0], strings.Split(afs, "\n")[1], strings.Split(afs, "\n")[2],
		adMatched(),
		cryptoDonate(),
		popunder(),
		foot())
}

func genCompare() string {
	title := "Best AI Marketing Tools Compared 2026"
	desc := "Compare top AI marketing platforms. Feature comparison, pricing, and real user reviews."
	llmIntro := llmGenerateOrTemplate(
		"Write 3 sentences comparing AI marketing tools for a comparison page. Be informative and highlight key differences.",
		"We've tested and compared the leading AI marketing platforms so you can choose the best fit for your business.",
	)
	rows := ""
	type P struct{ Name, Rating, Price, Pros, Cons, Link, SPrice string }
	prods := []P{
		{"ToolAlpha Pro", "★★★★★", "$29/mo", "Best AI automation; Intuitive UI; 24/7 support", "Limited integrations; No mobile app", "https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20", "29"},
		{"MarketGenius 360", "★★★★☆", "$49/mo", "Advanced analytics; Multi-channel; API", "Steep learning curve", "https://HOPLINK.hop.clickbank.net", "49"},
		{"ConvertFlow AI", "★★★★★", "$19/mo", "Best value; Easy setup; Templates", "Fewer features", "https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID", "19"},
		{"GrowthEngine X", "★★★★☆", "$79/mo", "Enterprise features; Custom AI models", "Expensive; Complex setup", "https://www.digistore24.com/redir/PRODUCT_ID", "79"},
	}
	for _, p := range prods {
		rows += fmt.Sprintf(`<div class="rb"><div class="rt">%s</div><h3>%s</h3><div class="pr" style="font-size:1.5em;margin:10px 0">%s</div><div class="pc"><div class="prs"><strong>✓ Pros:</strong><br>%s</div><div class="cns"><strong>✗ Cons:</strong><br>%s</div></div><div style="display:flex;gap:15px;flex-wrap:wrap;margin-top:15px"><a href="%s" class="btn btn-p" target="_blank" rel="nofollow sponsored">Buy via Affiliate →</a>%s</div></div>`, p.Rating, p.Name, p.Price, p.Pros, p.Cons, p.Link, stripeBtn(p.Name, p.SPrice))
	}
	return fmt.Sprintf(`%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">%s</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">%s</p></header><div class="ac">%s</div><p style="text-align:center;color:#444;font-size:1.05em;margin:20px 0">%s</p>%s<div class="ac">%s</div>%s<div class="ac">%s</div><p style="text-align:center;color:#999;margin:20px 0">Prices as of %s. Affiliate links may earn commission.</p>%s%s`,
		commonHead(title, desc), title, desc,
		adBanner(), llmIntro,
		rows,
		adInArticle(),
		emailForm(),
		adSidebar(),
		time.Now().Format("January 2006"),
		amazonAd(),
		foot())
}

func genReview() string {
	title := "Comprehensive Review: Top AI Marketing Platform 2026"
	desc := "In-depth review of leading AI marketing platform. Features, pricing, pros & cons, performance data."
	prod := pick([]string{"AI Marketing Hub Pro", "SmartCampaign 360", "ConvertBot Elite", "MarketForge AI", "GrowthEngine X"})
	rating := pick([]string{"★★★★★", "★★★★☆", "★★★★★"})
	pros := pickN([]string{"AI automation saves 20+ hrs/week", "Beautiful dashboard", "24/7 live support", "500+ integrations", "Real-time tracking", "Smart A/B testing", "No coding needed", "Regular updates", "Great enterprise value", "30-day guarantee"}, 5)
	cons := pickN([]string{"Pricey for small teams", "Learning curve for advanced features", "Some integrations in beta", "Mobile app could improve", "Limited customization on basic plan"}, 3)
	llmBody := llmGenerateOrTemplate(
		fmt.Sprintf("Write a 3-paragraph review of %s, an AI marketing tool. Highlight features, benefits, and who it's best for. Professional tone.", prod),
		fmt.Sprintf("%s is a comprehensive AI-powered marketing platform designed to automate and optimize your entire marketing funnel. From lead generation to conversion tracking, it uses machine learning to continuously improve campaign performance.", prod),
	)
	return fmt.Sprintf(`%s<header style="text-align:center;padding:50px 0 20px"><span class="bg">⭐ Editor's Pick 2026</span><h1 style="font-size:2.5em;color:#333;margin:10px 0">%s Review</h1><div class="rt" style="font-size:2em">%s</div><p style="color:#666;margin-top:10px">Our analysis after 90 days of testing</p></header><div class="ac">%s</div><div class="rb"><h2>What Is %s?</h2><p style="line-height:1.8;color:#444">%s</p></div><div class="ac">%s</div><div class="rb"><h2>Pros & Cons</h2><div class="pc"><div class="prs"><strong>✅ Pros</strong><br>%s</div><div class="cns"><strong>❌ Cons</strong><br>%s</div></div></div><div style="text-align:center;margin:30px 0"><a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-p" target="_blank" rel="nofollow sponsored" style="font-size:1.2em;padding:18px 60px">Check Price on Amazon →</a></div><div class="ac">%s</div>%s<div class="rb"><h2>Key Features</h2><div class="g"><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">🤖 AI</h3><p style="color:#666">Smart automation and predictive adjustments.</p></div><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">📊 Analytics</h3><p style="color:#666">Real-time dashboards with AI insights.</p></div><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">🔗 Integrations</h3><p style="color:#666">500+ tool integrations.</p></div></div></div><div class="ac">%s</div>%s<div style="text-align:center;margin:40px 0"><h3 style="margin-bottom:20px">Ready to Try %s?</h3><div style="display:flex;gap:15px;justify-content:center;flex-wrap:wrap"><a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-p" target="_blank" rel="nofollow sponsored">Amazon</a><a href="https://HOPLINK.hop.clickbank.net" class="btn btn-s" target="_blank" rel="nofollow sponsored">Official</a><a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" class="btn btn-w" target="_blank" rel="nofollow sponsored">Deal</a></div></div>%s%s`,
		commonHead(title, desc), prod, rating,
		adBanner(), prod, llmBody,
		adInArticle(),
		strings.Join(pros, "<br>"), strings.Join(cons, "<br>"),
		adMatched(),
		emailForm(),
		adSidebar(),
		cryptoDonate(),
		prod,
		amazonAd(),
		foot())
}

func genResources() string {
	title := "Ultimate Marketing Resource Directory 2026"
	desc := "Hand-picked marketing tools, resources, and guides. Save time with curated recommendations."
	res := []struct{ N, D, L string }{
		{"AI Marketing Platform", "All-in-one AI marketing automation", "https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20"},
		{"Email Marketing Tool", "Best email automation platform", "https://HOPLINK.hop.clickbank.net"},
		{"SEO Analyzer", "Comprehensive SEO audit tool", "https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID"},
		{"Social Media Manager", "Schedule and optimize social", "https://www.digistore24.com/redir/PRODUCT_ID"},
		{"Analytics Dashboard", "Real-time marketing analytics", "https://www.jvzoo.com/buy/PRODUCT_ID"},
		{"Landing Page Builder", "High-converting landing pages", "https://warriorplus.com/buy/PRODUCT_ID"},
	}
	cards := ""
	for _, r := range res {
		cards += fmt.Sprintf(`<div class="acd"><h4>%s</h4><p>%s</p><a href="%s" class="btn btn-p" target="_blank" rel="nofollow sponsored" style="font-size:0.9em;padding:10px 24px">Learn More →</a></div>`, r.N, r.D, r.L)
	}
	return fmt.Sprintf(`%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">%s</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">%s</p></header><p style="margin:20px 0 10px;font-size:0.9em;color:#999">Disclosure: Affiliate links. We may earn commission at no extra cost.</p><div class="ac">%s</div><div class="ag">%s</div><div class="ac">%s</div>%s<div class="ac">%s</div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), title, desc,
		adBanner(), cards,
		adInArticle(),
		emailForm(),
		adSidebar(), adMatched(),
		cryptoDonate(),
		foot())
}

func genBlog() string {
	niche := pick(niches)
	title := pick([]string{
		"How to Boost Your Marketing ROI with AI in 2026",
		"The Complete Guide to Automated Lead Generation",
		"10 Proven Strategies to Double Your Conversion Rate",
		fmt.Sprintf("How to Master %s Automation", niche.Name),
		fmt.Sprintf("The Ultimate %s Growth Playbook", niche.Name),
	})
	desc := fmt.Sprintf("Complete guide to %s. Proven strategies, expert tips, tools, and checklists.", title)
	h := []string{
		"Why Traditional Marketing Is Failing",
		"The AI Revolution in Marketing",
		"Step 1: Automate Lead Generation",
		"Step 2: Optimize Your Funnel",
		"Step 3: Scale with Analytics",
		"Real Results from Testing",
	}
	p := []string{
		"Traditional marketing methods are no longer enough. AI-powered automation is essential to cut through noise and reach your audience effectively.",
		fmt.Sprintf("AI has transformed marketing. Businesses using AI see 40%%+ efficiency gains and 25%% higher conversions. The %s industry is being disrupted.", niche.Name),
		"AI tools identify high-intent visitors, score leads, and trigger personalized follow-up sequences automatically.",
		"AI analytics identify exactly where prospects drop off and adjust messaging, timing, and offers to improve conversions.",
		"Modern platforms process millions of data points in real-time, identifying patterns humans would miss.",
		"AI-powered campaigns outperform manual ones by 3.2x in conversion rate and 4.7x in revenue per visitor.",
	}
	content := ""
	llmContent := llmGenerateOrTemplate(
		fmt.Sprintf("Write a 150-word blog post section about %s. First paragraph about why it matters, second about how to get started. Professional but accessible tone.", title),
		"",
	)
	if llmContent != "" {
		content = fmt.Sprintf(`<div class="rb"><p style="line-height:1.8;color:#444">%s</p></div>`, llmContent)
	}
	for i := 0; i < len(h); i++ {
		content += fmt.Sprintf(`<h2 style="margin:35px 0 15px;color:#333">%s</h2><p style="line-height:1.8;color:#444;margin-bottom:15px">%s</p>`, h[i], p[i])
		if i == 2 || i == 4 {
			content += fmt.Sprintf(`<div class="ac">%s</div>`, adInArticle())
		}
	}
	return fmt.Sprintf(`%s<article><header style="text-align:center;padding:40px 0 20px"><div class="bg">📝 Blog</div><h1 style="font-size:2.2em;color:#333;margin:10px 0;line-height:1.3">%s</h1><p style="color:#999;font-size:0.9em">Published: %s · Category: %s · 8 min read</p></header><div class="ac">%s</div>%s<div class="ac">%s</div>%s<div class="ac">%s</div><div style="background:#f8f9fa;padding:40px;border-radius:15px;margin:40px 0;text-align:center"><h3 style="margin-bottom:15px">🚀 Ready to Get Started?</h3><p style="margin-bottom:20px;color:#666">Try the tools we recommend. Start free today.</p><div style="display:flex;gap:15px;justify-content:center;flex-wrap:wrap"><a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-p" target="_blank" rel="nofollow sponsored">Amazon</a><a href="https://HOPLINK.hop.clickbank.net" class="btn btn-s" target="_blank" rel="nofollow sponsored">ClickBank</a><a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" class="btn btn-w" target="_blank" rel="nofollow sponsored">ShareASale</a></div></div>%s%s</article>%s`,
		commonHead(title, desc), title,
		time.Now().Format("January 2, 2006"), niche.Name,
		adBanner(),
		content,
		adSidebar(),
		emailForm(),
		adMatched(),
		outbrain(),
		foot())
}

// ─── Sitemap / Robots ───

func genSitemap(pages []string) string {
	urls := ""
	for _, p := range pages {
		urls += fmt.Sprintf("<url><loc>%s/%s</loc><changefreq>daily</changefreq><priority>0.8</priority></url>", baseURL, p)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">%s</urlset>`, urls)
}

func genRobots() string { return fmt.Sprintf("User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", baseURL) }

// ─── Git ───

func gitCommitPush(files []string) error {
	args := append([]string{"add"}, files...)
	if err := exec.Command("git", args...).Run(); err != nil {
		return fmt.Errorf("add: %w", err)
	}
	ts := time.Now().Format(time.RFC3339)
	if err := exec.Command("git", "commit", "--allow-empty", "-m", fmt.Sprintf("auto: monetized pages [%s]", ts)).Run(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	for _, r := range []string{"origin", "huggingface"} {
		if err := exec.Command("git", "push", r, "main").Run(); err != nil {
			fmt.Fprintf(os.Stderr, "[GIT] push %s: %v\n", r, err)
		} else {
			fmt.Printf("[GIT] pushed %s\n", r)
		}
	}
	return nil
}

// ─── Main ───

func main() {
	rand.Seed(time.Now().UnixNano())
	vault = loadVault()
	initLLMProviders()

	fmt.Printf("[ENGINE] Emerald Engine v4.0 starting\n")
	fmt.Printf("[ENGINE] Cycle: %ds | LLM: %d providers\n", cycleDelay, len(llmProviders))

	os.MkdirAll("public/blog", 0755)

	outputFiles := []string{
		"public/index.html", "public/compare.html", "public/review.html",
		"public/resources.html", "public/blog/getting-started.html",
		"public/sitemap.xml", "public/robots.txt",
	}

	for i := 1; ; i++ {
		fmt.Printf("\n[ENGINE] === Cycle %d ===\n", i)
		start := time.Now()
		total := 0

		niche := pick(niches)
		fmt.Printf("[ENGINE] Niche: %s %s\n", niche.Emoji, niche.Name)

		pages := map[string]func() string{
			"public/index.html":                func() string { return genIndex(niche) },
			"public/compare.html":              genCompare,
			"public/review.html":               genReview,
			"public/resources.html":            genResources,
			"public/blog/getting-started.html": genBlog,
		}

		for path, gen := range pages {
			html := gen()
			os.MkdirAll(filepath.Dir(path), 0755)
			os.WriteFile(path, []byte(html), 0644)
			total += len(html)
			fmt.Printf("[ENGINE] ✓ %s (%d bytes)\n", path, len(html))
		}

		allPages := []string{"index.html", "compare.html", "review.html", "resources.html", "blog/getting-started.html"}
		site := genSitemap(allPages)
		os.WriteFile("public/sitemap.xml", []byte(site), 0644)
		total += len(site)

		rob := genRobots()
		os.WriteFile("public/robots.txt", []byte(rob), 0644)
		total += len(rob)

		fmt.Printf("[ENGINE] Total: %d bytes across %d files\n", total, len(outputFiles))

		if err := gitCommitPush(outputFiles); err != nil {
			fmt.Fprintf(os.Stderr, "[ENGINE] git error: %v\n", err)
		}

		elapsed := time.Since(start)
		summary := fmt.Sprintf("Cycle %d: %d pages, %d bytes, %v", i, len(outputFiles), total, elapsed)
		telegramSend(fmt.Sprintf("<b>🤖 Emerald Engine v4</b>\n%s\nNiche: %s %s\nRemotes: origin + huggingface", summary, niche.Emoji, niche.Name))

		sleep := cycleDelay*time.Second - elapsed
		if sleep > 0 {
			fmt.Printf("[ENGINE] Done in %v. Next in %v\n", elapsed, sleep)
			time.Sleep(sleep)
		} else {
			fmt.Printf("[ENGINE] Done in %v (exceeded interval)\n", elapsed)
		}
	}
}
