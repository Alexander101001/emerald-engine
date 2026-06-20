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
		"chat_id":    chatID,
		"text":       msg,
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
	Name  string
	URL   string
	Key   string
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

	client := http.Client{Timeout: 5 * time.Second}
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
	{"Fitness & Nutrition", "fitness-nutrition", "🏋️"},
	{"Remote Work", "remote-work", "🏡"},
	{"Content Creation", "content-creation", "🎬"},
	{"Cybersecurity", "cybersecurity", "🔒"},
	{"Self Improvement", "self-improvement", "🌱"},
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
	{"#ee9ca7", "#ffdde1", "#667eea", "#e3a5b0"},
	{"#1cb5e0", "#000046", "#f39c12", "#001f3f"},
	{"#f857a6", "#ff5858", "#2ecc71", "#cc2366"},
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

func gaTag() string {
	id := vaultGet("GA_ID", "G-XXXXXXXXXX")
	return fmt.Sprintf(`<!-- GA4 --><script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script><script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config','%s');</script>`, id, id)
}

func adClientID() string { return vaultGet("ADSENSE_CLIENT_ID", "ca-pub-XXXXXXXXXXXXXXXX") }

func adBanner() string {
	return fmt.Sprintf(`<!-- AS --><ins class="adsbygoogle" style="display:block" data-ad-client="%s" data-ad-slot="XXXXXXXXXX" data-ad-format="auto" data-full-width-responsive="true"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({});</script>`, adClientID())
}
func adInArticle() string {
	return fmt.Sprintf(`<!-- AS --><ins class="adsbygoogle" style="display:block;text-align:center" data-ad-layout="in-article" data-ad-format="fluid" data-ad-client="%s" data-ad-slot="XXXXXXXXXX"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({});</script>`, adClientID())
}
func adSidebar() string {
	return fmt.Sprintf(`<!-- AS --><ins class="adsbygoogle" style="display:block" data-ad-client="%s" data-ad-slot="XXXXXXXXXX" data-ad-format="auto" data-full-width-responsive="true"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({});</script>`, adClientID())
}
func adMatched() string {
	return fmt.Sprintf(`<!-- AS --><ins class="adsbygoogle" style="display:block" data-ad-format="autorelaxed" data-ad-client="%s" data-ad-slot="XXXXXXXXXX"></ins><script>(adsbygoogle=window.adsbygoogle||[]).push({});</script>`, adClientID())
}

func amazonTag() string {
	tag := vaultGet("AMAZON_ASSOCIATES_TAG", "emeraldeng0e-20")
	return tag
}

func amazonAffiliateLinks() string {
	tag := amazonTag()
	return fmt.Sprintf(`<a href="https://www.amazon.com/dp/B0BXYZ?tag=%s" rel="nofollow sponsored" target="_blank">Amazon #1</a>
<a href="https://www.amazon.com/dp/B0CABC?tag=%s" rel="nofollow sponsored" target="_blank">Amazon #2</a>
<a href="https://www.amazon.com/dp/B0DDEF?tag=%s" rel="nofollow sponsored" target="_blank">Amazon #3</a>
<img src="https://ir-na.amazon-adsystem.com/e/ir?t=%s&l=am2&o=1" width="1" height="1" border="0" alt="" style="border:none!important;margin:0!important"/>`,
		tag, tag, tag, tag)
}

func amazonAd() string {
	return fmt.Sprintf(`<iframe src="https://rcm-na.amazon-adsystem.com/e/cm?o=1&p=48&l=ur1&category=amzn_main&banner=0P3Y7BVGKXBQ5CXKNW02&f=ifr&linkID=amzn-main&t=%s" width="728" height="90" scrolling="no" border="0" marginwidth="0" style="border:none" frameborder="0"></iframe>`, amazonTag())
}

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
.tc{display:flex;gap:15px;justify-content:center;flex-wrap:wrap;margin:20px 0}
.tc a{display:inline-flex;align-items:center;gap:6px;padding:10px 20px;border-radius:50px;background:#f8f9fa;border:1px solid #e0e0e0;font-size:0.9em;transition:all 0.3s}
.tc a:hover{background:#667eea;color:white;border-color:#667eea}
@media(max-width:768px){.ec input[type=email]{width:100%;border-radius:50px;margin-bottom:10px}.ec button{width:100%;border-radius:50px}.pc{grid-template-columns:1fr}}
`

func commonHead(title, desc string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>%s</title><meta name="description" content="%s"><meta name="robots" content="index,follow"><link rel="canonical" href="%s/%s">%s<style>%s</style>%s</head><body><div class="container">`,
		title, desc, baseURL,
		strings.ReplaceAll(strings.ToLower(title), " ", "-"),
		gaTag(), commonCSS, toolJS)
}

func foot() string {
	return fmt.Sprintf(`</div><footer><p>&copy; %d Emerald Engine. All rights reserved.</p><p style="margin-top:8px;font-size:0.8em;opacity:0.6;">As an Amazon Associate we earn from qualifying purchases. Affiliate links may earn us commission.</p><p style="margin-top:4px"><a href="/">Home</a> · <a href="/tools/">Tools</a> · <a href="/compare.html">Compare</a> · <a href="/review.html">Reviews</a> · <a href="/resources.html">Resources</a> · <a href="/products.html">Products</a> · <a href="/blog/">Blog</a> · <a href="/about.html">About</a> · <a href="/contact.html">Contact</a> · <a href="/privacy.html">Privacy</a></p></footer></body></html>`, time.Now().Year())
}

func navBar(niche Niche) string {
	return fmt.Sprintf(`<div class="tc"><a href="/">🏠 Home</a><a href="/tools/">🔧 Tools</a><a href="/compare.html">📊 Compare</a><a href="/review.html">⭐ Reviews</a><a href="/products.html">💳 Products</a><a href="/resources.html">📚 Resources</a><a href="/blog/">📝 Blog</a></div>`)
}

// ─── Page Generators ───

func genIndex(niche Niche) string {
	scheme := pick(colorSchemes)
	title := niche.Name + " - AI-Powered Solutions"
	desc := fmt.Sprintf("Discover cutting-edge %s solutions. Boost your results with AI-driven tools.", niche.Name)
	llmBlurb := llmGenerateOrTemplate(
		fmt.Sprintf("Write 2 sentences promoting a %s landing page. Make it compelling and benefit-driven.", niche.Name),
		fmt.Sprintf("Automate your %s workflow with AI. Get more done in less time with our intelligent platform.", niche.Name),
	)
	prods := fetchAmazonProducts(niche.Keyword, amazonTag())
	prodCards := ""
	for _, p := range prods {
		prodCards += fmt.Sprintf(
			`<div class="acd" style="text-align:center"><div class="rt">%.1f ★</div><h4>%s</h4><p style="font-size:0.9em;color:#999">%s</p><div class="pr">$%.2f</div><a href="%s" class="btn btn-p" target="_blank" rel="nofollow sponsored" style="font-size:0.9em;padding:10px 24px;margin-top:10px;display:inline-block">Buy on Amazon →</a></div>`,
			p.Rating, p.Title, p.Category, p.Price, p.URL,
		)
	}
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:60px 0 30px;background:linear-gradient(135deg,%s 0%%,%s 100%%);border-radius:0 0 30px 30px;color:white;margin:-20px -20px 0"><div class="bg" style="background:rgba(255,255,255,0.2)">%s %s</div><h1 style="font-size:3em;margin-bottom:15px">%s</h1><p style="font-size:1.2em;opacity:0.9;max-width:700px;margin:0 auto">%s</p>%s</header><div class="ac">%s</div><p style="line-height:1.8;color:#444;font-size:1.1em;margin:30px 0;text-align:center">%s</p><div class="g"><div class="acd" style="background:white;padding:35px;border-radius:15px;box-shadow:0 8px 25px rgba(0,0,0,0.08)"><h3 style="color:#333;margin-bottom:12px;font-size:1.4em">AI Automation</h3><p style="color:#666;line-height:1.7">Automate repetitive tasks and focus on growth. AI handles the heavy lifting.</p></div><div class="acd" style="background:white;padding:35px;border-radius:15px;box-shadow:0 8px 25px rgba(0,0,0,0.08)"><h3 style="color:#333;margin-bottom:12px;font-size:1.4em">Smart Analytics</h3><p style="color:#666;line-height:1.7">Real-time dashboards with ML-powered insights and predictions.</p></div><div class="acd" style="background:white;padding:35px;border-radius:15px;box-shadow:0 8px 25px rgba(0,0,0,0.08)"><h3 style="color:#333;margin-bottom:12px;font-size:1.4em">Auto-Optimization</h3><p style="color:#666;line-height:1.7">Continuous testing and optimization across all channels.</p></div></div><div style="text-align:center;margin:40px 0"><a href="#" class="btn btn-p" style="background:%s">Get Started Free</a></div><div class="ac">%s</div>%s<h2 style="margin:40px 0 20px;color:#333;text-align:center">Recommended %s Products</h2><div class="ag">%s</div>%s<div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche),
		scheme[0], scheme[1], niche.Emoji, niche.Name, title, desc,
		generateSubdomainLinks(niche.Keyword),
		adBanner(), llmBlurb,
		scheme[2],
		adInArticle(),
		dynamicEmailForm(),
		niche.Name, prodCards,
		adMatched(),
		cryptoDonate(),
		popunder(),
		foot())
}

func genCompare() string {
	niche := pick(niches)
	title := "Best " + niche.Name + " Tools Compared 2026"
	desc := fmt.Sprintf("Compare top %s platforms. Feature comparison, pricing, and real user reviews.", niche.Name)
	llmIntro := llmGenerateOrTemplate(
		"Write 3 sentences comparing AI marketing tools for a comparison page. Be informative and highlight key differences.",
		"We've tested and compared the leading AI marketing platforms so you can choose the best fit for your business.",
	)
	rows := ""
	type P struct{ Name, Rating, Price, Pros, Cons, Link, SPrice string }
	prods := fetchAmazonProducts(niche.Keyword, amazonTag())
	for i, p := range prods {
		if i >= 4 {
			break
		}
		ratingStars := strings.Repeat("★", int(p.Rating)) + strings.Repeat("☆", 5-int(p.Rating))
		rows += fmt.Sprintf(`<div class="rb"><div class="rt">%s</div><h3>%s</h3><div class="pr" style="font-size:1.5em;margin:10px 0">$%.2f</div><div class="pc"><div class="prs"><strong>✓ Key Features:</strong><br>AI-powered · Cloud-based · 24/7 Support</div><div class="cns"><strong>✗ Considerations:</strong><br>Price varies · Check listing for details</div></div><div style="display:flex;gap:15px;flex-wrap:wrap;margin-top:15px"><a href="%s" class="btn btn-p" target="_blank" rel="nofollow sponsored">Check Price →</a>%s</div></div>`, ratingStars, p.Title, p.Price, p.URL, stripeBtn(p.Title, fmt.Sprintf("%.0f", p.Price)))
	}
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">%s</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">%s</p></header><div class="ac">%s</div><p style="text-align:center;color:#444;font-size:1.05em;margin:20px 0">%s</p>%s<div class="ac">%s</div>%s<div class="ac">%s</div><p style="text-align:center;color:#999;margin:20px 0">Prices as of %s. Affiliate links may earn commission.</p>%s%s`,
		commonHead(title, desc), navBar(niche), title, desc,
		adBanner(), llmIntro,
		rows,
		adInArticle(),
		dynamicEmailForm(),
		adSidebar(),
		time.Now().Format("January 2006"),
		amazonAd(),
		foot())
}

func genReview() string {
	niche := pick(niches)
	prod := pick(fetchAmazonProducts(niche.Keyword, amazonTag()))
	title := fmt.Sprintf("Comprehensive Review: %s 2026", prod.Title)
	desc := fmt.Sprintf("In-depth review of %s. Features, pricing, pros & cons, performance data.", prod.Title)
	pros := pickN([]string{"Excellent build quality", "Easy to set up", "Great value for money", "Outstanding customer support", "Feature-rich", "Regular updates", "Industry-leading warranty", "User-friendly interface", "High reliability", "Energy efficient"}, 5)
	cons := pickN([]string{"Premium pricing", "Limited color options", "May be overkill for casual users", "Requires initial setup time", "Some accessories sold separately"}, 3)
	llmBody := llmGenerateOrTemplate(
		fmt.Sprintf("Write a 3-paragraph review of %s. Highlight features, benefits, and who it's best for. Professional tone.", prod.Title),
		fmt.Sprintf("%s is a comprehensive product designed to deliver outstanding results. With top-rated features and reliable performance, it's an excellent choice for professionals and enthusiasts alike.", prod.Title),
	)
	stars := strings.Repeat("★", int(prod.Rating)) + strings.Repeat("☆", 5-int(prod.Rating))
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 20px"><span class="bg">⭐ Editor's Pick 2026</span><h1 style="font-size:2.5em;color:#333;margin:10px 0">%s Review</h1><div class="rt" style="font-size:2em">%s</div><p style="color:#666;margin-top:10px">Our analysis after extensive testing</p></header><div class="ac">%s</div><div class="rb"><h2>What Is %s?</h2><p style="line-height:1.8;color:#444">%s</p></div><div class="ac">%s</div><div class="rb"><h2>Pros & Cons</h2><div class="pc"><div class="prs"><strong>✅ Pros</strong><br>%s</div><div class="cns"><strong>❌ Cons</strong><br>%s</div></div></div><div style="text-align:center;margin:30px 0"><a href="%s" class="btn btn-p" target="_blank" rel="nofollow sponsored" style="font-size:1.2em;padding:18px 60px">Check Price on Amazon →</a></div><div class="ac">%s</div>%s<div class="rb"><h2>Key Features</h2><div class="g"><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">⚡ Performance</h3><p style="color:#666">Industry-leading performance and reliability.</p></div><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">🎯 Precision</h3><p style="color:#666">Accurate and consistent results every time.</p></div><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">🔧 Quality</h3><p style="color:#666">Built with premium materials and craftsmanship.</p></div></div></div><div class="ac">%s</div>%s<div style="text-align:center;margin:40px 0"><h3 style="margin-bottom:20px">Ready to Buy %s?</h3><div style="display:flex;gap:15px;justify-content:center;flex-wrap:wrap"><a href="%s" class="btn btn-p" target="_blank" rel="nofollow sponsored">Amazon →</a>%s</div></div>%s%s`,
		commonHead(title, desc), navBar(niche), prod.Title, stars,
		adBanner(), prod.Title, llmBody,
		adInArticle(),
		strings.Join(pros, "<br>"), strings.Join(cons, "<br>"),
		prod.URL,
		adMatched(),
		dynamicEmailForm(),
		adSidebar(),
		cryptoDonate(),
		prod.Title,
		prod.URL, stripeBtn(prod.Title, fmt.Sprintf("%.0f", prod.Price)),
		amazonAd(),
		foot())
}

func genResources() string {
	niche := pick(niches)
	title := fmt.Sprintf("Ultimate %s Resource Directory 2026", niche.Name)
	desc := fmt.Sprintf("Hand-picked %s tools, resources, and guides. Save time with curated recommendations.", niche.Name)
	prods := fetchAmazonProducts(niche.Keyword, amazonTag())
	cards := ""
	for _, p := range prods {
		cards += fmt.Sprintf(`<div class="acd"><h4>%s</h4><p style="font-size:0.9em;color:#999">%s · $%.2f · %.1f★</p><a href="%s" class="btn btn-p" target="_blank" rel="nofollow sponsored" style="font-size:0.9em;padding:10px 24px">Check Price →</a></div>`, p.Title, p.Category, p.Price, p.Rating, p.URL)
	}
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">%s</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">%s</p></header><p style="margin:20px 0 10px;font-size:0.9em;color:#999">Disclosure: Affiliate links. We may earn commission at no extra cost.</p><div class="ac">%s</div><div class="ag">%s</div><div class="ac">%s</div>%s<div class="ac">%s</div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche), title, desc,
		adBanner(), cards,
		adInArticle(),
		dynamicEmailForm(),
		adSidebar(), adMatched(),
		cryptoDonate(),
		foot())
}

func genBlog() string {
	niche := pick(niches)
	title := pick([]string{
		"How to Boost Your " + niche.Name + " ROI with AI in 2026",
		"The Complete Guide to Automated " + niche.Name,
		"10 Proven " + niche.Name + " Strategies That Work",
		fmt.Sprintf("How to Master %s Automation", niche.Name),
		fmt.Sprintf("The Ultimate %s Growth Playbook", niche.Name),
	})
	desc := fmt.Sprintf("Complete guide to %s. Proven strategies, expert tips, tools, and checklists.", title)
	h := []string{
		"Why " + niche.Name + " Matters More Than Ever",
		"The AI Revolution in " + niche.Name,
		"Step 1: Automated Approach",
		"Step 2: Optimize Your Results",
		"Step 3: Scale with Analytics",
		"Real Results from Testing",
	}
	p := []string{
		niche.Name + " has never been more important. AI-powered solutions are transforming how businesses operate and compete in this space.",
		fmt.Sprintf("AI has transformed %s. Organizations using AI see 40%%+ efficiency gains and 25%% higher conversions. The industry is being disrupted.", niche.Name),
		"AI tools identify high-intent visitors, score leads, and trigger personalized follow-up sequences automatically.",
		"AI analytics identify exactly where prospects drop off and adjust messaging, timing, and offers to improve conversions.",
		"Modern platforms process millions of data points in real-time, identifying patterns humans would miss.",
		"AI-powered approaches outperform manual methods by 3.2x in conversion rate and 4.7x in revenue per visitor.",
	}
	content := ""
	llmContent := llmGenerateOrTemplate(
		fmt.Sprintf("Write a 150-word blog post section about %s. First paragraph about why it matters, second about how to get started.", title),
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
	return fmt.Sprintf(`%s%s<article><header style="text-align:center;padding:40px 0 20px"><div class="bg">📝 Blog</div><h1 style="font-size:2.2em;color:#333;margin:10px 0;line-height:1.3">%s</h1><p style="color:#999;font-size:0.9em">Published: %s · Category: %s · 8 min read</p></header><div class="ac">%s</div>%s<div class="ac">%s</div>%s<div class="ac">%s</div><div style="background:#f8f9fa;padding:40px;border-radius:15px;margin:40px 0;text-align:center"><h3 style="margin-bottom:15px">🚀 Ready to Get Started?</h3><p style="margin-bottom:20px;color:#666">Try the tools we recommend. Start free today.</p><div style="display:flex;gap:15px;justify-content:center;flex-wrap:wrap"><a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-p" target="_blank" rel="nofollow sponsored">Amazon</a><a href="https://HOPLINK.hop.clickbank.net" class="btn btn-s" target="_blank" rel="nofollow sponsored">ClickBank</a><a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" class="btn btn-w" target="_blank" rel="nofollow sponsored">ShareASale</a></div></div>%s%s</article>%s`,
		commonHead(title, desc), navBar(niche), title,
		time.Now().Format("January 2, 2006"), niche.Name,
		adBanner(),
		content,
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		outbrain(),
		foot())
}

func genProductsPage() string {
	niche := pick(niches)
	title := "Premium Digital Products - " + niche.Name
	desc := fmt.Sprintf("Professional %s tools and resources. Downloadable templates, guides, and software.", niche.Name)
	products := generateStripeProductList()
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">💳 Premium Products</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">Professional %s tools to accelerate your success. Instant digital delivery.</p></header><div class="ac">%s</div><div class="ag">%s</div><div class="ac">%s</div>%s<div class="ac">%s</div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche), niche.Name,
		adBanner(), products,
		adInArticle(),
		dynamicEmailForm(),
		adSidebar(), adMatched(),
		cryptoDonate(),
		foot())
}

func genAbout() string {
	niche := pick(niches)
	title := "About Emerald Engine - AI-Powered " + niche.Name + " Solutions"
	desc := "Learn about Emerald Engine. Our mission is to provide free, professional tools and resources for " + niche.Name + "."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">About Emerald Engine</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">Our mission: free professional tools for everyone</p></header><div class="ac">%s</div><div class="rb"><h2>Our Mission</h2><p style="line-height:1.8;color:#444">Emerald Engine provides free, professional-grade calculators, tools, and resources for %s professionals and enthusiasts. We believe high-quality business tools should be accessible to everyone.</p></div><div class="ac">%s</div><div class="rb"><h2>Why We're Different</h2><div class="g"><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">🆓 Always Free</h3><p style="color:#666">All our core tools are completely free to use. No sign-ups, no credit cards.</p></div><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">⚡ Instant Results</h3><p style="color:#666">No waiting. No processing. Get your calculations instantly in your browser.</p></div><div style="background:white;padding:25px;border-radius:12px;box-shadow:0 4px 15px rgba(0,0,0,0.06)"><h3 style="color:#333;margin-bottom:8px">🔒 Privacy First</h3><p style="color:#666">We don't track, store, or share your data. All calculations happen locally.</p></div></div></div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche), niche.Name,
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		foot())
}

func genContact() string {
	niche := pick(niches)
	title := "Contact Us - " + niche.Name + " Support"
	desc := "Get in touch with the Emerald Engine team. We're here to help with your " + niche.Name + " needs."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">📬 Contact Us</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">Have questions or suggestions? We'd love to hear from you.</p></header><div class="ac">%s</div><div class="rb" style="max-width:600px;margin:30px auto"><form><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Name</label><input type="text" placeholder="Your name" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Email</label><input type="email" placeholder="your@email.com" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Message</label><textarea rows="5" placeholder="How can we help?" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em;resize:vertical"></textarea></div><button type="submit" class="btn btn-p" style="font-size:1.1em">Send Message →</button></form></div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		dynamicEmailForm(),
		foot())
}

func genPrivacy() string {
	niche := pick(niches)
	title := "Privacy Policy - " + niche.Name
	desc := "Emerald Engine privacy policy. How we handle your data and privacy."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">🔒 Privacy Policy</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">Last updated: %s</p></header><div class="ac">%s</div><div class="rb"><h2>Information We Collect</h2><p style="line-height:1.8;color:#444">We use Google Analytics to understand how visitors use our site. This collects anonymized data about page views, time on site, and referral sources. We do not collect personally identifiable information unless you voluntarily submit it via our email subscription form.</p><h2 style="margin-top:30px">Cookies</h2><p style="line-height:1.8;color:#444">We use cookies for Google Analytics and AdSense. These are standard third-party cookies that help us serve relevant ads and understand site usage. You can disable cookies in your browser settings.</p><h2 style="margin-top:30px">Third-Party Services</h2><p style="line-height:1.8;color:#444">We use Google AdSense, Amazon Associates, and Stripe. Each has its own privacy policy governing how they handle your data. We do not share your personal data with these services unless you explicitly interact with them (click an ad, purchase a product).</p><h2 style="margin-top:30px">Your Rights</h2><p style="line-height:1.8;color:#444">You have the right to access, correct, or delete any personal data we hold. Contact us at privacy@emerald-engine.com for requests.</p></div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche),
		time.Now().Format("January 2, 2006"),
		adBanner(),
		adInArticle(),
		dynamicEmailForm(),
		foot())
}

func genTerms() string {
	niche := pick(niches)
	title := "Terms of Service - " + niche.Name
	desc := "Emerald Engine terms of service. Please read these terms carefully before using our services."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">📋 Terms of Service</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">Last updated: %s</p></header><div class="ac">%s</div><div class="rb"><h2>Use of Service</h2><p style="line-height:1.8;color:#444">By using Emerald Engine, you agree to these terms. Our tools are provided for informational and educational purposes. We make no guarantees about accuracy or availability.</p><h2 style="margin-top:30px">Affiliate Disclosure</h2><p style="line-height:1.8;color:#444">Some links on this site are affiliate links. We may earn a commission if you purchase through these links at no extra cost to you. This includes Amazon Associates, ClickBank, ShareASale, and other affiliate programs.</p><h2 style="margin-top:30px">Intellectual Property</h2><p style="line-height:1.8;color:#444">All content on this site is original and protected by copyright. You may not reproduce, distribute, or modify our content without permission.</p><h2 style="margin-top:30px">Limitation of Liability</h2><p style="line-height:1.8;color:#444">We are not liable for any damages arising from use of our tools or services. All tools are provided "as is" without warranty.</p></div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche),
		time.Now().Format("January 2, 2006"),
		adBanner(),
		adInArticle(),
		dynamicEmailForm(),
		foot())
}

func genDisclosure() string {
	niche := pick(niches)
	title := "Affiliate Disclosure - " + niche.Name
	desc := "Emerald Engine affiliate disclosure. Transparency about our affiliate relationships."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">📢 Affiliate Disclosure</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">Transparency about our monetization</p></header><div class="ac">%s</div><div class="rb"><p style="line-height:1.8;color:#444">Emerald Engine participates in various affiliate programs. When you click on links to products and make a purchase, we may earn a commission at no additional cost to you.</p><h2 style="margin-top:30px">Affiliate Programs We Participate In:</h2><ul style="line-height:2;color:#444;margin:15px 20px"><li>Amazon Associates Program</li><li>ClickBank</li><li>ShareASale</li><li>DigiStore24</li><li>JVZoo</li><li>WarriorPlus</li></ul><p style="line-height:1.8;color:#444;margin-top:20px">All affiliate links are marked with <code>rel="nofollow sponsored"</code>. We only recommend products we believe provide value to our users.</p></div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		dynamicEmailForm(),
		foot())
}

// ─── Micro Tools ───

const toolJS = `
<script>
function calcROI(){var i=parseFloat(document.getElementById('roi-invest').value)||0,r=parseFloat(document.getElementById('roi-return').value)||0,g=((r-i)/i*100).toFixed(2);document.getElementById('roi-result').innerHTML='<strong>ROI: '+g+'%</strong>'+(g>0?' 📈 Profit':' 📉 Loss')+(g>50?' — Excellent!':'')+(g<0?' — Needs improvement':'')+'<br><small>Net profit: $'+(r-i).toFixed(2)+'</small>'}
function calcProfit(){var c=parseFloat(document.getElementById('prof-cost').value)||0,p=parseFloat(document.getElementById('prof-price').value)||0,m=((p-c)/p*100).toFixed(2),pr=(p-c).toFixed(2);document.getElementById('prof-result').innerHTML='<strong>Margin: '+m+'%</strong><br>Profit per unit: $'+pr+'<br>Markup: '+((p-c)/c*100).toFixed(2)+'%'}
function calcConv(){var v=parseFloat(document.getElementById('conv-visitors').value)||0,c=parseFloat(document.getElementById('conv-conversions').value)||0,r=(c/v*100).toFixed(2);document.getElementById('conv-result').innerHTML='<strong>Conversion Rate: '+r+'%</strong><br>Visitors: '+v+' | Conversions: '+c+'<br>'+(r>5?'🔥 Excellent!':r>2?'👍 Good':r>1?'👌 Average':'💪 Room for improvement')}
function calcLead(){var v=parseFloat(document.getElementById('lead-visitors').value)||0,c=parseFloat(document.getElementById('lead-conversions').value)||0,cv=parseFloat(document.getElementById('lead-value').value)||0,cr=c/v*100,rev=c*cv;document.getElementById('lead-result').innerHTML='<strong>Lead Value Analysis</strong><br>Conversion Rate: '+cr.toFixed(2)+'%<br>Total Revenue: $'+rev.toFixed(2)+'<br>Revenue/Visitor: $'+(rev/v).toFixed(2)+'<br>If rate doubles: $'+(rev*2).toFixed(2)+' revenue'}
function calcBudget(){var b=parseFloat(document.getElementById('budget-total').value)||0;if(!b)return;var a=document.getElementById('budget-channel').value;var p={social:b*0.35,search:b*0.25,email:b*0.15,content:b*0.15,display:b*0.10};document.getElementById('budget-result').innerHTML='<strong>Recommended Budget Allocation</strong><br>📱 Social Media: $'+p.social.toFixed(2)+'<br>🔍 Search Ads: $'+p.search.toFixed(2)+'<br>📧 Email: $'+p.email.toFixed(2)+'<br>📝 Content: $'+p.content.toFixed(2)+'<br>🖼️ Display: $'+p.display.toFixed(2)}
function calcSEO(){var kw=parseFloat(document.getElementById('seo-volume').value)||0,df=parseFloat(document.getElementById('seo-difficulty').value)||0,cl=parseFloat(document.getElementById('seo-clicks').value)||0;var sc=Math.max(0,Math.min(100,100-df+Math.min(kw/100,30)+(cl>0?10:0)));document.getElementById('seo-result').innerHTML='<strong>SEO Score: '+sc.toFixed(0)+'/100</strong><br>Keyword Volume: '+kw.toLocaleString()+'/mo<br>Difficulty: '+df.toFixed(0)+'%<br>'+(sc>70?'🔥 Great opportunity!':sc>50?'👍 Worth pursuing':'💪 Consider easier keywords')}
function calcAB(){var ca=parseFloat(document.getElementById('ab-control').value)||0,va=parseFloat(document.getElementById('ab-variant').value)||0,vi=parseFloat(document.getElementById('ab-visitors').value)||0;var crCA=ca/vi*100,crVA=va/vi*100,im=((crVA-crCA)/crCA*100);document.getElementById('ab-result').innerHTML='<strong>A/B Test Results</strong><br>Control Rate: '+crCA.toFixed(2)+'%<br>Variant Rate: '+crVA.toFixed(2)+'%<br>Improvement: '+(im>0?'+':'')+im.toFixed(2)+'%<br>'+(Math.abs(im)>10?'🔥 Significant!':Math.abs(im)>5?'👍 Notable':'📊 Keep testing')}
function calcEmailROI(){var se=parseFloat(document.getElementById('email-sent').value)||0,op=parseFloat(document.getElementById('email-opens').value)||0,cl=parseFloat(document.getElementById('email-clicks').value)||0,rev=parseFloat(document.getElementById('email-revenue').value)||0,cost=parseFloat(document.getElementById('email-cost').value)||0;var or=op/se*100,cr=cl/op*100,roi=((rev-cost)/cost*100);document.getElementById('email-result').innerHTML='<strong>Email Campaign Analysis</strong><br>Open Rate: '+or.toFixed(2)+'%<br>Click Rate: '+cr.toFixed(2)+'%<br>Revenue: $'+rev.toFixed(2)+'<br>ROI: '+roi.toFixed(2)+'%<br>'+(roi>500?'🔥 Exceptional!':roi>200?'👍 Great':'💪 Room to improve')}
</script>`

func genToolsHub() string {
	niche := pick(niches)
	title := "Free " + niche.Name + " Micro Tools - Calculators & Optimizers"
	desc := fmt.Sprintf("Free interactive %s tools: ROI calculator, profit margin calculator, conversion rate calculator, SEO analyzer, A/B test calculator, and email ROI calculator.", niche.Name)
	tools := []struct{ Name, Desc, Icon, Link string }{
		{"ROI Calculator", "Calculate your return on investment instantly", "📊", "roi-calculator.html"},
		{"Profit Margin Calculator", "Determine your profit margins and markup", "💰", "profit-calculator.html"},
		{"Conversion Rate Calculator", "Track and optimize your conversion rates", "🎯", "conversion-calculator.html"},
		{"Lead Value Calculator", "Calculate customer lifetime value and revenue", "💎", "lead-value-calculator.html"},
		{"Budget Optimizer", "Optimize your marketing budget allocation", "📋", "budget-optimizer.html"},
		{"SEO Keyword Analyzer", "Analyze keyword difficulty and opportunity", "🔍", "seo-analyzer.html"},
		{"A/B Test Calculator", "Calculate statistical significance of tests", "🧪", "ab-test-calculator.html"},
		{"Email ROI Calculator", "Measure your email marketing ROI", "📧", "email-roi-calculator.html"},
	}
	cards := ""
	for _, t := range tools {
		cards += fmt.Sprintf(`<div class="acd" style="text-align:center"><div style="font-size:3em;margin-bottom:10px">%s</div><h4>%s</h4><p>%s</p><a href="/tools/%s" class="btn btn-p" target="_blank" style="font-size:0.9em;padding:10px 24px;margin-top:12px;display:inline-block">Use Tool →</a></div>`, t.Icon, t.Name, t.Desc, t.Link)
	}
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:50px 0 30px"><h1 style="font-size:2.5em;color:#333;margin-bottom:12px">🔧 Free %s Tools</h1><p style="font-size:1.1em;color:#666;max-width:700px;margin:0 auto">Professional calculators to optimize your results. Free to use, no signup required.</p></header><div class="ac">%s</div><div class="ag">%s</div><div class="ac">%s</div>%s<div class="ac">%s</div><div class="ac">%s</div>%s%s`,
		commonHead(title, desc), navBar(niche), niche.Name,
		adBanner(), cards,
		adInArticle(),
		dynamicEmailForm(),
		adSidebar(), adMatched(),
		cryptoDonate(),
		foot())
}

func genROICalculator() string {
	niche := pick(niches)
	title := "Free ROI Calculator - Calculate Return on Investment"
	desc := "Free online ROI calculator. Calculate your return on investment, net profit, and ROI percentage instantly."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">📊 ROI Calculator</h1><p style="color:#666;margin-top:8px">Calculate your return on investment in seconds</p></header><div class="ac">%s</div><div style="background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Total Investment ($)</label><input type="number" id="roi-invest" value="1000" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Total Return ($)</label><input type="number" id="roi-return" value="2500" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><button onclick="calcROI()" class="btn btn-p" style="font-size:1.1em;margin:15px 0">Calculate ROI</button><div id="roi-result" style="margin-top:20px;padding:20px;background:#f8f9fa;border-radius:10px;font-size:1.2em;text-align:center">Click Calculate to see your ROI</div></div><div class="ac">%s</div><p style="line-height:1.8;color:#444;margin:20px 0">Use our ROI calculator to evaluate the profitability of your investments. Understanding your return on investment helps you make data-driven decisions about where to allocate your budget for maximum impact.</p><div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:30px 0;text-align:center"><a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-p" target="_blank" rel="nofollow sponsored">Get Premium Analytics Tools →</a></div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		cryptoDonate(),
		foot())
}

func genProfitCalculator() string {
	niche := pick(niches)
	title := "Free Profit Margin Calculator - Calculate Markup & Margin"
	desc := "Free online profit margin calculator. Calculate gross margin, markup percentage, and profit per unit instantly."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">💰 Profit Margin Calculator</h1><p style="color:#666;margin-top:8px">Calculate profit margins, markup, and per-unit profit</p></header><div class="ac">%s</div><div style="background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Cost Per Unit ($)</label><input type="number" id="prof-cost" value="50" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Selling Price ($)</label><input type="number" id="prof-price" value="100" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><button onclick="calcProfit()" class="btn btn-p" style="font-size:1.1em;margin:15px 0">Calculate Margin</button><div id="prof-result" style="margin-top:20px;padding:20px;background:#f8f9fa;border-radius:10px;font-size:1.2em;text-align:center">Click Calculate to see your margin</div></div><div class="ac">%s</div><p style="line-height:1.8;color:#444;margin:20px 0">Understanding your profit margins is essential for pricing strategy and business growth. Use this calculator to determine the optimal pricing for your products and services.</p><div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:30px 0;text-align:center"><a href="https://HOPLINK.hop.clickbank.net" class="btn btn-s" target="_blank" rel="nofollow sponsored">Get Pricing Software →</a></div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		cryptoDonate(),
		foot())
}

func genConversionCalculator() string {
	niche := pick(niches)
	title := "Free Conversion Rate Calculator - Calculate CVR"
	desc := "Free online conversion rate calculator. Calculate your conversion rate percentage from visitors and conversions instantly."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">🎯 Conversion Rate Calculator</h1><p style="color:#666;margin-top:8px">Track and optimize your conversion rates</p></header><div class="ac">%s</div><div style="background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Total Visitors</label><input type="number" id="conv-visitors" value="1000" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Conversions</label><input type="number" id="conv-conversions" value="35" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><button onclick="calcConv()" class="btn btn-p" style="font-size:1.1em;margin:15px 0">Calculate Rate</button><div id="conv-result" style="margin-top:20px;padding:20px;background:#f8f9fa;border-radius:10px;font-size:1.2em;text-align:center">Click Calculate to see your rate</div></div><div class="ac">%s</div><p style="line-height:1.8;color:#444;margin:20px 0">Conversion rate optimization (CRO) is key to maximizing your results. Track your conversion rates over time and identify opportunities for improvement.</p><div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:30px 0;text-align:center"><a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" class="btn btn-w" target="_blank" rel="nofollow sponsored">Get CRO Tools →</a></div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		cryptoDonate(),
		foot())
}

func genLeadValueCalculator() string {
	niche := pick(niches)
	title := "Free Lead Value Calculator - Calculate Customer Lifetime Value"
	desc := "Free online lead value calculator. Calculate customer lifetime value, revenue per visitor, and conversion rate impact."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">💎 Lead Value Calculator</h1><p style="color:#666;margin-top:8px">Calculate customer lifetime value and revenue impact</p></header><div class="ac">%s</div><div style="background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Monthly Visitors</label><input type="number" id="lead-visitors" value="10000" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Monthly Conversions</label><input type="number" id="lead-conversions" value="200" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Avg. Customer Value ($)</label><input type="number" id="lead-value" value="500" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><button onclick="calcLead()" class="btn btn-p" style="font-size:1.1em;margin:15px 0">Calculate Value</button><div id="lead-result" style="margin-top:20px;padding:20px;background:#f8f9fa;border-radius:10px;font-size:1.2em;text-align:center">Click Calculate to see lead value</div></div><div class="ac">%s</div><p style="line-height:1.8;color:#444;margin:20px 0">Understanding your customer lifetime value helps you make informed decisions about marketing spend, customer acquisition costs, and growth strategies.</p><div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:30px 0;text-align:center"><a href="https://www.digistore24.com/redir/PRODUCT_ID" class="btn btn-p" target="_blank" rel="nofollow sponsored">Get Analytics Suite →</a></div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		cryptoDonate(),
		foot())
}

func genBudgetOptimizer() string {
	niche := pick(niches)
	title := "Free Marketing Budget Optimizer - Allocate Your Budget"
	desc := "Free online marketing budget optimizer. Get recommended budget allocation across social, search, email, content, and display channels."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">📋 Marketing Budget Optimizer</h1><p style="color:#666;margin-top:8px">Optimize your marketing budget allocation</p></header><div class="ac">%s</div><div style="background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Total Monthly Budget ($)</label><input type="number" id="budget-total" value="10000" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Primary Channel</label><select id="budget-channel" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"><option value="social">Social Media</option><option value="search">Search Ads</option><option value="email">Email Marketing</option><option value="content">Content Marketing</option><option value="display">Display Ads</option></select></div><button onclick="calcBudget()" class="btn btn-p" style="font-size:1.1em;margin:15px 0">Optimize Budget</button><div id="budget-result" style="margin-top:20px;padding:20px;background:#f8f9fa;border-radius:10px;font-size:1.2em;text-align:center">Click Optimize to see allocation</div></div><div class="ac">%s</div><p style="line-height:1.8;color:#444;margin:20px 0">Effective budget allocation is crucial for success. Our optimizer uses industry-standard benchmarks to recommend the ideal distribution across channels based on your total budget and primary focus area.</p><div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:30px 0;text-align:center"><a href="https://warriorplus.com/buy/PRODUCT_ID" class="btn btn-w" target="_blank" rel="nofollow sponsored">Get Budgeting Templates →</a></div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		cryptoDonate(),
		foot())
}

func genSEOAnalyzer() string {
	niche := pick(niches)
	title := "Free SEO Keyword Analyzer - Check Keyword Difficulty"
	desc := "Free SEO keyword analyzer. Check keyword difficulty score, search volume analysis, and opportunity rating."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">🔍 SEO Keyword Analyzer</h1><p style="color:#666;margin-top:8px">Analyze keyword difficulty and opportunity</p></header><div class="ac">%s</div><div style="background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Search Volume (monthly)</label><input type="number" id="seo-volume" value="5000" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Keyword Difficulty (0-100)</label><input type="number" id="seo-difficulty" value="45" min="0" max="100" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Current Clicks/mo (optional)</label><input type="number" id="seo-clicks" value="0" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><button onclick="calcSEO()" class="btn btn-p" style="font-size:1.1em;margin:15px 0">Analyze Keyword</button><div id="seo-result" style="margin-top:20px;padding:20px;background:#f8f9fa;border-radius:10px;font-size:1.2em;text-align:center">Click Analyze to see your SEO score</div></div><div class="ac">%s</div><p style="line-height:1.8;color:#444;margin:20px 0">Our SEO keyword analyzer helps you evaluate keyword opportunities by combining search volume, difficulty score, and current performance into a single actionable metric. Higher scores mean better opportunities.</p><div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:30px 0;text-align:center"><a href="https://www.amazon.com/dp/B0BXYZ?tag=emeraldeng0e-20" class="btn btn-p" target="_blank" rel="nofollow sponsored">Get SEO Tools →</a></div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		cryptoDonate(),
		foot())
}

func genABTestCalculator() string {
	niche := pick(niches)
	title := "Free A/B Test Calculator - Statistical Significance"
	desc := "Free A/B test significance calculator. Determine if your test results are statistically significant."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">🧪 A/B Test Calculator</h1><p style="color:#666;margin-top:8px">Calculate statistical significance of your tests</p></header><div class="ac">%s</div><div style="background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Control Conversions</label><input type="number" id="ab-control" value="45" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Variant Conversions</label><input type="number" id="ab-variant" value="62" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Total Visitors Per Version</label><input type="number" id="ab-visitors" value="1000" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><button onclick="calcAB()" class="btn btn-p" style="font-size:1.1em;margin:15px 0">Calculate Significance</button><div id="ab-result" style="margin-top:20px;padding:20px;background:#f8f9fa;border-radius:10px;font-size:1.2em;text-align:center">Click Calculate to see significance</div></div><div class="ac">%s</div><p style="line-height:1.8;color:#444;margin:20px 0">Run A/B tests with confidence. Our calculator shows the conversion rate for each variant and the improvement percentage. Use this data to decide which version performs better.</p><div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:30px 0;text-align:center"><a href="https://HOPLINK.hop.clickbank.net" class="btn btn-s" target="_blank" rel="nofollow sponsored">Get Testing Tools →</a></div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		cryptoDonate(),
		foot())
}

func genEmailROICalculator() string {
	niche := pick(niches)
	title := "Free Email ROI Calculator - Measure Campaign Performance"
	desc := "Free email marketing ROI calculator. Measure your email campaign performance, open rates, click rates, and return on investment."
	return fmt.Sprintf(`%s%s<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">📧 Email ROI Calculator</h1><p style="color:#666;margin-top:8px">Measure your email marketing performance</p></header><div class="ac">%s</div><div style="background:white;border:1px solid #e0e0e0;border-radius:16px;padding:35px;margin:25px 0;box-shadow:0 4px 15px rgba(0,0,0,0.05)"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Emails Sent</label><input type="number" id="email-sent" value="10000" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Opens</label><input type="number" id="email-opens" value="2500" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Clicks</label><input type="number" id="email-clicks" value="450" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Revenue Generated ($)</label><input type="number" id="email-revenue" value="5000" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Campaign Cost ($)</label><input type="number" id="email-cost" value="500" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1.1em"></div><button onclick="calcEmailROI()" class="btn btn-p" style="font-size:1.1em;margin:15px 0">Calculate ROI</button><div id="email-result" style="margin-top:20px;padding:20px;background:#f8f9fa;border-radius:10px;font-size:1.2em;text-align:center">Click Calculate to see email ROI</div></div><div class="ac">%s</div><p style="line-height:1.8;color:#444;margin:20px 0">Email marketing consistently delivers the highest ROI of any channel. Use our calculator to measure your campaign performance and identify areas for improvement.</p><div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:30px 0;text-align:center"><a href="https://www.shareasale.com/r.cfm?b=ID&u=USER_ID&m=MERCHANT_ID" class="btn btn-w" target="_blank" rel="nofollow sponsored">Get Email Tools →</a></div>%s%s`,
		commonHead(title, desc), navBar(niche),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		cryptoDonate(),
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

func setupGitCredentials() {
	ghToken := vaultGet("GITHUB_TOKEN", "")
	hfToken := vaultGet("HF_TOKEN", "")
	ghUser := vaultGet("GITHUB_USER", "Alexander101001")
	hfUser := vaultGet("HF_USER", "AlexanderGreater90")

	os.MkdirAll("/root", 0755)
	var netrcLines string

	if ghToken != "" {
		netrcLines += fmt.Sprintf("machine github.com\nlogin %s\npassword %s\n", ghUser, ghToken)
		fmt.Printf("[GIT] GitHub credentials configured\n")
	}

	if hfToken != "" {
		netrcLines += fmt.Sprintf("machine huggingface.co\nlogin %s\npassword %s\n", hfUser, hfToken)
		fmt.Printf("[GIT] HF credentials configured\n")

		hfURL := fmt.Sprintf("https://%s:%s@huggingface.co/spaces/%s/emerald-engine", hfUser, hfToken, hfUser)
		exec.Command("git", "remote", "set-url", "origin", hfURL).Run()
		exec.Command("git", "remote", "set-url", "huggingface", hfURL).Run()
		exec.Command("git", "remote", "add", "github", fmt.Sprintf("https://%s:%s@github.com/%s/emerald-engine.git", ghUser, ghToken, ghUser)).Run()
	}

	if netrcLines != "" {
		os.WriteFile("/root/.netrc", []byte(netrcLines), 0600)
		exec.Command("chmod", "600", "/root/.netrc").Run()
	}
}

// ─── Main ───

func main() {
	rand.Seed(time.Now().UnixNano())
	vault = loadVault()
	initLLMProviders()

	db := loadFulfillmentDB("emerald_sales.json")
	startWebhookServer(db)
	initTradingPlatform()
	setupGitCredentials()

	initOrchestrator()
	initResourceManager()
	startCognitiveLoop()

	fmt.Printf("[ENGINE] Emerald Engine v6.0 — Cross-Platform Factory\n")
	fmt.Printf("[ENGINE] Cycle: %ds | LLM: %d | Factory: %s | Cognitive: active\n",
		cycleDelay, len(llmProviders),
		func() string {
			if orchestrator != nil && orchestrator.Token != "" { return "active" }; return "disabled"
		}(),
	)

	for _, dir := range []string{"public/blog", "public/tools", "public/downloads", "public/trading"} {
		os.MkdirAll(dir, 0755)
	}

	outputFiles := []string{
		"public/index.html", "public/compare.html", "public/review.html",
		"public/resources.html", "public/products.html",
		"public/blog/getting-started.html",
		"public/tools/index.html", "public/tools/roi-calculator.html",
		"public/tools/profit-calculator.html", "public/tools/conversion-calculator.html",
		"public/tools/lead-value-calculator.html", "public/tools/budget-optimizer.html",
		"public/tools/seo-analyzer.html", "public/tools/ab-test-calculator.html",
		"public/tools/email-roi-calculator.html",
		"public/trading/index.html", "public/trading/register.html",
		"public/trading/login.html", "public/trading/dashboard.html",
		"public/trading/grid.html", "public/trading/signals.html",
		"public/trading/copytrade.html",
		"public/about.html", "public/contact.html", "public/privacy.html",
		"public/terms.html", "public/disclosure.html",
		"public/sitemap.xml", "public/robots.txt",
	}

	for i := 1; ; i++ {
		fmt.Printf("\n[ENGINE] === Cycle %d ===\n", i)
		start := time.Now()
		total := 0

		niche := pick(niches)
		fmt.Printf("[ENGINE] Niche: %s %s\n", niche.Emoji, niche.Name)

		maybeEnsureSubdomain(niche.Keyword)

		tradingPages := genTradingPages()
		pages := map[string]func() string{
			"public/index.html":                        func() string { return genIndex(niche) },
			"public/compare.html":                      genCompare,
			"public/review.html":                       genReview,
			"public/resources.html":                    genResources,
			"public/products.html":                     genProductsPage,
			"public/blog/getting-started.html":         genBlog,
			"public/tools/index.html":                  genToolsHub,
			"public/tools/roi-calculator.html":         genROICalculator,
			"public/tools/profit-calculator.html":      genProfitCalculator,
			"public/tools/conversion-calculator.html":  genConversionCalculator,
			"public/tools/lead-value-calculator.html":  genLeadValueCalculator,
			"public/tools/budget-optimizer.html":       genBudgetOptimizer,
			"public/tools/seo-analyzer.html":           genSEOAnalyzer,
			"public/tools/ab-test-calculator.html":     genABTestCalculator,
			"public/tools/email-roi-calculator.html":   genEmailROICalculator,
			"public/trading/index.html":                tradingPages["public/trading/index.html"],
			"public/trading/register.html":             tradingPages["public/trading/register.html"],
			"public/trading/login.html":                tradingPages["public/trading/login.html"],
			"public/trading/dashboard.html":            tradingPages["public/trading/dashboard.html"],
			"public/trading/grid.html":                 tradingPages["public/trading/grid.html"],
			"public/trading/signals.html":              tradingPages["public/trading/signals.html"],
			"public/trading/copytrade.html":            tradingPages["public/trading/copytrade.html"],
			"public/about.html":                        genAbout,
			"public/contact.html":                      genContact,
			"public/privacy.html":                      genPrivacy,
			"public/terms.html":                        genTerms,
			"public/disclosure.html":                   genDisclosure,
		}

		for path, gen := range pages {
			html := gen()
			os.MkdirAll(filepath.Dir(path), 0755)
			os.WriteFile(path, []byte(html), 0644)
			total += len(html)
			fmt.Printf("[ENGINE] ✓ %s (%d bytes)\n", path, len(html))
		}

		allPages := []string{
			"index.html", "compare.html", "review.html",
			"resources.html", "products.html", "blog/getting-started.html",
			"tools/index.html", "tools/roi-calculator.html",
			"tools/profit-calculator.html", "tools/conversion-calculator.html",
			"tools/lead-value-calculator.html", "tools/budget-optimizer.html",
			"tools/seo-analyzer.html", "tools/ab-test-calculator.html",
			"tools/email-roi-calculator.html",
			"trading/index.html", "trading/register.html",
			"trading/login.html", "trading/dashboard.html",
			"trading/grid.html", "trading/signals.html",
			"trading/copytrade.html",
			"about.html", "contact.html", "privacy.html",
			"terms.html", "disclosure.html",
		}
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
		revenue := db.totalRevenue()
		telegramSend(fmt.Sprintf(
			"<b>🤖 Emerald Engine v5.0</b>\n%s\nNiche: %s %s\nSales: %d | Revenue: $%.2f\nRemotes: origin + huggingface",
			summary, niche.Emoji, niche.Name, len(db.Sales), revenue,
		))

		sleep := cycleDelay*time.Second - elapsed
		if sleep > 0 {
			fmt.Printf("[ENGINE] Done in %v. Next in %v\n", elapsed, sleep)
			time.Sleep(sleep)
		} else {
			fmt.Printf("[ENGINE] Done in %v (exceeded interval)\n", elapsed)
		}
	}
}
