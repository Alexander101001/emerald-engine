package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"
)

const (
	outputFile = "public/index.html"
	cycleDelay = 300
)

type Template struct {
	Title       string
	Headline    string
	Subheadline string
	Features    []Feature
	CTAText     string
	Year        int
	Primary     string
	Secondary   string
	Accent      string
}

type Feature struct {
	Title       string
	Description string
	Icon        string
}

var colorSchemes = [][]string{
	{"#667eea", "#764ba2", "#ff6b6b"},
	{"#11998e", "#38ef7d", "#f39c12"},
	{"#fc5c7d", "#6a82fb", "#2ecc71"},
	{"#0c3483", "#a2b6df", "#e74c3c"},
	{"#2c3e50", "#3498db", "#e67e22"},
	{"#8e2de2", "#4a00e0", "#00b894"},
}

var headlines = []string{
	"AI-Powered Marketing Suite",
	"Smart Automation Platform",
	"Next-Gen Analytics Engine",
	"Growth Intelligence System",
	"Conversion Optimization Hub",
}

var subheadlines = []string{
	"Automate campaigns, boost engagement, maximize ROI with cutting-edge AI",
	"Transform your data into revenue with autonomous optimization",
	"Let AI handle the heavy lifting while you focus on growth",
	"Enterprise-grade analytics powered by machine learning",
	"Turn visitors into customers with intelligent automation",
}

var ctaTexts = []string{
	"Start Free Trial",
	"Get Started Now",
	"Claim Your Account",
	"Launch Your Journey",
	"Begin Free Today",
}

var featureSets = [][]Feature{
	{
		{"Smart Campaigns", "AI-driven optimization delivering the right message at the perfect time", "🎯"},
		{"Real-Time Analytics", "Comprehensive dashboards with actionable ML-powered insights", "📊"},
		{"Automated A/B Testing", "Continuous optimization across all channels with zero effort", "🧪"},
	},
	{
		{"Predictive Scoring", "Identify high-value leads before they convert", "⚡"},
		{"Multi-Channel Orchestration", "Coordinate campaigns across email, social, and web", "🔄"},
		{"Revenue Attribution", "Track every dollar back to its source automatically", "💰"},
	},
	{
		{"Sentiment Analysis", "Understand customer emotions at scale", "💡"},
		{"Behavioral Segmentation", "Group users by real actions, not guesses", "👥"},
		{"Dynamic Pricing", "Optimize prices in real-time based on demand", "🏷️"},
	},
}

var featureTitles = []string{
	"Smart Campaigns",
	"Real-Time Analytics",
	"Automated A/B Testing",
	"Predictive Scoring",
	"Multi-Channel Orchestration",
	"Revenue Attribution",
	"Sentiment Analysis",
	"Behavioral Segmentation",
	"Dynamic Pricing",
}

var featureDescs = []string{
	"AI-driven optimization delivering the right message at the perfect time",
	"Comprehensive dashboards with actionable ML-powered insights",
	"Continuous optimization across all channels with zero effort",
	"Identify high-value leads before they convert",
	"Coordinate campaigns across email, social, and web",
	"Track every dollar back to its source automatically",
	"Understand customer emotions at scale",
	"Group users by real actions, not guesses",
	"Optimize prices in real-time based on demand",
}

func pick[T any](items []T) T {
	return items[rand.Intn(len(items))]
}

func pickN[T any](items []T, n int) []T {
	perm := rand.Perm(len(items))
	result := make([]T, n)
	for i := 0; i < n; i++ {
		result[i] = items[perm[i]]
	}
	return result
}

func generatePage() string {
	scheme := pick(colorSchemes)
	title := pick(headlines)
	features := pickN(featureTitles, 3)
	descs := pickN(featureDescs, 3)

	featureHTML := ""
	for i := 0; i < 3; i++ {
		featureHTML += fmt.Sprintf(`
            <div class="feature-card">
                <h3>%s</h3>
                <p>%s</p>
            </div>`, features[i], descs[i])
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Boost Your ROI</title>
    <script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-XXXXXXXXXXXXXXXX"
        crossorigin="anonymous"></script>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, %s 0%%, %s 100%%);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
        }
        .container {
            max-width: 1200px;
            width: 100%%;
            padding: 20px;
        }
        header {
            text-align: center;
            padding: 60px 0 40px;
            color: white;
        }
        header h1 {
            font-size: 3em;
            margin-bottom: 20px;
        }
        header p {
            font-size: 1.3em;
            opacity: 0.9;
        }
        .ad-banner {
            background: rgba(255, 255, 255, 0.1);
            padding: 20px;
            border-radius: 10px;
            text-align: center;
            margin: 30px 0;
            min-height: 90px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: rgba(255, 255, 255, 0.5);
            font-size: 1.1em;
        }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 30px;
            margin: 50px 0;
        }
        .feature-card {
            background: white;
            padding: 40px;
            border-radius: 15px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
            transition: transform 0.3s;
        }
        .feature-card:hover {
            transform: translateY(-5px);
        }
        .feature-card h3 {
            color: #333;
            margin-bottom: 15px;
            font-size: 1.5em;
        }
        .feature-card p {
            color: #666;
            line-height: 1.6;
        }
        .cta-button {
            display: inline-block;
            padding: 18px 50px;
            background: %s;
            color: white;
            text-decoration: none;
            border-radius: 50px;
            font-size: 1.2em;
            font-weight: bold;
            transition: background 0.3s, transform 0.3s;
            margin: 40px 0;
        }
        .cta-button:hover {
            background: %s;
            transform: scale(1.05);
        }
        .ad-sidebar {
            background: rgba(255, 255, 255, 0.1);
            padding: 20px;
            border-radius: 10px;
            text-align: center;
            margin: 30px 0;
            min-height: 250px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: rgba(255, 255, 255, 0.5);
            font-size: 1.1em;
        }
        footer {
            color: rgba(255, 255, 255, 0.7);
            padding: 40px 0;
            text-align: center;
            width: 100%%;
        }
        .badge {
            display: inline-block;
            background: rgba(255,255,255,0.15);
            color: white;
            padding: 6px 16px;
            border-radius: 20px;
            font-size: 0.85em;
            margin-bottom: 20px;
        }
        @media (max-width: 768px) {
            header h1 {
                font-size: 2em;
            }
            header p {
                font-size: 1em;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="badge">&#9733; AI-Powered</div>
            <h1>%s</h1>
            <p>%s</p>
        </header>

        <div class="ad-banner">
            <ins class="adsbygoogle"
                style="display:block"
                data-ad-client="ca-pub-XXXXXXXXXXXXXXXX"
                data-ad-slot="XXXXXXXXXX"
                data-ad-format="auto"
                data-full-width-responsive="true"></ins>
            <script>(adsbygoogle = window.adsbygoogle || []).push({});</script>
        </div>

        <div class="features">
            %s
        </div>

        <div style="text-align: center;">
            <a href="#" class="cta-button">%s</a>
        </div>

        <div class="ad-sidebar">
            <ins class="adsbygoogle"
                style="display:block"
                data-ad-client="ca-pub-XXXXXXXXXXXXXXXX"
                data-ad-slot="XXXXXXXXXX"
                data-ad-format="auto"
                data-full-width-responsive="true"></ins>
            <script>(adsbygoogle = window.adsbygoogle || []).push({});</script>
        </div>

        <footer>
            <p>&copy; %d Emerald Engine. All rights reserved.</p>
            <p style="margin-top:8px;font-size:0.8em;opacity:0.6;">Generated autonomously by Emerald Engine v2.0</p>
        </footer>
    </div>
</body>
</html>`, title, scheme[0], scheme[1], scheme[2], scheme[2],
		title, pick(subheadlines), featureHTML,
		pick(ctaTexts), time.Now().Year())
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitCommitAndPush() error {
	if err := runCmd("git", "add", outputFile); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	ts := time.Now().Format(time.RFC3339)
	if err := runCmd("git", "commit", "--allow-empty", "-m", fmt.Sprintf("auto: regenerate landing page [%s]", ts)); err != nil {
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
	fmt.Println("[ENGINE] Starting Emerald Engine v2.0 (Go)")
	fmt.Printf("[ENGINE] Cycle interval: %ds\n", cycleDelay)

	if err := os.MkdirAll("public", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[ENGINE] mkdir public: %v\n", err)
		os.Exit(1)
	}

	for i := 1; ; i++ {
		fmt.Printf("\n[ENGINE] === Cycle %d ===\n", i)
		start := time.Now()

		html := generatePage()
		if err := os.WriteFile(outputFile, []byte(html), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[ENGINE] write failed: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Printf("[ENGINE] Generated %s (%d bytes)\n", outputFile, len(html))

		if err := gitCommitAndPush(); err != nil {
			fmt.Fprintf(os.Stderr, "[ENGINE] git error: %v\n", err)
		}

		elapsed := time.Since(start)
		sleep := cycleDelay*time.Second - elapsed
		if sleep > 0 {
			fmt.Printf("[ENGINE] Cycle done in %v. Next cycle in %v\n", elapsed, sleep)
			time.Sleep(sleep)
		} else {
			fmt.Printf("[ENGINE] Cycle took %v (exceeded interval)\n", elapsed)
		}
	}
}
