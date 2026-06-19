package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type CloudflareClient struct {
	APIToken string
	APIKey   string
	Email    string
	ZoneID   string
	BaseURL  string
}

func newCloudflareClient() *CloudflareClient {
	return &CloudflareClient{
		APIToken: vaultGet("CLOUDFLARE_API_TOKEN", ""),
		APIKey:   vaultGet("CLOUDFLARE_API_KEY", ""),
		Email:    vaultGet("CLOUDFLARE_EMAIL", ""),
		ZoneID:   vaultGet("CLOUDFLARE_ZONE_ID", ""),
		BaseURL:  "https://api.cloudflare.com/client/v4",
	}
}

func (cf *CloudflareClient) enabled() bool {
	if cf.APIToken != "" {
		return cf.ZoneID != ""
	}
	return cf.APIKey != "" && cf.Email != "" && cf.ZoneID != ""
}

func (cf *CloudflareClient) request(method, path string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}

	req, _ := http.NewRequest(method, cf.BaseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	if cf.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+cf.APIToken)
	} else {
		req.Header.Set("X-Auth-Email", cf.Email)
		req.Header.Set("X-Auth-Key", cf.APIKey)
	}

	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}

func (cf *CloudflareClient) listZones() ([]map[string]interface{}, error) {
	result, err := cf.request("GET", "/zones", nil)
	if err != nil {
		return nil, err
	}
	zonesRaw, _ := result["result"].([]interface{})
	var zones []map[string]interface{}
	for _, z := range zonesRaw {
		if zMap, ok := z.(map[string]interface{}); ok {
			zones = append(zones, zMap)
		}
	}
	return zones, nil
}

func (cf *CloudflareClient) autoDetectZone() bool {
	zones, err := cf.listZones()
	if err != nil || len(zones) == 0 {
		return false
	}
	cf.ZoneID = fmt.Sprintf("%v", zones[0]["id"])
	fmt.Printf("[CLOUDFLARE] Auto-detected zone: %s (%s)\n", zones[0]["name"], cf.ZoneID)
	return true
}

func (cf *CloudflareClient) ensureDNSRecord(subdomain, target string) error {
	if !cf.enabled() {
		fmt.Printf("[CLOUDFLARE] Skipping DNS (not configured)\n")
		return nil
	}

	if cf.ZoneID == "" {
		if !cf.autoDetectZone() {
			return fmt.Errorf("could not detect zone")
		}
	}

	name := subdomain + ".emeraldtools.com"

	result, err := cf.request("GET", "/zones/"+cf.ZoneID+"/dns_records?type=CNAME&name="+name, nil)
	if err != nil {
		return err
	}

	records, _ := result["result"].([]interface{})
	if len(records) > 0 {
		fmt.Printf("[CLOUDFLARE] DNS record %s already exists\n", name)
		return nil
	}

	dnsBody := map[string]interface{}{
		"type":    "CNAME",
		"name":    name,
		"content": target,
		"ttl":     120,
		"proxied": true,
	}

	result, err = cf.request("POST", "/zones/"+cf.ZoneID+"/dns_records", dnsBody)
	if err != nil {
		return err
	}

	if success, _ := result["success"].(bool); success {
		fmt.Printf("[CLOUDFLARE] Created DNS: %s → %s\n", name, target)
	} else {
		errs, _ := result["errors"].([]interface{})
		for _, e := range errs {
			errMap, _ := e.(map[string]interface{})
			fmt.Fprintf(os.Stderr, "[CLOUDFLARE] Error: %v\n", errMap["message"])
		}
	}
	return nil
}

func (cf *CloudflareClient) purgeCache() error {
	if !cf.enabled() {
		return nil
	}

	body := map[string]interface{}{
		"purge_everything": true,
	}

	result, err := cf.request("POST", "/zones/"+cf.ZoneID+"/purge_cache", body)
	if err != nil {
		return err
	}

	if success, _ := result["success"].(bool); success {
		fmt.Printf("[CLOUDFLARE] Cache purged\n")
	}
	return nil
}

func maybeEnsureSubdomain(nicheKeyword string) {
	cf := newCloudflareClient()
	if !cf.enabled() {
		fmt.Printf("[CLOUDFLARE] Not configured, skipping subdomain setup\n")
		return
	}

	target := "emerald-engine.com"
	err := cf.ensureDNSRecord(nicheKeyword, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[CLOUDFLARE] %v\n", err)
	}

	for _, tld := range []string{"tools", "blog", "reviews"} {
		sub := nicheKeyword + "-" + tld
		err := cf.ensureDNSRecord(sub, target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[CLOUDFLARE] %v\n", err)
		}
	}

	cf.purgeCache()
}

func generateSubdomainLinks(nicheKeyword string) string {
	subs := []string{nicheKeyword, nicheKeyword + "-tools", nicheKeyword + "-blog", nicheKeyword + "-reviews"}
	var links []string
	for _, s := range subs {
		links = append(links, fmt.Sprintf(
			`<a href="https://%s.emeraldtools.com" target="_blank">%s.emeraldtools.com</a>`,
			s, s,
		))
	}
	return strings.Join(links, " · ")
}
