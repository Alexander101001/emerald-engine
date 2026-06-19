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
	"sync"
	"time"
)

type BinanceClient struct {
	APIKey    string
	SecretKey string
	BaseURL   string
	mu        sync.Mutex
}

type Kline struct {
	OpenTime  int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime int64
}

type Ticker24 struct {
	Symbol      string  `json:"symbol"`
	PriceChange float64 `json:"priceChange,string"`
	LastPrice   float64 `json:"lastPrice,string"`
	Volume      float64 `json:"volume,string"`
	QuoteVolume float64 `json:"quoteVolume,string"`
	HighPrice   float64 `json:"highPrice,string"`
	LowPrice    float64 `json:"lowPrice,string"`
}

type AccountBalance struct {
	Asset  string  `json:"asset"`
	Free   float64 `json:"free,string"`
	Locked float64 `json:"locked,string"`
}

type OrderResult struct {
	Symbol        string `json:"symbol"`
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	TransactTime  int64  `json:"transactTime"`
	Price         string `json:"price"`
	OrigQty       string `json:"origQty"`
	ExecutedQty   string `json:"executedQty"`
	Status        string `json:"status"`
	Side          string `json:"side"`
	Type          string `json:"type"`
}

var binanceBaseURL = "https://api.binance.com"

func newBinanceClient(apiKey, secretKey string) *BinanceClient {
	return &BinanceClient{
		APIKey:    apiKey,
		SecretKey: secretKey,
		BaseURL:   binanceBaseURL,
	}
}

func (b *BinanceClient) signedRequest(method, endpoint string, params map[string]string) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ts := time.Now().UnixMilli()
	params["timestamp"] = fmt.Sprintf("%d", ts)

	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	queryString := query.Encode()

	mac := hmac.New(sha256.New, []byte(b.SecretKey))
	mac.Write([]byte(queryString))
	signature := hex.EncodeToString(mac.Sum(nil))
	fullURL := b.BaseURL + endpoint + "?" + queryString + "&signature=" + signature

	req, _ := http.NewRequest(method, fullURL, nil)
	if b.APIKey != "" {
		req.Header.Set("X-MBX-APIKEY", b.APIKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		var errResp map[string]interface{}
		json.Unmarshal(body, &errResp)
		return nil, fmt.Errorf("binance error %d: %v", resp.StatusCode, string(body))
	}
	return body, nil
}

func (b *BinanceClient) publicRequest(endpoint string, params map[string]string) ([]byte, error) {
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	fullURL := b.BaseURL + endpoint + "?" + query.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("binance public: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("binance public error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (b *BinanceClient) GetAccount() ([]AccountBalance, error) {
	data, err := b.signedRequest("GET", "/api/v3/account", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Balances []AccountBalance `json:"balances"`
	}
	json.Unmarshal(data, &result)
	var nonZero []AccountBalance
	for _, bal := range result.Balances {
		if bal.Free > 0 || bal.Locked > 0 {
			nonZero = append(nonZero, bal)
		}
	}
	return nonZero, nil
}

func (b *BinanceClient) GetTicker(symbol string) (*Ticker24, error) {
	data, err := b.publicRequest("/api/v3/ticker/24hr", map[string]string{"symbol": symbol})
	if err != nil {
		return nil, err
	}
	var ticker Ticker24
	json.Unmarshal(data, &ticker)
	return &ticker, nil
}

func (b *BinanceClient) GetKlines(symbol, interval string, limit int) ([]Kline, error) {
	params := map[string]string{
		"symbol":   symbol,
		"interval": interval,
		"limit":    fmt.Sprintf("%d", limit),
	}
	data, err := b.publicRequest("/api/v3/klines", params)
	if err != nil {
		return nil, err
	}
	var raw [][]interface{}
	json.Unmarshal(data, &raw)

	klines := make([]Kline, len(raw))
	for i, r := range raw {
		klines[i] = Kline{
			OpenTime:  int64(r[0].(float64)),
			Open:      parseFloat(r[1]),
			High:      parseFloat(r[2]),
			Low:       parseFloat(r[3]),
			Close:     parseFloat(r[4]),
			Volume:    parseFloat(r[5]),
			CloseTime: int64(r[6].(float64)),
		}
	}
	return klines, nil
}

func (b *BinanceClient) PlaceOrder(symbol, side, orderType string, quantity float64, price float64) (*OrderResult, error) {
	params := map[string]string{
		"symbol":           symbol,
		"side":             side,
		"type":             orderType,
		"quantity":         fmt.Sprintf("%.8f", quantity),
		"newOrderRespType": "FULL",
	}
	if orderType == "LIMIT" {
		params["price"] = fmt.Sprintf("%.8f", price)
		params["timeInForce"] = "GTC"
	}
	if orderType == "STOP_LOSS_LIMIT" || orderType == "TAKE_PROFIT_LIMIT" {
		params["price"] = fmt.Sprintf("%.8f", price)
		params["stopPrice"] = fmt.Sprintf("%.8f", price)
		params["timeInForce"] = "GTC"
	}

	data, err := b.signedRequest("POST", "/api/v3/order", params)
	if err != nil {
		return nil, err
	}
	var result OrderResult
	json.Unmarshal(data, &result)
	return &result, nil
}

func (b *BinanceClient) CancelOrder(symbol string, orderID int64) error {
	_, err := b.signedRequest("DELETE", "/api/v3/order", map[string]string{
		"symbol":  symbol,
		"orderId": fmt.Sprintf("%d", orderID),
	})
	return err
}

func (b *BinanceClient) GetOpenOrders(symbol string) ([]OrderResult, error) {
	params := map[string]string{}
	if symbol != "" {
		params["symbol"] = symbol
	}
	data, err := b.signedRequest("GET", "/api/v3/openOrders", params)
	if err != nil {
		return nil, err
	}
	var orders []OrderResult
	json.Unmarshal(data, &orders)
	return orders, nil
}

func (b *BinanceClient) GetExchangeInfo() (map[string]interface{}, error) {
	data, err := b.publicRequest("/api/v3/exchangeInfo", nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}

func (b *BinanceClient) GetAccountSnapshot() (map[string]interface{}, error) {
	data, err := b.signedRequest("GET", "/sapi/v1/accountSnapshot", map[string]string{"type": "SPOT"})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}

func parseFloat(v interface{}) float64 {
	switch val := v.(type) {
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	case float64:
		return val
	}
	return 0
}

func userBinanceAPI(userID string) *BinanceClient {
	key := vaultGet("BINANCE_API_KEY_"+userID, "")
	secret := vaultGet("BINANCE_SECRET_KEY_"+userID, "")
	if key == "" || secret == "" {
		return nil
	}
	return newBinanceClient(key, secret)
}

func validateBinanceKeys(apiKey, secretKey string) ([]AccountBalance, error) {
	client := newBinanceClient(apiKey, secretKey)
	return client.GetAccount()
}

var topCryptos = []string{
	"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT",
	"ADAUSDT", "AVAXUSDT", "DOGEUSDT", "DOTUSDT", "MATICUSDT",
	"LINKUSDT", "UNIUSDT", "SHIBUSDT", "LTCUSDT", "ATOMUSDT",
}

func getMarketOverview() []Ticker24 {
	var tickers []Ticker24
	client := newBinanceClient("", "")
	for _, sym := range topCryptos {
		t, err := client.GetTicker(sym)
		if err != nil {
			continue
		}
		tickers = append(tickers, *t)
	}
	return tickers
}

func getBTCDominance() float64 {
	client := newBinanceClient("", "")
	btc, err := client.GetTicker("BTCUSDT")
	if err != nil {
		return 0
	}
	total := btc.QuoteVolume
	for _, sym := range topCryptos[1:] {
		t, err := client.GetTicker(sym)
		if err != nil {
			continue
		}
		total += t.QuoteVolume
	}
	if total == 0 {
		return 0
	}
	return (btc.QuoteVolume / total) * 100
}
