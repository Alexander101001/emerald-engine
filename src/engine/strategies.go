package main

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type Signal struct {
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"` // BUY or SELL
	Price     float64 `json:"price"`
	Strength  float64 `json:"strength"` // 0-100
	Strategy  string  `json:"strategy"`
	Timestamp int64   `json:"timestamp"`
	Reason    string  `json:"reason"`
}

type BotStatus struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"` // grid, signal, copy
	Symbol    string    `json:"symbol"`
	Status    string    `json:"status"` // running, paused, stopped
	CreatedAt time.Time `json:"created_at"`
	Config    string    `json:"config"`
	PnL       float64   `json:"pnl"`
	TradeCount int      `json:"trade_count"`
}

type GridConfig struct {
	Symbol     string  `json:"symbol"`
	LowerPrice float64 `json:"lower_price"`
	UpperPrice float64 `json:"upper_price"`
	GridCount  int     `json:"grid_count"`
	TotalUSDT  float64 `json:"total_usdt"`
	Side       string  `json:"side"` // LONG, SHORT, NEUTRAL
}

type GridLevel struct {
	BuyPrice  float64 `json:"buy_price"`
	SellPrice float64 `json:"sell_price"`
	Size      float64 `json:"size"`
	Filled    bool    `json:"filled"`
}

var (
	activeBots     = make(map[string]*BotRunner)
	activeBotsMu   sync.RWMutex
	signalHistory  []Signal
	signalHistoryMu sync.RWMutex
	signalCh       = make(chan Signal, 100)
)

type BotRunner struct {
	Bot      BotStatus
	Grid     *GridConfig
	Client   *BinanceClient
	Stop     chan struct{}
	mu       sync.Mutex
}

func startBot(userID, botType, symbol, apiKey, secretKey string, config interface{}) (*BotRunner, error) {
	client := newBinanceClient(apiKey, secretKey)

	botID := fmt.Sprintf("%s-%s-%d", userID, botType, time.Now().Unix())
	bot := BotStatus{
		ID:        botID,
		UserID:    userID,
		Type:      botType,
		Symbol:    symbol,
		Status:    "running",
		CreatedAt: time.Now(),
	}
	runner := &BotRunner{
		Bot:    bot,
		Client: client,
		Stop:   make(chan struct{}),
	}

	switch bt := config.(type) {
	case GridConfig:
		runner.Grid = &bt
		runner.Bot.Config = fmt.Sprintf("grid:%s-%.8f-%.8f-%d", bt.Symbol, bt.LowerPrice, bt.UpperPrice, bt.GridCount)
		go runner.runGridBot()
	default:
		return nil, fmt.Errorf("unknown bot config type")
	}

	activeBotsMu.Lock()
	activeBots[botID] = runner
	activeBotsMu.Unlock()

	return runner, nil
}

func stopBot(botID string) error {
	activeBotsMu.Lock()
	defer activeBotsMu.Unlock()

	runner, ok := activeBots[botID]
	if !ok {
		return fmt.Errorf("bot not found: %s", botID)
	}
	close(runner.Stop)
	runner.Bot.Status = "stopped"
	delete(activeBots, botID)
	return nil
}

func getBots(userID string) []BotStatus {
	activeBotsMu.RLock()
	defer activeBotsMu.RUnlock()

	var bots []BotStatus
	for _, r := range activeBots {
		if r.Bot.UserID == userID {
			bots = append(bots, r.Bot)
		}
	}
	return bots
}

func getBot(botID string) *BotRunner {
	activeBotsMu.RLock()
	defer activeBotsMu.RUnlock()
	return activeBots[botID]
}

// ─── Grid Trading Bot ───

func (r *BotRunner) runGridBot() {
	r.mu.Lock()
	cfg := *r.Grid
	r.mu.Unlock()

	gridSize := (cfg.UpperPrice - cfg.LowerPrice) / float64(cfg.GridCount)
	investPerGrid := cfg.TotalUSDT / float64(cfg.GridCount)

	levels := make([]GridLevel, cfg.GridCount)
	for i := 0; i < cfg.GridCount; i++ {
		levels[i] = GridLevel{
			BuyPrice:  cfg.LowerPrice + float64(i)*gridSize,
			SellPrice: cfg.LowerPrice + float64(i+1)*gridSize,
			Size:      investPerGrid / (cfg.LowerPrice + float64(i)*gridSize),
		}
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Stop:
			return
		case <-ticker.C:
			ticker24, err := r.Client.GetTicker(cfg.Symbol)
			if err != nil {
				continue
			}

			currentPrice := ticker24.LastPrice

			r.mu.Lock()
			for i := range levels {
				level := &levels[i]
				if !level.Filled && currentPrice <= level.BuyPrice {
					_, err := r.Client.PlaceOrder(cfg.Symbol, "BUY", "LIMIT", level.Size, level.BuyPrice)
					if err == nil {
						level.Filled = true
						r.Bot.TradeCount++
						emitSignal(Signal{
							Symbol:   cfg.Symbol,
							Side:     "BUY",
							Price:    level.BuyPrice,
							Strength: 70,
							Strategy: "GRID",
							Reason:   fmt.Sprintf("Grid buy at %.8f", level.BuyPrice),
						})
					}
				}
				if level.Filled && currentPrice >= level.SellPrice {
					_, err := r.Client.PlaceOrder(cfg.Symbol, "SELL", "LIMIT", level.Size, level.SellPrice)
					if err == nil {
						level.Filled = false
						r.Bot.TradeCount++
						r.Bot.PnL += (level.SellPrice - level.BuyPrice) * level.Size
						emitSignal(Signal{
							Symbol:   cfg.Symbol,
							Side:     "SELL",
							Price:    level.SellPrice,
							Strength: 70,
							Strategy: "GRID",
							Reason:   fmt.Sprintf("Grid sell at %.8f", level.SellPrice),
						})
					}
				}
			}
			r.mu.Unlock()
		}
	}
}

func emitSignal(s Signal) {
	s.Timestamp = time.Now().UnixMilli()
	signalHistoryMu.Lock()
	signalHistory = append(signalHistory, s)
	if len(signalHistory) > 10000 {
		signalHistory = signalHistory[len(signalHistory)-10000:]
	}
	signalHistoryMu.Unlock()

	select {
	case signalCh <- s:
	default:
	}
}

// ─── Signal Strategies ───

func calculateSMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return nil
	}
	sma := make([]float64, len(prices)-period+1)
	for i := 0; i <= len(prices)-period; i++ {
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += prices[i+j]
		}
		sma[i] = sum / float64(period)
	}
	return sma
}

func calculateEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return nil
	}
	multiplier := 2.0 / float64(period+1)
	ema := make([]float64, len(prices))
	// First value is SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)
	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i]-ema[i-1])*multiplier + ema[i-1]
	}
	return ema
}

func calculateRSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return nil
	}
	rsi := make([]float64, len(prices)-period)
	gains, losses := 0.0, 0.0
	for i := 1; i <= period; i++ {
		diff := prices[i] - prices[i-1]
		if diff > 0 {
			gains += diff
		} else {
			losses -= diff
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	if avgLoss == 0 {
		rsi[0] = 100
	} else {
		rs := avgGain / avgLoss
		rsi[0] = 100 - 100/(1+rs)
	}
	for i := period + 1; i < len(prices); i++ {
		diff := prices[i] - prices[i-1]
		g, l := 0.0, 0.0
		if diff > 0 {
			g = diff
		} else {
			l = -diff
		}
		avgGain = (avgGain*float64(period-1) + g) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + l) / float64(period)
		if avgLoss == 0 {
			rsi[i-period] = 100
		} else {
			rs := avgGain / avgLoss
			rsi[i-period] = 100 - 100/(1+rs)
		}
	}
	return rsi
}

func calculateMACD(prices []float64) ([]float64, []float64, []float64) {
	ema12 := calculateEMA(prices, 12)
	ema26 := calculateEMA(prices, 26)
	if ema12 == nil || ema26 == nil {
		return nil, nil, nil
	}
	minLen := len(ema12)
	if len(ema26) < minLen {
		minLen = len(ema26)
	}
	macdLine := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		macdLine[i] = ema12[i] - ema26[i]
	}
	signal := calculateEMA(macdLine, 9)
	if signal == nil {
		return macdLine, nil, nil
	}
	histogram := make([]float64, len(signal))
	for i := 0; i < len(signal); i++ {
		histogram[i] = macdLine[i] - signal[i]
	}
	return macdLine, signal, histogram
}

func generateRSISignal(symbol string, client *BinanceClient, interval string) *Signal {
	klines, err := client.GetKlines(symbol, interval, 50)
	if err != nil || len(klines) < 20 {
		return nil
	}
	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
	}
	rsi := calculateRSI(prices, 14)
	if len(rsi) == 0 {
		return nil
	}
	lastRSI := rsi[len(rsi)-1]
	currentPrice := prices[len(prices)-1]

	if lastRSI < 30 {
		return &Signal{
			Symbol:   symbol,
			Side:     "BUY",
			Price:    currentPrice,
			Strength: 100 - lastRSI,
			Strategy: "RSI",
			Reason:   fmt.Sprintf("RSI oversold: %.1f (threshold: 30)", lastRSI),
		}
	}
	if lastRSI > 70 {
		return &Signal{
			Symbol:   symbol,
			Side:     "SELL",
			Price:    currentPrice,
			Strength: lastRSI,
			Strategy: "RSI",
			Reason:   fmt.Sprintf("RSI overbought: %.1f (threshold: 70)", lastRSI),
		}
	}
	return nil
}

func generateMACDSignal(symbol string, client *BinanceClient, interval string) *Signal {
	klines, err := client.GetKlines(symbol, interval, 50)
	if err != nil || len(klines) < 26 {
		return nil
	}
	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
	}
	_, _, histogram := calculateMACD(prices)
	if len(histogram) < 2 {
		return nil
	}
	currentPrice := prices[len(prices)-1]
	prevHist := histogram[len(histogram)-2]
	currHist := histogram[len(histogram)-1]

	if prevHist < 0 && currHist > 0 {
		return &Signal{
			Symbol:   symbol,
			Side:     "BUY",
			Price:    currentPrice,
			Strength: 75,
			Strategy: "MACD",
			Reason:   "MACD histogram turned positive (bullish crossover)",
		}
	}
	if prevHist > 0 && currHist < 0 {
		return &Signal{
			Symbol:   symbol,
			Side:     "SELL",
			Price:    currentPrice,
			Strength: 75,
			Strategy: "MACD",
			Reason:   "MACD histogram turned negative (bearish crossover)",
		}
	}
	return nil
}

func generateEMACrossSignal(symbol string, client *BinanceClient, interval string) *Signal {
	klines, err := client.GetKlines(symbol, interval, 100)
	if err != nil || len(klines) < 50 {
		return nil
	}
	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
	}
	ema20 := calculateEMA(prices, 20)
	ema50 := calculateEMA(prices, 50)
	if len(ema20) < 2 || len(ema50) < 2 {
		return nil
	}
	ema20Prev := ema20[len(ema20)-2]
	ema20Curr := ema20[len(ema20)-1]
	ema50Prev := ema50[len(ema50)-2]
	ema50Curr := ema50[len(ema50)-1]
	currentPrice := prices[len(prices)-1]

	if ema20Prev < ema50Prev && ema20Curr > ema50Curr {
		return &Signal{
			Symbol:   symbol,
			Side:     "BUY",
			Price:    currentPrice,
			Strength: 80,
			Strategy: "EMA_CROSS",
			Reason:   "EMA20 crossed above EMA50 (golden cross)",
		}
	}
	if ema20Prev > ema50Prev && ema20Curr < ema50Curr {
		return &Signal{
			Symbol:   symbol,
			Side:     "SELL",
			Price:    currentPrice,
			Strength: 80,
			Strategy: "EMA_CROSS",
			Reason:   "EMA20 crossed below EMA50 (death cross)",
		}
	}
	return nil
}

func scanAllSignals() []Signal {
	var signals []Signal
	client := newBinanceClient("", "")

	for _, sym := range topCryptos {
		for _, generator := range []func(string, *BinanceClient, string) *Signal{
			generateRSISignal,
			generateMACDSignal,
			generateEMACrossSignal,
		} {
			for _, interval := range []string{"1h", "4h", "1d"} {
				sig := generator(sym, client, interval)
				if sig != nil {
					sig.Timestamp = time.Now().UnixMilli()
					signals = append(signals, *sig)
				}
			}
		}
	}

	sort.Slice(signals, func(i, j int) bool {
		return signals[i].Strength > signals[j].Strength
	})
	if len(signals) > 20 {
		signals = signals[:20]
	}

	return signals
}

func startSignalScanner() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			signals := scanAllSignals()
			for _, s := range signals {
				emitSignal(s)
			}
			fmt.Printf("[SIGNALS] Scanned %d cryptos, found %d signals\n", len(topCryptos), len(signals))
			<-ticker.C
		}
	}()
}

func getRecentSignals(limit int) []Signal {
	signalHistoryMu.RLock()
	defer signalHistoryMu.RUnlock()

	n := len(signalHistory)
	if n == 0 {
		return nil
	}
	if limit > n {
		limit = n
	}
	result := make([]Signal, limit)
	copy(result, signalHistory[n-limit:])
	return result
}

// ─── Copy Trading ───

type TraderLeaderboard struct {
	UserID    string  `json:"user_id"`
	PnL       float64 `json:"pnl"`
	WinRate   float64 `json:"win_rate"`
	TradeCount int    `json:"trade_count"`
	Followers int     `json:"followers"`
}

type CopyTradeFollow struct {
	ID        string    `json:"id"`
	Follower  string    `json:"follower"`
	Leader    string    `json:"leader"`
	AllocUSDT float64   `json:"alloc_usdt"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	copyFollowers   = make(map[string][]CopyTradeFollow)
	copyLeaderboard []TraderLeaderboard
	copyMu          sync.RWMutex
)

func followTrader(followerID, leaderID string, allocUSDT float64) *CopyTradeFollow {
	follow := &CopyTradeFollow{
		ID:        fmt.Sprintf("cp-%s-%s-%d", followerID, leaderID, time.Now().Unix()),
		Follower:  followerID,
		Leader:    leaderID,
		AllocUSDT: allocUSDT,
		Active:    true,
		CreatedAt: time.Now(),
	}
	copyMu.Lock()
	copyFollowers[leaderID] = append(copyFollowers[leaderID], *follow)
	copyMu.Unlock()
	return follow
}

func unfollowTrader(followerID, leaderID string) {
	copyMu.Lock()
	defer copyMu.Unlock()

	follows := copyFollowers[leaderID]
	for i, f := range follows {
		if f.Follower == followerID {
			follows[i].Active = false
			break
		}
	}
}

// ─── Arbitrage Scanner ───

type ArbitrageOpportunity struct {
	Symbol    string  `json:"symbol"`
	BuyAt     string  `json:"buy_at"`
	SellAt    string  `json:"sell_at"`
	BuyPrice  float64 `json:"buy_price"`
	SellPrice float64 `json:"sell_price"`
	ProfitPct float64 `json:"profit_pct"`
	Timestamp int64   `json:"timestamp"`
}

func scanArbitrage() []ArbitrageOpportunity {
	var opportunities []ArbitrageOpportunity

	for _, sym := range topCryptos {
		spotClient := newBinanceClient("", "")
		ticker, err := spotClient.GetTicker(sym)
		if err != nil {
			continue
		}

		bidPrice := ticker.LastPrice * 0.998
		askPrice := ticker.LastPrice * 1.002
		profitPct := ((askPrice - bidPrice) / bidPrice) * 100

		if profitPct > 0.1 {
			opportunities = append(opportunities, ArbitrageOpportunity{
				Symbol:    sym,
				BuyAt:     "SPOT",
				SellAt:    "SPOT",
				BuyPrice:  bidPrice,
				SellPrice: askPrice,
				ProfitPct: profitPct,
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}

	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].ProfitPct > opportunities[j].ProfitPct
	})
	if len(opportunities) > 5 {
		opportunities = opportunities[:5]
	}
	return opportunities
}

// ─── Portfolio Tracker ───

type PortfolioSummary struct {
	TotalUSDT    float64            `json:"total_usdt"`
	TotalBTC     float64            `json:"total_btc"`
	PnL24h       float64            `json:"pnl_24h"`
	PnLPercentage float64           `json:"pnl_percentage"`
	Balances     []AccountBalance   `json:"balances"`
	Allocation   map[string]float64 `json:"allocation"`
}

func calculatePortfolio(client *BinanceClient) (*PortfolioSummary, error) {
	balances, err := client.GetAccount()
	if err != nil {
		return nil, err
	}

	summary := &PortfolioSummary{
		Allocation: make(map[string]float64),
		Balances:   balances,
	}

	btcPrice := 0.0
	if btcTicker, err := client.GetTicker("BTCUSDT"); err == nil {
		btcPrice = btcTicker.LastPrice
	}

	for _, bal := range balances {
		if bal.Free <= 0 && bal.Locked <= 0 {
			continue
		}
		usdValue := 0.0
		if bal.Asset == "USDT" {
			usdValue = bal.Free + bal.Locked
		} else {
			ticker, err := client.GetTicker(bal.Asset + "USDT")
			if err == nil {
				usdValue = (bal.Free + bal.Locked) * ticker.LastPrice
			}
		}
		summary.TotalUSDT += usdValue
		if bal.Asset != "USDT" && usdValue > 0 {
			summary.Allocation[bal.Asset] = math.Round(usdValue/summary.TotalUSDT*10000) / 100
		}
	}
	if btcPrice > 0 {
		summary.TotalBTC = summary.TotalUSDT / btcPrice
	}
	return summary, nil
}
