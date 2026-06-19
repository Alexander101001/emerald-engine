package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type AmazonProduct struct {
	ASIN       string  `json:"asin"`
	Title      string  `json:"title"`
	Price      float64 `json:"price"`
	Currency   string  `json:"currency"`
	URL        string  `json:"url"`
	Image      string  `json:"image"`
	Rating     float64 `json:"rating"`
	ReviewCount int    `json:"review_count"`
	Category   string  `json:"category"`
	Commission float64 `json:"commission"`
}

type AmazonCache struct {
	mu       sync.RWMutex
	products map[string][]AmazonProduct
	ttl      time.Duration
	updated  map[string]time.Time
}

func newAmazonCache() *AmazonCache {
	return &AmazonCache{
		products: make(map[string][]AmazonProduct),
		ttl:      24 * time.Hour,
		updated:  make(map[string]time.Time),
	}
}

func (c *AmazonCache) get(keyword string) ([]AmazonProduct, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	prods, ok := c.products[keyword]
	updated, hasUpdate := c.updated[keyword]
	if !ok || !hasUpdate || time.Since(updated) > c.ttl {
		return nil, false
	}
	return prods, true
}

func (c *AmazonCache) set(keyword string, products []AmazonProduct) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.products[keyword] = products
	c.updated[keyword] = time.Now()
}

var amazonProductCache = newAmazonCache()

var amazonStaticProducts = map[string][]AmazonProduct{
	"saas-marketing": {
		{ASIN: "B0BXYZ1", Title: "AI Marketing Hub Pro", Price: 299.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ1?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/51XYZ.jpg", Rating: 4.5, ReviewCount: 234, Category: "Software", Commission: 29.99},
		{ASIN: "B0CABC1", Title: "SmartCampaign 360", Price: 149.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0CABC1?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/52ABC.jpg", Rating: 4.2, ReviewCount: 189, Category: "Software", Commission: 14.99},
		{ASIN: "B0DDEF1", Title: "ConvertBot Elite", Price: 499.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0DDEF1?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/53DEF.jpg", Rating: 4.8, ReviewCount: 567, Category: "Software", Commission: 49.99},
	},
	"health-wellness": {
		{ASIN: "B0BXYZ2", Title: "FitTracker Pro Watch", Price: 199.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ2?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/61XYZ.jpg", Rating: 4.6, ReviewCount: 1234, Category: "Electronics", Commission: 11.99},
		{ASIN: "B0CABC2", Title: "Smart Water Bottle", Price: 49.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0CABC2?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/62ABC.jpg", Rating: 4.3, ReviewCount: 892, Category: "Sports", Commission: 2.99},
		{ASIN: "B0DDEF2", Title: "Ergonomic Office Chair", Price: 399.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0DDEF2?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/63DEF.jpg", Rating: 4.7, ReviewCount: 3456, Category: "Home", Commission: 23.99},
	},
	"personal-finance": {
		{ASIN: "B0BXYZ3", Title: "Personal Finance Planner", Price: 34.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ3?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/71XYZ.jpg", Rating: 4.4, ReviewCount: 567, Category: "Books", Commission: 2.09},
		{ASIN: "B0CABC3", Title: "Budget Tracking Ledger", Price: 24.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0CABC3?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/72ABC.jpg", Rating: 4.1, ReviewCount: 345, Category: "Books", Commission: 1.49},
	},
	"digital-products": {
		{ASIN: "B0BXYZ4", Title: "Digital Creator Kit", Price: 89.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ4?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/81XYZ.jpg", Rating: 4.5, ReviewCount: 432, Category: "Software", Commission: 8.99},
	},
	"online-education": {
		{ASIN: "B0BXYZ5", Title: "Online Course Platform", Price: 199.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ5?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/91XYZ.jpg", Rating: 4.6, ReviewCount: 678, Category: "Software", Commission: 19.99},
	},
	"ecommerce": {
		{ASIN: "B0BXYZ6", Title: "E-Commerce Starter Bundle", Price: 149.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ6?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/A1XYZ.jpg", Rating: 4.3, ReviewCount: 234, Category: "Software", Commission: 14.99},
	},
	"ai-tools": {
		{ASIN: "B0BXYZ7", Title: "AI Content Generator Pro", Price: 249.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ7?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/B1XYZ.jpg", Rating: 4.7, ReviewCount: 123, Category: "Software", Commission: 24.99},
	},
	"crypto-web3": {
		{ASIN: "B0BXYZ8", Title: "Hardware Wallet Ledger", Price: 119.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ8?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/C1XYZ.jpg", Rating: 4.8, ReviewCount: 8901, Category: "Electronics", Commission: 7.19},
	},
	"real-estate": {
		{ASIN: "B0BXYZ9", Title: "Real Estate Investment Book", Price: 29.99, Currency: "USD", URL: "https://www.amazon.com/dp/B0BXYZ9?tag=emeraldeng0e-20", Image: "https://images-na.ssl-images-amazon.com/images/I/D1XYZ.jpg", Rating: 4.2, ReviewCount: 456, Category: "Books", Commission: 1.79},
	},
}

func fetchAmazonProducts(keyword string, tag string) []AmazonProduct {
	if cached, ok := amazonProductCache.get(keyword); ok {
		return cached
	}

	if k := vaultGet("AMAZON_PAAPI_KEY", ""); k != "" {
		prods := paapi5Search(keyword, tag)
		if len(prods) > 0 {
			amazonProductCache.set(keyword, prods)
			return prods
		}
	}

	static, ok := amazonStaticProducts[keyword]
	if !ok {
		static = amazonStaticProducts["saas-marketing"]
	}
	amazonProductCache.set(keyword, static)
	return static
}

func paapi5Search(keyword, tag string) []AmazonProduct {
	accessKey := vaultGet("AMAZON_PAAPI_KEY", "")
	secretKey := vaultGet("AMAZON_PAAPI_SECRET", "")
	partnerTag := tag
	if partnerTag == "" {
		partnerTag = "emeraldeng0e-20"
	}
	marketplace := "www.amazon.com"

	payload := map[string]interface{}{
		"Keywords":             keyword,
		"Resources":            []string{"Images.Primary.Medium", "ItemInfo.Title", "Offers.Listings.Price", "ParentASIN"},
		"PartnerTag":           partnerTag,
		"PartnerType":          "Associates",
		"Marketplace":          marketplace,
		"ItemCount":            5,
	}
	body, _ := json.Marshal(payload)

	host := "webservices.amazon.com"
	path := "/paapi5/searchitems"
	now := time.Now().UTC()
	date := now.Format("20060102T150405Z")

	req, _ := http.NewRequest("POST", "https://"+host+path, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Amz-Date", date)
	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Target", "com.amazon.paapi5.v1.ProductAdvertisingAPIv1.SearchItems")

	if accessKey != "" && secretKey != "" {
		sig := signPAAPI5(req, accessKey, secretKey, "us-east-1", "ProductAdvertisingAPI")
		req.Header.Set("Authorization", sig)
	}

	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[PAAPI5] request failed: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if items, ok := result["SearchResult"].(map[string]interface{})["Items"].([]interface{}); ok {
		var products []AmazonProduct
		for _, item := range items {
			it, _ := item.(map[string]interface{})
			p := AmazonProduct{
				ASIN: fmt.Sprintf("%v", it["ASIN"]),
			}
			if info, ok := it["ItemInfo"].(map[string]interface{}); ok {
				if title, ok := info["Title"].(map[string]interface{})["DisplayValue"].(string); ok {
					p.Title = title
				}
			}
			if images, ok := it["Images"].(map[string]interface{}); ok {
				if primary, ok := images["Primary"].(map[string]interface{}); ok {
					if medium, ok := primary["Medium"].(map[string]interface{}); ok {
						p.Image, _ = medium["URL"].(string)
					}
				}
			}
			if offers, ok := it["Offers"].(map[string]interface{}); ok {
				if listings, ok := offers["Listings"].([]interface{}); ok && len(listings) > 0 {
					if listing, ok := listings[0].(map[string]interface{}); ok {
						if price, ok := listing["Price"].(map[string]interface{}); ok {
							p.Price, _ = price["Amount"].(float64)
							p.Currency, _ = price["Currency"].(string)
						}
					}
				}
			}
			if p.ASIN != "" && p.Title != "" {
				p.URL = fmt.Sprintf("https://www.amazon.com/dp/%s?tag=%s", p.ASIN, partnerTag)
				p.Commission = p.Price * 0.06
				products = append(products, p)
			}
		}
		if len(products) > 0 {
			return products
		}
	}

	fmt.Fprintf(os.Stderr, "[PAAPI5] no results for '%s', using static\n", keyword)
	return nil
}

func signPAAPI5(req *http.Request, accessKey, secretKey, region, service string) string {
	now := time.Now().UTC()
	date := now.Format("20060102")
	dateStamp := now.Format("20060102T150405Z")

	body, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(strings.NewReader(string(body)))

	hashedBody := sha256Hex(body)
	canonicalURI := req.URL.Path
	canonicalQS := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), req.Host, dateStamp)
	signedHeaders := "content-type;host;x-amz-date"

	canonicalReq := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method, canonicalURI, canonicalQS, canonicalHeaders, signedHeaders, hashedBody)

	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", date, region, service)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm, dateStamp, credentialScope, sha256Hex([]byte(canonicalReq)))

	h1 := hmacSHA256([]byte("AWS4"+secretKey), []byte(date))
	h2 := hmacSHA256(h1, []byte(region))
	h3 := hmacSHA256(h2, []byte(service))
	signingKey := hmacSHA256(h3, []byte("aws4_request"))
	signature := fmt.Sprintf("%x", hmacSHA256(signingKey, []byte(stringToSign)))

	return fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, accessKey, credentialScope, signedHeaders, signature)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func dynamicAmazonLinks(nicheKeyword, tag string) string {
	prods := fetchAmazonProducts(nicheKeyword, tag)
	var links []string
	for _, p := range prods {
		links = append(links, fmt.Sprintf(
			`<a href="%s" rel="nofollow sponsored" target="_blank">%s - $%.2f</a>`,
			p.URL, p.Title, p.Price,
		))
	}
	return strings.Join(links, "\n")
}

func dynamicAmazonCards(nicheKeyword, tag string) string {
	prods := fetchAmazonProducts(nicheKeyword, tag)
	var cards []string
	for _, p := range prods {
		cards = append(cards, fmt.Sprintf(
			`<div class="acd" style="text-align:center"><div class="rt">%.1f ★</div><h4>%s</h4><p style="font-size:0.9em;color:#999">%s</p><div class="pr">$%.2f</div><a href="%s" class="btn btn-p" target="_blank" rel="nofollow sponsored" style="font-size:0.9em;padding:10px 24px;margin-top:10px;display:inline-block">Buy on Amazon →</a></div>`,
			p.Rating, p.Title, p.Category, p.Price, p.URL,
		))
	}
	return strings.Join(cards, "\n")
}
