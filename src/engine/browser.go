package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type BrowserResult struct {
	HTML       string
	Screenshot []byte
	StatusCode int
	Error      string
	DurationMs int64
}

var chromiumPath string

func init() {
	for _, p := range []string{"google-chrome-stable", "google-chrome", "chromium-browser", "chromium"} {
		if _, err := exec.LookPath(p); err == nil {
			chromiumPath = p
			break
		}
	}
	if chromiumPath == "" {
		fmt.Println("[BROWSER] Chromium not found — deep pings disabled")
	} else {
		fmt.Printf("[BROWSER] Found: %s\n", chromiumPath)
	}
}

func browserDeepPing(url string) *BrowserResult {
	if chromiumPath == "" {
		return &BrowserResult{Error: "chromium not available"}
	}

	start := time.Now()

	tmpDir, err := os.MkdirTemp("", "emerald-chrome-*")
	if err != nil {
		return &BrowserResult{Error: fmt.Sprintf("tmpdir: %v", err)}
	}
	defer os.RemoveAll(tmpDir)

	screenshotPath := filepath.Join(tmpDir, "screenshot.png")

	args := []string{
		"--headless=new",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-software-rasterizer",
		"--disable-extensions",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-default-apps",
		"--mute-audio",
		"--hide-scrollbars",
		fmt.Sprintf("--user-data-dir=%s", filepath.Join(tmpDir, "profile")),
		fmt.Sprintf("--screenshot=%s", screenshotPath),
		"--dump-dom",
		"--window-size=1920,1080",
		"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		url,
	}

	cmd := exec.Command(chromiumPath, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	duration := time.Since(start).Milliseconds()

	result := &BrowserResult{
		HTML:       stdout.String(),
		StatusCode: 0,
		DurationMs: duration,
	}

	if runErr != nil {
		result.Error = fmt.Sprintf("chromium error: %v (stderr: %s)", runErr, trimStr(stderr.String(), 500))
		return result
	}

	if data, err := os.ReadFile(screenshotPath); err == nil {
		result.Screenshot = data
	}

	result.StatusCode = 200
	return result
}

func trimStr(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
