package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Zero-Touch Monetization Engine
// Auto-injects ad codes, auto-stakes via Binance API,
// integrates crypto payment buttons — all without manual
// account creation beyond instant self-serve signups.

type MonetizationStream struct {
	Source string  `json:"source"`
	Amount float64 `json:"amount"`
	Time   string  `json:"time"`
}

type StakedPosition struct {
	Asset    string  `json:"asset"`
	Amount   float64 `json:"amount"`
	APR      float64 `json:"apr"`
	EstDaily float64 `json:"est_daily"`
}

type ZeroTouchMonetizer struct {
	mu sync.Mutex

	adsterraID string
	propellerID string

	binKey    string
	binSecret string

	stream       []MonetizationStream
	positions    []StakedPosition
	totalEarned  float64
	startedAt    time.Time
	activeAssets int
}

var zeroTouch *ZeroTouchMonetizer

func initZeroTouchMonetizer() *ZeroTouchMonetizer {
	zt := &ZeroTouchMonetizer{
		adsterraID:  vaultGet("ADSTERRA_PUBLISHER_ID", ""),
		propellerID: vaultGet("PROPELLERADS_PUBLISHER_ID", ""),
		binKey:      vaultGet("BINANCE_API_KEY", ""),
		binSecret:   vaultGet("BINANCE_SECRET_KEY", ""),
		startedAt:   time.Now(),
	}
	zeroTouch = zt

	fmt.Printf("[monetizer] Zero-Touch Monetization Engine started\n")
	fmt.Printf("[monetizer]   Adsterra: %s\n", map[bool]string{true: "READY", false: "NO ID (add ADSTERRA_PUBLISHER_ID to vault)"}[zt.adsterraID != ""])
	fmt.Printf("[monetizer]   PropellerAds: %s\n", map[bool]string{true: "READY", false: "NO ID (add PROPELLERADS_PUBLISHER_ID to vault)"}[zt.propellerID != ""])
	fmt.Printf("[monetizer]   Binance API: %s\n", map[bool]string{true: "READY", false: "NO KEY"}[zt.binKey != ""])

	if zt.binKey != "" {
		go zt.binanceAutoEarnLoop()
	}

	return zt
}

// ========== Ad Code Injection ==========

func (zt *ZeroTouchMonetizer) InjectAdCodes(html string) string {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	var sb strings.Builder

	// Adsterra popunder (most profitable, instant approval)
	if zt.adsterraID != "" {
		sb.WriteString(fmt.Sprintf(`
<script type="text/javascript">
	(function(d,s) {
		var f=d.getElementsByTagName(s)[0], j=d.createElement(s);
		j.async=true; j.src='https://a.pub.network/%s/pub.js';
		f.parentNode.insertBefore(j,f);
	})(document,'script');
</script>
<noscript><img src="https://a.pub.network/%s/pixel.png" style="display:none"/></noscript>`, zt.adsterraID, zt.adsterraID))
	}

	// PropellerAds push notifications
	if zt.propellerID != "" {
		sb.WriteString(fmt.Sprintf(`
<script type="text/javascript">
	(function(d) {
		var f=d.getElementsByTagName('script')[0], s=d.createElement('script');
		s.async=true; s.src='https://%s.push.world/init.js';
		f.parentNode.insertBefore(s,f);
	})(document);
</script>`, zt.propellerID))
	}

	// Binance Pay donate button
	if zt.binKey != "" {
		sb.WriteString(`
<style>.bpay-btn{display:inline-block;background:#f0b90b;color:#1e2329;padding:10px 24px;border-radius:8px;font-weight:700;text-decoration:none;margin:8px 0}</style>
<a class="bpay-btn" href="https://www.binance.com/en/pay/store/merchant/emerald-engine" target="_blank">Donate with Binance Pay</a>`)
	}

	if sb.Len() == 0 {
		return html
	}

	// Inject before </head>
	headEnd := strings.LastIndex(html, "</head>")
	if headEnd == -1 {
		return html
	}
	return html[:headEnd] + sb.String() + html[headEnd:]
}

func (zt *ZeroTouchMonetizer) InjectAdCodeToAllPages(pages []PageOutput) []PageOutput {
	for i, p := range pages {
		pages[i].HTML = zt.InjectAdCodes(p.HTML)
	}
	return pages
}

// ========== Binance API Helpers ==========

func (zt *ZeroTouchMonetizer) binanceSignature(queryString string) string {
	mac := hmac.New(sha256.New, []byte(zt.binSecret))
	mac.Write([]byte(queryString))
	return hex.EncodeToString(mac.Sum(nil))
}

func (zt *ZeroTouchMonetizer) binanceRequest(method, path string, params map[string]string) ([]byte, error) {
	base := "https://api.binance.com"
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	query.Set("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	sig := zt.binanceSignature(query.Encode())
	query.Set("signature", sig)

	full := fmt.Sprintf("%s%s?%s", base, path, query.Encode())
	req, err := http.NewRequest(method, full, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", zt.binKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ========== Binance Auto-Earn ==========

func (zt *ZeroTouchMonetizer) binanceAutoEarnLoop() {
	// Check every 6h for available balances to stake
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// Initial check after 30s delay for vault readiness
	time.Sleep(30 * time.Second)
	zt.binanceEarnCycle()

	for range ticker.C {
		zt.binanceEarnCycle()
	}
}

func (zt *ZeroTouchMonetizer) binanceEarnCycle() {
	if zt.binKey == "" || zt.binSecret == "" {
		return
	}

	fmt.Printf("[monetizer] Binance auto-earn cycle\n")

	// 1. Get account balances
	data, err := zt.binanceRequest("GET", "/sapi/v1/capital/config/getall", nil)
	if err != nil {
		fmt.Printf("[monetizer] balance fetch error: %v\n", err)
		zt.logStream("binance_earn", 0, fmt.Sprintf("balance_error: %v", err))
		return
	}

	var balances []struct {
		Coin      string `json:"coin"`
		Free      string `json:"free"`
		Locked    string `json:"locked"`
	}
	if json.Unmarshal(data, &balances) != nil {
		fmt.Printf("[monetizer] balance decode error\n")
		return
	}

	// 2. Find available assets above minimum threshold
	for _, b := range balances {
		free := parseFloat(b.Free)
		if free < 0.001 {
			continue
		}
		// Try to subscribe to Simple Earn flexible product
		zt.tryStakeAsset(b.Coin, free)
	}

	// 3. Query current positions
	zt.refreshPositions()
}

func (zt *ZeroTouchMonetizer) tryStakeAsset(asset string, amount float64) {
	// Subscribe to flexible Simple Earn product
	params := map[string]string{
		"productId": fmt.Sprintf("%s001", asset),
		"amount":    fmt.Sprintf("%.8f", amount),
	}
	data, err := zt.binanceRequest("POST", "/sapi/v1/simple-earn/flexible/subscribe", params)
	if err != nil {
		fmt.Printf("[monetizer] stake %s error: %v\n", asset, err)
		return
	}

	var result struct {
		Success    bool   `json:"success"`
		PurchaseId string `json:"purchaseId"`
	}
	if json.Unmarshal(data, &result) == nil && result.Success {
		fmt.Printf("[monetizer] Staked %.4f %s → Simple Earn (ID: %s)\n", amount, asset, result.PurchaseId)
		zt.logStream("binance_stake", amount, fmt.Sprintf("staked %.4f %s", amount, asset))
		zt.activeAssets++
	}
}

func (zt *ZeroTouchMonetizer) refreshPositions() {
	data, err := zt.binanceRequest("GET", "/sapi/v1/simple-earn/flexible/position", nil)
	if err != nil {
		return
	}

	var resp struct {
		Rows []struct {
			Asset        string `json:"asset"`
			TotalAmount  string `json:"totalAmount"`
			AnnualAPR    string `json:"annualAPR"`
			DailyEarning string `json:"dailyEarning"`
		} `json:"rows"`
	}
	if json.Unmarshal(data, &resp) != nil {
		return
	}

	zt.mu.Lock()
	zt.positions = nil
	var totalDaily float64
	for _, r := range resp.Rows {
		daily := parseFloat(r.DailyEarning)
		totalDaily += daily
		zt.positions = append(zt.positions, StakedPosition{
			Asset:    r.Asset,
			Amount:   parseFloat(r.TotalAmount),
			APR:      parseFloat(r.AnnualAPR),
			EstDaily: daily,
		})
	}

	if totalDaily > 0 {
		zt.totalEarned += totalDaily
		zt.logStream("binance_interest", totalDaily, fmt.Sprintf("daily interest: %.8f across %d assets", totalDaily, len(resp.Rows)))
	}
	zt.mu.Unlock()
}

func (zt *ZeroTouchMonetizer) logStream(source string, amount float64, desc string) {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	zt.stream = append(zt.stream, MonetizationStream{
		Source: source,
		Amount: amount,
		Time:   time.Now().UTC().Format(time.RFC3339),
	})
	if len(zt.stream) > 1000 {
		zt.stream = zt.stream[len(zt.stream)-500:]
	}

	if archivistAgent != nil {
		archivistAgent.ArchiveExperience("ZeroTouchMonetizer", source, desc)
	}
}

func (zt *ZeroTouchMonetizer) Stats() map[string]interface{} {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	adsterraReady := zt.adsterraID != ""
	propellerReady := zt.propellerID != ""
	binanceReady := zt.binKey != "" && zt.binSecret != ""

	var totalDaily float64
	for _, p := range zt.positions {
		totalDaily += p.EstDaily
	}

	return map[string]interface{}{
		"adsterra_ready":     adsterraReady,
		"propellerads_ready": propellerReady,
		"binance_ready":      binanceReady,
		"active_assets":      zt.activeAssets,
		"positions":          zt.positions,
		"est_daily_interest": totalDaily,
		"total_earned":       zt.totalEarned,
		"stream_events":      len(zt.stream),
		"uptime":             time.Since(zt.startedAt).String(),
		"setup_required": map[string]string{
			"adsterra":  "Sign up at adsterra.com (5min, instant approval) → add ADSTERRA_PUBLISHER_ID to vault",
			"propeller": "Sign up at propellerads.com (24h approval) → add PROPELLERADS_PUBLISHER_ID to vault",
			"binance":   "Already configured — Binance API key found in vault",
		},
	}
}

// ========== PageOutput type used for injection ==========
type PageOutput struct {
	HTML string
	// other fields used in page generation
}
