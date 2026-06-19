package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type MailchimpClient struct {
	APIKey string
	Prefix string
	ListID string
}

func newMailchimpClient() *MailchimpClient {
	key := vaultGet("MAILCHIMP_API_KEY", "")
	if key == "" {
		return nil
	}
	parts := key[strings.LastIndex(key, "-")+1:]
	return &MailchimpClient{
		APIKey: key,
		Prefix: parts,
		ListID: vaultGet("MAILCHIMP_LIST_ID", ""),
	}
}

func (mc *MailchimpClient) enabled() bool {
	return mc != nil && mc.APIKey != "" && mc.ListID != ""
}

func (mc *MailchimpClient) subscribe(email, niche string) error {
	if !mc.enabled() {
		return nil
	}

	body := map[string]interface{}{
		"email_address": email,
		"status":        "subscribed",
		"merge_fields": map[string]string{
			"NICHE": niche,
		},
		"tags": []string{"emerald-engine", "organic", niche},
	}

	data, _ := json.Marshal(body)
	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/lists/%s/members", mc.Prefix, mc.ListID)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.SetBasicAuth("apikey", mc.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mailchimp subscribe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		fmt.Printf("[MAILCHIMP] Subscribed: %s (niche: %s)\n", email, niche)
	} else {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		if title, ok := result["title"].(string); ok && title != "Member Exists" {
			fmt.Fprintf(os.Stderr, "[MAILCHIMP] %s: %v\n", title, result["detail"])
		}
	}
	return nil
}

func (mc *MailchimpClient) triggerCampaign(niche string) error {
	if !mc.enabled() {
		return nil
	}

	body := map[string]interface{}{
		"type": "automation",
		"settings": map[string]interface{}{
			"title":       fmt.Sprintf("Niche Follow-up: %s", niche),
			"subject_line": fmt.Sprintf("Your %s Resources & Tools", niche),
			"from_name":   "Emerald Engine Team",
			"reply_to":    "noreply@emerald-engine.com",
		},
	}

	data, _ := json.Marshal(body)
	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/campaigns", mc.Prefix)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.SetBasicAuth("apikey", mc.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mailchimp campaign: %w", err)
	}
	defer resp.Body.Close()

	_ = resp.Body.Close()
	return nil
}

func dynamicEmailForm() string {
	mc := newMailchimpClient()
	if mc != nil && mc.ListID != "" {
		return fmt.Sprintf(
			`<div class="ec"><h3>📬 Get Free Tools & Resources</h3><form action="https://%s.api.mailchimp.com/3.0/lists/%s/members" method="post" target="_blank"><input type="email" name="EMAIL" placeholder="Your best email" required style="padding:14px 20px;border:2px solid #ddd;border-radius:50px;width:280px;font-size:1em"><button type="submit" style="background:#667eea;color:white;padding:14px 30px;border:none;border-radius:50px;font-size:1em;font-weight:bold;cursor:pointer;margin-left:10px">Subscribe Free →</button></form></div>`,
			mc.Prefix, mc.ListID,
		)
	}
	return emailForm()
}
