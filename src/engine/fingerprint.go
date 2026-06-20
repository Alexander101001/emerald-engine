package main

import (
	"fmt"
	"net/http"
	"time"
)

type BrowserFingerprint struct {
	UserAgent      string `json:"user_agent"`
	CanvasHash     string `json:"canvas_hash"`
	WebGLHash      string `json:"webgl_hash"`
	ViewportWidth  int    `json:"viewport_width"`
	ViewportHeight int    `json:"viewport_height"`
	ColorDepth     int    `json:"color_depth"`
	DeviceMemory   int    `json:"device_memory"`
	HardwareConcurrency int `json:"hardware_concurrency"`
	TimeZone       string `json:"timezone"`
	Platform       string `json:"platform"`
	Languages      string `json:"languages"`
	TouchSupport   bool   `json:"touch_support"`
}

type FingerprintEngine struct {
	profiles []BrowserFingerprint
}

var fingerprintEngine *FingerprintEngine

func initFingerprintEngine() *FingerprintEngine {
	fe := &FingerprintEngine{
		profiles: []BrowserFingerprint{
			{
				UserAgent:           "Mozilla/5.0 (Linux; Android 10; SM-N970F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",
				CanvasHash:          "a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9",
				WebGLHash:           "b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3",
				ViewportWidth:       412, ViewportHeight: 915, ColorDepth: 24,
				DeviceMemory: 4, HardwareConcurrency: 8,
				TimeZone: "Asia/Kolkata", Platform: "Linux aarch64",
				Languages: "en-IN,en-US;q=0.9,hi;q=0.8", TouchSupport: true,
			},
			{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
				CanvasHash:          "c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0",
				WebGLHash:           "d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1",
				ViewportWidth:       1920, ViewportHeight: 1080, ColorDepth: 24,
				DeviceMemory: 8, HardwareConcurrency: 16,
				TimeZone: "America/New_York", Platform: "Win32",
				Languages: "en-US,en;q=0.9", TouchSupport: false,
			},
			{
				UserAgent:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
				CanvasHash:          "e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4",
				WebGLHash:           "f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5",
				ViewportWidth:       1512, ViewportHeight: 982, ColorDepth: 30,
				DeviceMemory: 16, HardwareConcurrency: 12,
				TimeZone: "Europe/London", Platform: "MacIntel",
				Languages: "en-GB,en;q=0.9", TouchSupport: false,
			},
			{
				UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
				CanvasHash:          "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
				WebGLHash:           "b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1",
				ViewportWidth:       2560, ViewportHeight: 1440, ColorDepth: 24,
				DeviceMemory: 32, HardwareConcurrency: 32,
				TimeZone: "Europe/Berlin", Platform: "Linux x86_64",
				Languages: "de-DE,de;q=0.9,en;q=0.8", TouchSupport: false,
			},
			{
				UserAgent:           "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1",
				CanvasHash:          "c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2",
				WebGLHash:           "d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3",
				ViewportWidth:       390, ViewportHeight: 844, ColorDepth: 24,
				DeviceMemory: 6, HardwareConcurrency: 6,
				TimeZone: "America/Los_Angeles", Platform: "iPhone",
				Languages: "en-US,en;q=0.9", TouchSupport: true,
			},
		},
	}
	fingerprintEngine = fe
	return fe
}

func (fe *FingerprintEngine) Random() BrowserFingerprint {
	n := time.Now().UnixNano()
	return fe.profiles[n%int64(len(fe.profiles))]
}

func (fe *FingerprintEngine) RotateDay() BrowserFingerprint {
	day := time.Now().YearDay()
	return fe.profiles[day%len(fe.profiles)]
}

func ApplyFingerprintHeaders(req *http.Request, fp BrowserFingerprint) {
	req.Header.Set("User-Agent", fp.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", fp.Languages)
	req.Header.Set("Sec-CH-UA", fmt.Sprintf(`"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"`))
	req.Header.Set("Sec-CH-UA-Mobile", fmt.Sprintf("%t", fp.TouchSupport))
	req.Header.Set("Sec-CH-UA-Platform", fp.Platform)
	req.Header.Set("Sec-CH-Viewport-Width", fmt.Sprintf("%d", fp.ViewportWidth))
	req.Header.Set("Sec-CH-Viewport-Height", fmt.Sprintf("%d", fp.ViewportHeight))
	req.Header.Set("X-Canvas-Fingerprint", fp.CanvasHash)
	req.Header.Set("X-WebGL-Fingerprint", fp.WebGLHash)
	req.Header.Set("X-Device-Memory", fmt.Sprintf("%d", fp.DeviceMemory))
	req.Header.Set("X-Hardware-Concurrency", fmt.Sprintf("%d", fp.HardwareConcurrency))
	req.Header.Set("X-Color-Depth", fmt.Sprintf("%d", fp.ColorDepth))
	req.Header.Set("X-TimeZone", fp.TimeZone)
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

func GenerateCanvasFingerprintJS() string {
	return `(function(){
  var c=document.createElement('canvas');
  c.width=280;c.height=60;
  var ctx=c.getContext('2d');
  ctx.textBaseline='top';
  ctx.font='14px Arial';
  ctx.fillStyle='#f60';
  ctx.fillRect(0,0,280,60);
  ctx.fillStyle='#069';
  ctx.fillText('EMERALD-ENGINE-FINGERPRINT',10,20);
  var fp=c.toDataURL();
  var hash=0;
  for(var i=0;i<fp.length;i++){
    var ch=fp.charCodeAt(i);
    hash=((hash<<5)-hash)+ch;
    hash=hash&hash;
  }
  document.cookie='_fp='+hash+';path=/;max-age=86400';
})();
`
}

func GenerateWebGLFingerprintJS() string {
	return `(function(){
  try{
    var c=document.createElement('canvas');
    var gl=c.getContext('webgl')||c.getContext('experimental-webgl');
    if(!gl)return;
    var ext=gl.getExtension('WEBGL_debug_renderer_info');
    if(ext){
      document.cookie='_wgl='+gl.getParameter(ext.UNMASKED_RENDERER_WEBGL)+'|'+gl.getParameter(ext.UNMASKED_VENDOR_WEBGL)+';path=/;max-age=86400';
    }
  }catch(e){}
})();
`
}

func GenerateFingerprintCollector() string {
	return fmt.Sprintf(`<script>%s%s</script>`, GenerateCanvasFingerprintJS(), GenerateWebGLFingerprintJS())
}
