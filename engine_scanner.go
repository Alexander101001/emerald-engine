package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type TargetPlatform struct {
	ID  string
	URL string
}

func testPlatform(platform TargetPlatform, wg *sync.WaitGroup, client *http.Client, results chan<- string) {
	defer wg.Done()

	req, err := http.NewRequest("GET", platform.URL, nil)
	if err != nil {
		results <- fmt.Sprintf("EngineReport:Platform:%s:Status:Error:%v", platform.ID, err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (EmeraldEngine/2.0)")

	resp, err := client.Do(req)
	if err != nil {
		results <- fmt.Sprintf("EngineReport:Platform:%s:Status:Offline", platform.ID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		results <- fmt.Sprintf("EngineReport:Platform:%s:Status:Active", platform.ID)
	} else {
		results <- fmt.Sprintf("EngineReport:Platform:%s:Status:Unreachable:%d", platform.ID, resp.StatusCode)
	}
}

func main() {
	platforms := []TargetPlatform{
		{ID: "github", URL: "https://api.github.com"},
		{ID: "huggingface", URL: "https://huggingface.co/api/models"},
		{ID: "gitlab", URL: "https://gitlab.com/api/v4/projects"},
		{ID: "bitbucket", URL: "https://api.bitbucket.org/2.0/repositories"},
		{ID: "digitalocean", URL: "https://api.digitalocean.com/v2/account"},
		{ID: "linode", URL: "https://api.linode.com/v4/account"},
		{ID: "vultr", URL: "https://api.vultr.com/v2/account"},
		{ID: "heroku", URL: "https://api.heroku.com/apps"},
		{ID: "railway", URL: "https://backboard.railway.app/graphql"},
		{ID: "render", URL: "https://api.render.com/v1/services"},
		{ID: "netlify", URL: "https://api.netlify.com/api/v1/sites"},
		{ID: "vercel", URL: "https://api.vercel.com/v9/projects"},
		{ID: "cloudflare", URL: "https://api.cloudflare.com/client/v4/zones"},
		{ID: "flyio", URL: "https://api.fly.io/v1/apps"},
		{ID: "koyeb", URL: "https://app.koyeb.com/api/v1/apps"},
		{ID: "deno_deploy", URL: "https://api.deno.com/v1/projects"},
		{ID: "replit", URL: "https://replit.com/api/v1/user"},
		{ID: "cyclic", URL: "https://api.cyclic.sh/v1/apps"},
		{ID: "adaptable", URL: "https://api.adaptable.io/v1/apps"},
		{ID: "pythonanywhere", URL: "https://www.pythonanywhere.com/api/v0/user"},
		{ID: "scaleway", URL: "https://api.scaleway.com/v1/instances"},
		{ID: "civo", URL: "https://api.civo.com/v2/instances"},
		{ID: "hetzner", URL: "https://api.hetzner.cloud/v1/servers"},
		{ID: "upcloud", URL: "https://api.upcloud.com/1.3/account"},
		{ID: "ovhcloud", URL: "https://api.ovh.com/1.0/me"},
		{ID: "clever_cloud", URL: "https://api.clever-cloud.com/v2/products"},
		{ID: "scalingo", URL: "https://api.scalingo.com/v1/apps"},
		{ID: "glitch", URL: "https://api.glitch.com/v1/projects"},
		{ID: "deta", URL: "https://api.deta.sh/v1/projects"},
		{ID: "mogenius", URL: "https://api.mogenius.com/v1/projects"},
		{ID: "alwaysdata", URL: "https://api.alwaysdata.com/v1/account"},
		{ID: "exoscale", URL: "https://api.exoscale.com/v1/compute"},
		{ID: "ionos", URL: "https://api.ionos.com/cloudapi/v5/datacenters"},
		{ID: "pulumi", URL: "https://api.pulumi.com/api/user"},
		{ID: "terraform_cloud", URL: "https://app.terraform.io/api/v2/account/details"},
		{ID: "supabase", URL: "https://api.supabase.com/v1/projects"},
		{ID: "neon", URL: "https://console.neon.tech/api/v2/projects"},
		{ID: "planetscale", URL: "https://api.planetscale.com/v1/organizations"},
		{ID: "mongodb_atlas", URL: "https://cloud.mongodb.com/api/atlas/v1.0/groups"},
		{ID: "redis_cloud", URL: "https://api.redislabs.com/v1/subscriptions"},
		{ID: "cloudamqp", URL: "https://customer.cloudamqp.com/api/instances"},
		{ID: "confluent_cloud", URL: "https://api.confluent.cloud/org/v2/organizations"},
		{ID: "ably", URL: "https://api.ably.io/v1/apps"},
		{ID: "sentry", URL: "https://sentry.io/api/0/projects"},
		{ID: "datadog", URL: "https://api.datadoghq.com/api/v1/validate"},
		{ID: "grafana_cloud", URL: "https://grafana.com/api/instances"},
		{ID: "betterstack", URL: "https://uptime.betterstack.com/api/v2/monitors"},
		{ID: "checkly", URL: "https://api.checklyhq.com/v1/checks"},
		{ID: "algolia", URL: "https://api.algolia.com/1/indexes"},
		{ID: "logz_io", URL: "https://api.logz.io/v1/account"},
	}

	outputFile := os.Getenv("SCANNER_OUTPUT")
	if outputFile == "" {
		outputFile = "/tmp/scanner_report.txt"
	}
	maxConcurrent := 10
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	results := make(chan string, len(platforms))
	httpClient := &http.Client{Timeout: 6 * time.Second}

	for _, p := range platforms {
		wg.Add(1)
		sem <- struct{}{}
		go func(pl TargetPlatform) {
			defer func() { <-sem }()
			testPlatform(pl, &wg, httpClient, results)
		}(p)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var lines []string
	for r := range results {
		fmt.Println(r)
		lines = append(lines, r)
	}
	fmt.Println("EngineReport:Cycle:Completed")

	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err == nil {
			for _, l := range lines {
				f.WriteString(l + "\n")
			}
			f.WriteString("EngineReport:Cycle:Completed\n")
			f.Close()
		}
	}
}
