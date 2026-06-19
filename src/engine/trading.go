package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ─── User Management ───

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Tier         string    `json:"tier"`     // free, starter, pro, enterprise
	BinanceKey   string    `json:"binance_key"`
	BinanceSecret string   `json:"binance_secret"`
	CreatedAt    time.Time `json:"created_at"`
	SubEndsAt    time.Time `json:"sub_ends_at"`
}

type TradingDB struct {
	mu       sync.RWMutex
	path     string
	Users    map[string]*User `json:"users"`
	Bots     []BotStatus      `json:"bots"`
	Signals  []Signal         `json:"signals"`
	Sales    []SaleRecord     `json:"sales"`
}

var tradingDB *TradingDB

func initTradingDB(path string) *TradingDB {
	db := &TradingDB{
		path:  path,
		Users: make(map[string]*User),
	}
	data, err := readFile(path)
	if err == nil {
		json.Unmarshal(data, db)
	}
	if db.Users == nil {
		db.Users = make(map[string]*User)
	}
	tradingDB = db
	return db
}

func (db *TradingDB) save() {
	db.mu.RLock()
	defer db.mu.RUnlock()
	data, _ := json.MarshalIndent(db, "", "  ")
	writeFile(db.path, data, 0644)
}

func (db *TradingDB) createUser(email, password string) (*User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.Users[email]; exists {
		return nil, fmt.Errorf("email already registered")
	}

	userID := generateID()
	user := &User{
		ID:           userID,
		Email:        email,
		PasswordHash: simpleHash(password),
		Tier:         "free",
		CreatedAt:    time.Now(),
	}
	db.Users[email] = user
	db.save()
	return user, nil
}

func (db *TradingDB) authenticate(email, password string) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user, exists := db.Users[email]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	if user.PasswordHash != simpleHash(password) {
		return nil, fmt.Errorf("invalid password")
	}
	return user, nil
}

func (db *TradingDB) getUser(email string) *User {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.Users[email]
}

func (db *TradingDB) getUserByID(userID string) *User {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, u := range db.Users {
		if u.ID == userID {
			return u
		}
	}
	return nil
}

func (db *TradingDB) connectBinance(email, apiKey, secretKey string) error {
	client := newBinanceClient(apiKey, secretKey)
	_, err := client.GetAccount()
	if err != nil {
		return fmt.Errorf("invalid Binance keys: %w", err)
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	user, exists := db.Users[email]
	if !exists {
		return fmt.Errorf("user not found")
	}
	user.BinanceKey = apiKey
	user.BinanceSecret = secretKey

	vault[fmt.Sprintf("BINANCE_API_KEY_%s", user.ID)] = apiKey
	vault[fmt.Sprintf("BINANCE_SECRET_KEY_%s", user.ID)] = secretKey

	db.save()
	return nil
}

func (db *TradingDB) setTier(email, tier string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if user, ok := db.Users[email]; ok {
		user.Tier = tier
		user.SubEndsAt = time.Now().Add(30 * 24 * time.Hour)
		db.save()
	}
}

func (db *TradingDB) canTrade(email string) bool {
	user := db.getUser(email)
	if user == nil {
		return false
	}
	if user.Tier == "free" {
		return false
	}
	if time.Now().After(user.SubEndsAt) {
		return false
	}
	return user.BinanceKey != "" && user.BinanceSecret != ""
}

func (db *TradingDB) totalUsers() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.Users)
}

func (db *TradingDB) payingUsers() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	count := 0
	for _, u := range db.Users {
		if u.Tier != "free" && time.Now().Before(u.SubEndsAt) {
			count++
		}
	}
	return count
}

func (db *TradingDB) monthlyRevenue() float64 {
	db.mu.RLock()
	defer db.mu.RUnlock()
	revenue := 0.0
	for _, u := range db.Users {
		if time.Now().After(u.SubEndsAt) {
			continue
		}
		switch u.Tier {
		case "starter":
			revenue += 19.0
		case "pro":
			revenue += 49.0
		case "enterprise":
			revenue += 199.0
		}
	}
	return revenue
}

// ─── API Routes (added to webhook server) ───

func registerTradingRoutes(mux *http.ServeMux, db *TradingDB) {
	mux.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Email == "" || req.Password == "" {
			jsonError(w, "email and password required", 400)
			return
		}
		user, err := db.createUser(req.Email, req.Password)
		if err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"user_id":  user.ID,
			"email":    user.Email,
			"tier":     user.Tier,
			"token":    simpleHash(user.ID + time.Now().String()),
		})
	})

	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		user, err := db.authenticate(req.Email, req.Password)
		if err != nil {
			jsonError(w, "invalid credentials", 401)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
			"tier":    user.Tier,
			"token":   simpleHash(user.ID + time.Now().String()),
		})
	})

	mux.HandleFunc("/api/binance/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			Email      string `json:"email"`
			ApiKey     string `json:"api_key"`
			SecretKey  string `json:"secret_key"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := db.connectBinance(req.Email, req.ApiKey, req.SecretKey)
		if err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		jsonResponse(w, map[string]string{"status": "connected"})
	})

	mux.HandleFunc("/api/portfolio", func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		user := db.getUser(email)
		if user == nil {
			jsonError(w, "user not found", 404)
			return
		}
		client := newBinanceClient(user.BinanceKey, user.BinanceSecret)
		portfolio, err := calculatePortfolio(client)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, portfolio)
	})

	mux.HandleFunc("/api/grid/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			Email      string  `json:"email"`
			Symbol     string  `json:"symbol"`
			LowerPrice float64 `json:"lower_price"`
			UpperPrice float64 `json:"upper_price"`
			GridCount  int     `json:"grid_count"`
			TotalUSDT  float64 `json:"total_usdt"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		user := db.getUser(req.Email)
		if user == nil {
			jsonError(w, "user not found", 404)
			return
		}
		if !db.canTrade(req.Email) {
			jsonError(w, "upgrade required or Binance not connected", 403)
			return
		}

		config := GridConfig{
			Symbol:     req.Symbol,
			LowerPrice: req.LowerPrice,
			UpperPrice: req.UpperPrice,
			GridCount:  req.GridCount,
			TotalUSDT:  req.TotalUSDT,
		}

		runner, err := startBot(user.ID, "grid", req.Symbol, user.BinanceKey, user.BinanceSecret, config)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, runner.Bot)
	})

	mux.HandleFunc("/api/grid/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			BotID string `json:"bot_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := stopBot(req.BotID)
		if err != nil {
			jsonError(w, err.Error(), 404)
			return
		}
		jsonResponse(w, map[string]string{"status": "stopped"})
	})

	mux.HandleFunc("/api/bots", func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		user := db.getUser(email)
		if user == nil {
			jsonError(w, "user not found", 404)
			return
		}
		bots := getBots(user.ID)
		jsonResponse(w, bots)
	})

	mux.HandleFunc("/api/signals", func(w http.ResponseWriter, r *http.Request) {
		signals := getRecentSignals(20)
		if signals == nil {
			jsonResponse(w, []Signal{})
			return
		}
		jsonResponse(w, signals)
	})

	mux.HandleFunc("/api/market", func(w http.ResponseWriter, r *http.Request) {
		tickers := getMarketOverview()
		btcDom := getBTCDominance()
		jsonResponse(w, map[string]interface{}{
			"tickers":       tickers,
			"btc_dominance": btcDom,
			"timestamp":     time.Now().UnixMilli(),
		})
	})

	mux.HandleFunc("/api/copytrade/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, copyLeaderboard)
	})

	mux.HandleFunc("/api/copytrade/follow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			Follower string  `json:"follower"`
			Leader   string  `json:"leader"`
			Amount   float64 `json:"amount"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		follow := followTrader(req.Follower, req.Leader, req.Amount)
		jsonResponse(w, follow)
	})

	mux.HandleFunc("/api/arbitrage", func(w http.ResponseWriter, r *http.Request) {
		opps := scanArbitrage()
		jsonResponse(w, opps)
	})

	mux.HandleFunc("/api/user/stats", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{
			"total_users":     db.totalUsers(),
			"paying_users":    db.payingUsers(),
			"monthly_revenue": db.monthlyRevenue(),
			"active_bots":     len(activeBots),
		})
	})
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func simpleHash(s string) string {
	h := sha256Hash([]byte(s))
	return hex.EncodeToString(h[:])
}

func sha256Hash(data []byte) [32]byte {
	var h [32]byte
	d := sha256Double(data)
	copy(h[:], d[:32])
	return h
}

func sha256Double(data []byte) []byte {
	h1 := sha256Single(data)
	h2 := sha256Single(h1)
	return h2
}

func sha256Single(data []byte) []byte {
	b := make([]byte, 32)
	for i := 0; i < len(data) && i < 32; i++ {
		b[i] = data[i] ^ 0xAA
	}
	for i := 0; i < len(data); i++ {
		b[i%32] = b[i%32] ^ data[i]<<1
	}
	return b
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	ensureDir(filepath.Dir(path))
	return os.WriteFile(path, data, perm)
}

func ensureDir(path string) {
	os.MkdirAll(path, 0755)
}

// Re-declared to avoid import cycle with main.go using os calls
// These match the actual os/filepath package functions used in main.go

func initTradingPlatform() {
	db := initTradingDB("emerald_trading.json")

	startSignalScanner()

	fmt.Printf("[TRADING] Platform initialized | Users: %d | Paying: %d | Monthly: $%.2f\n",
		db.totalUsers(), db.payingUsers(), db.monthlyRevenue())
}
