package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SaleRecord struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Product   string    `json:"product"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type FulfillmentDB struct {
	mu     sync.RWMutex
	path   string
	Sales  []SaleRecord `json:"sales"`
}

func loadFulfillmentDB(path string) *FulfillmentDB {
	db := &FulfillmentDB{path: path}
	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, db)
	}
	return db
}

func (db *FulfillmentDB) save() {
	db.mu.RLock()
	defer db.mu.RUnlock()
	data, _ := json.MarshalIndent(db, "", "  ")
	os.MkdirAll(filepath.Dir(db.path), 0755)
	os.WriteFile(db.path, data, 0644)
}

func (db *FulfillmentDB) addSale(s SaleRecord) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.Sales = append(db.Sales, s)
	db.save()
}

func startWebhookServer(db *FulfillmentDB) {
	mux := http.NewServeMux()

	mux.HandleFunc("/webhook/stripe", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var evt map[string]interface{}
		if err := json.Unmarshal(body, &evt); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		typ, _ := evt["type"].(string)
		if typ == "checkout.session.completed" {
			session, _ := evt["data"].(map[string]interface{})["object"].(map[string]interface{})
			sale := SaleRecord{
				ID:        fmt.Sprintf("%v", session["id"]),
				Email:     fmt.Sprintf("%v", session["customer_email"]),
				Product:   fmt.Sprintf("%v", session["metadata"]),
				Amount:    session["amount_total"].(float64) / 100,
				Currency:  fmt.Sprintf("%v", session["currency"]),
				Status:    "completed",
				Timestamp: time.Now(),
			}
			db.addSale(sale)
			fmt.Printf("[WEBHOOK] Stripe sale: %s %.2f %s\n", sale.Email, sale.Amount, sale.Currency)

			productID := extractProductID(session)
			fulfillProduct(sale.Email, productID)

			telegramSend(fmt.Sprintf(
				"<b>💰 Stripe Sale!</b>\nProduct: %s\nAmount: $%.2f\nEmail: %s\nTime: %s",
				sale.Product, sale.Amount, sale.Email, sale.Timestamp.Format(time.RFC3339),
			))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"received":true}`))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		children := 0
		if orchestrator != nil {
			orchestrator.mu.Lock()
			children = len(orchestrator.Children)
			orchestrator.mu.Unlock()
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "ok",
			"time":            time.Now().Unix(),
			"sales":           len(db.Sales),
			"revenue":         db.totalRevenue(),
			"version":         "6.0",
			"factory":         orchestrator != nil && orchestrator.Token != "",
			"cognitive_cycle": getCycleNum(),
			"children":        children,
		})
	})

	mux.HandleFunc("/api/factory/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := orchestratorStats()
		json.NewEncoder(w).Encode(stats)
	})

	mux.HandleFunc("/api/cognitive/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cognitiveStats())
	})

	mux.HandleFunc("/api/scraper/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(scraperStats())
	})

	mux.HandleFunc("/api/resource/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resourceManager.harvestStats())
	})

	mux.HandleFunc("/api/agents/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if agentCoordinator == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "not_initialized"})
			return
		}
		json.NewEncoder(w).Encode(agentCoordinator.Stats())
	})

	mux.HandleFunc("/api/mcp/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if mcpRegistry == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "not_initialized"})
			return
		}
		json.NewEncoder(w).Encode(mcpRegistry.Stats())
	})

	mux.HandleFunc("/api/mcp/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			Tool   string                 `json:"tool"`
			Params map[string]interface{} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
			return
		}
		result, err := mcpRegistry.ExecuteTool(req.Tool, req.Params)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"result": result})
	})

	mux.HandleFunc("/api/cf-agent/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if cfAgent == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "not_initialized"})
			return
		}
		json.NewEncoder(w).Encode(cfAgent.Stats())
	})

	mux.HandleFunc("/api/tokens/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if tokenMatrix == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "not_initialized"})
			return
		}
		json.NewEncoder(w).Encode(tokenMatrix.Stats())
	})

	mux.HandleFunc("/api/heartbeat/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if heartbeatDaemon == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "not_initialized"})
			return
		}
		json.NewEncoder(w).Encode(heartbeatDaemon.Stats())
	})

	mux.HandleFunc("/api/heartbeat/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Printf("[HEARTBEAT] Callback received\n")
		w.Write([]byte(`{"received":true}`))
	})

	mux.HandleFunc("/api/scraper-agent/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if scraperAgent == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "not_initialized"})
			return
		}
		json.NewEncoder(w).Encode(scraperAgent.Stats())
	})

	mux.HandleFunc("/api/shield/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if shield == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "not_initialized"})
			return
		}
		json.NewEncoder(w).Encode(shield.Stats())
	})

	port := "8080"
	if p := os.Getenv("WEBHOOK_PORT"); p != "" {
		port = p
	}

	if tradingDB != nil {
		registerTradingRoutes(mux, tradingDB)
	}

	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/", fs)

	go func() {
		fmt.Printf("[WEBHOOK] Server on :%s\n", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			fmt.Fprintf(os.Stderr, "[WEBHOOK] %v\n", err)
		}
	}()
}

func (db *FulfillmentDB) totalRevenue() float64 {
	var total float64
	for _, s := range db.Sales {
		total += s.Amount
	}
	return total
}

func extractProductID(session map[string]interface{}) string {
	if meta, ok := session["metadata"].(map[string]interface{}); ok {
		if pid, ok := meta["product_id"].(string); ok {
			return pid
		}
	}
	return "unknown"
}

func fulfillProduct(email, productID string) {
	dir := "public/downloads"
	os.MkdirAll(dir, 0755)

	token := fmt.Sprintf("%x", time.Now().UnixNano())
	link := fmt.Sprintf("https://emerald-engine.com/downloads/%s", token)

	page := fmt.Sprintf(`<!DOCTYPE html><html><head><title>Download</title></head><body style="font-family:sans-serif;text-align:center;padding:60px;background:#f8f9fa"><h1 style="color:#333">✅ Thank you for your purchase!</h1><p style="color:#666;margin:20px 0">Your download link is ready. It expires in 24 hours.</p><a href="/downloads/%s/file.zip" style="display:inline-block;padding:16px 48px;background:#667eea;color:white;text-decoration:none;border-radius:50px;font-weight:bold;font-size:1.2em">Download Now</a><p style="color:#999;margin-top:30px">A copy has been sent to: %s</p></body></html>`, token, token, email)
	os.WriteFile(filepath.Join(dir, token+".html"), []byte(page), 0644)
	os.WriteFile(filepath.Join(dir, token+".txt"), []byte(fmt.Sprintf("Download link: %s\nProduct: %s\nEmail: %s\n", link, productID, email)), 0644)

	fmt.Printf("[FULFILL] %s → %s (product: %s)\n", email, link, productID)
}

var productCatalog = []struct {
	ID    string
	Name  string
	Price string
	Desc  string
	File  string
}{
	{"pro_roi", "ROI Pro Spreadsheet", "47", "Professional ROI tracking spreadsheet with templates", "roi-pro.zip"},
	{"pro_strategy", "Marketing Strategy Pack", "27", "10 proven marketing strategy templates", "strategy-pack.zip"},
	{"pro_ai", "AI Content Generator Access", "97", "Monthly access to AI content generation API", "ai-access.zip"},
	{"pro_bundle", "Full Marketing Suite Bundle", "197", "All products bundled at 60% discount", "bundles.zip"},
}

func generateStripeProductList() string {
	var rows []string
	for _, p := range productCatalog {
		rows = append(rows, fmt.Sprintf(
			`<div class="acd" style="text-align:center"><h4>%s</h4><p>%s</p><div class="pr">$%s</div>%s</div>`,
			p.Name, p.Desc, p.Price,
			stripeBtn(p.Name, p.Price),
		))
	}
	return strings.Join(rows, "\n")
}
