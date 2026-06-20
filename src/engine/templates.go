package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type LayoutVariant struct {
	HeaderStyle  string
	CardStyle    string
	ButtonStyle  string
	FontPair     [2]string
	BorderRadius string
	Shadow       string
}

type TemplateEngine struct {
	variants []LayoutVariant
}

var templateEngine *TemplateEngine

func initTemplateEngine() *TemplateEngine {
	te := &TemplateEngine{
		variants: []LayoutVariant{
			{HeaderStyle: "gradient-header", CardStyle: "elevated-card", ButtonStyle: "pill-btn", FontPair: [2]string{"Inter", "system-ui"}, BorderRadius: "12px", Shadow: "0 4px 24px rgba(0,0,0,0.08)"},
			{HeaderStyle: "solid-header", CardStyle: "bordered-card", ButtonStyle: "sharp-btn", FontPair: [2]string{"Poppins", "sans-serif"}, BorderRadius: "8px", Shadow: "0 2px 12px rgba(0,0,0,0.06)"},
			{HeaderStyle: "minimal-header", CardStyle: "ghost-card", ButtonStyle: "underline-btn", FontPair: [2]string{"DM Sans", "sans-serif"}, BorderRadius: "4px", Shadow: "none"},
			{HeaderStyle: "glass-header", CardStyle: "glass-card", ButtonStyle: "glow-btn", FontPair: [2]string{"Outfit", "system-ui"}, BorderRadius: "16px", Shadow: "0 8px 32px rgba(102,126,234,0.15)"},
			{HeaderStyle: "hero-header", CardStyle: "split-card", ButtonStyle: "outline-btn", FontPair: [2]string{"Lexend", "sans-serif"}, BorderRadius: "10px", Shadow: "0 6px 20px rgba(0,0,0,0.1)"},
		},
	}
	templateEngine = te
	return te
}

func (te *TemplateEngine) Variant(seed int) LayoutVariant {
	return te.variants[seed%len(te.variants)]
}

func (te *TemplateEngine) RandomVariant() LayoutVariant {
	n := time.Now().UnixNano()
	return te.variants[n%int64(len(te.variants))]
}

func GenerateSVGLogo(niche Niche, seed int) string {
	r := rand.New(rand.NewSource(int64(seed)))
	shape := r.Intn(4)
	color := colorSchemes[seed%len(colorSchemes)]

	symbols := []string{
		`<circle cx="24" cy="24" r="20" fill="%s" opacity="0.9"/><text x="24" y="30" text-anchor="middle" fill="white" font-size="18" font-weight="bold">%s</text>`,
		`<rect x="4" y="4" width="40" height="40" rx="10" fill="%s"/><text x="24" y="30" text-anchor="middle" fill="white" font-size="20">%s</text>`,
		`<polygon points="24,4 44,40 4,40" fill="%s" opacity="0.9"/><text x="24" y="32" text-anchor="middle" fill="white" font-size="16" font-weight="bold">%s</text>`,
		`<path d="M4 24 Q24 4 44 24 Q24 44 4 24" fill="%s"/><text x="24" y="28" text-anchor="middle" fill="white" font-size="16" font-weight="bold">%s</text>`,
	}

	emoji := niche.Emoji
	if len(emoji) > 2 {
		emoji = string([]rune(niche.Emoji)[0])
	}

	symbol := fmt.Sprintf(symbols[shape], color[0], emoji)

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48" width="48" height="48">%s</svg>`, symbol)
}

func GenerateSVGIcon(name string, seed int) string {
	r := rand.New(rand.NewSource(int64(seed)))
	colors := []string{"#667eea", "#764ba2", "#f093fb", "#4facfe", "#43e97b", "#fa709a", "#a18cd1", "#fbc2eb"}

	icons := map[string]string{
		"rocket":  `<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-12h2v6h-2zm0-4h2v2h-2z" fill="%s"/>`,
		"chart":   `<path d="M5 9.2h3V19H5V9.2zM10.6 5h3v14h-3V5zm5.6 8H19v6h-3v-6z" fill="%s"/>`,
		"globe":   `<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z" fill="%s"/>`,
		"star":    `<path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" fill="%s"/>`,
		"bolt":    `<path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" fill="%s"/>`,
	}

	path, ok := icons[name]
	if !ok {
		path = icons["star"]
	}

	c := colors[r.Intn(len(colors))]
	svg := fmt.Sprintf(path, c)

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24">%s</svg>`, svg)
}

func GenerateLayoutCSS(variant LayoutVariant, niche Niche) string {
	colors := variantStyles(time.Now().UnixNano(), niche)

	return fmt.Sprintf(`
:root{
  --primary:%s;--secondary:%s;--accent:%s;
  --radius:%s;--shadow:%s;
  --font-heading:'%s';--font-body:'%s';
}
body{font-family:var(--font-body);margin:0;padding:0;background:#fafafa;color:#333}
.container{max-width:1200px;margin:0 auto;padding:0 20px}
.header.%s{background:linear-gradient(135deg,var(--primary),var(--secondary));color:#fff;padding:40px 0 60px}
.card.%s{background:#fff;border-radius:var(--radius);box-shadow:var(--shadow);padding:24px;margin:16px 0}
.btn.%s{background:var(--primary);color:#fff;padding:12px 32px;border:none;border-radius:%s;cursor:pointer;font-weight:600;transition:all .2s}
.btn.%s:hover{transform:translateY(-2px);box-shadow:0 8px 24px rgba(0,0,0,0.15)}
@media(max-width:768px){.container{padding:0 12px}.header.%s{padding:24px 0 40px}}
`,
		colors[0], colors[1], colors[2],
		variant.BorderRadius, variant.Shadow,
		variant.FontPair[0], variant.FontPair[1],
		variant.HeaderStyle, variant.CardStyle,
		variant.ButtonStyle, variant.BorderRadius,
		variant.ButtonStyle,
		variant.HeaderStyle,
	)
}

func GenerateNicheHero(niche Niche, variant LayoutVariant) string {
	colors := pick(colorSchemes)
	return fmt.Sprintf(`
<section class="hero %s" style="background:linear-gradient(135deg,%s,%s);padding:60px 0;text-align:center;color:#fff">
  <div class="container">
    <h1 style="font-size:2.5em;margin:0 0 16px;font-family:var(--font-heading)">%s %s — Premium Solutions</h1>
    <p style="font-size:1.2em;opacity:0.9;max-width:600px;margin:0 auto">Comprehensive %s tools and resources for modern professionals</p>
    <a href="/products" class="btn %s" style="margin-top:24px;display:inline-block;text-decoration:none">Get Started →</a>
  </div>
</section>`,
		variant.HeaderStyle, colors[0], colors[1],
		niche.Emoji, niche.Name,
		niche.Keyword,
		variant.ButtonStyle,
	)
}

func variantStyles(seed int64, niche Niche) []string {
	base := pick(colorSchemes)
	shift := 0
	return []string{
		shiftHue(base[0], shift),
		shiftHue(base[1], shift),
		base[2],
	}
}

func shiftHue(hex string, shift int) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return hex
	}
	return fmt.Sprintf("#%s", hex)
}

func GenerateSEOIntro(niche Niche) string {
	intros := []string{
		fmt.Sprintf("Looking for the best %s solutions? You're in the right place. Our platform offers comprehensive tools and insights designed specifically for %s.", niche.Name, niche.Keyword),
		fmt.Sprintf("Welcome to the ultimate resource for %s. Whether you're a beginner or an expert, our curated tools help you achieve more with less effort.", niche.Keyword),
		fmt.Sprintf("Discover how %s can transform your workflow. We've assembled everything you need to succeed in the competitive %s landscape.", niche.Name, niche.Keyword),
	}
	idx := time.Now().UnixNano() % int64(len(intros))
	return intros[idx]
}
