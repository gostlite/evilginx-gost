package core

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/playwright-community/playwright-go"
)

// launchStealthBrowser launches a stealth-configured browser
func launchStealthBrowser(pw *playwright.Playwright) (playwright.Browser, error) {
	// Generate random viewport
	widths := []int{1920, 1366, 1536, 1440, 1280}
	heights := []int{1080, 768, 864, 900, 720}
	
	randWidth := widths[time.Now().UnixNano()%int64(len(widths))]
	randHeight := heights[time.Now().UnixNano()%int64(len(heights))]
	
	// Browser launch options with stealth settings
	opts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // Always headless for production
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--disable-software-rasterizer",
			"--disable-features=IsolateOrigins,site-per-process",
			"--disable-blink-features=AutomationControlled",
			"--disable-automation-controller",
			"--disable-popup-blocking",
			"--disable-background-timer-throttling",
			"--disable-backgrounding-occluded-windows",
			"--disable-renderer-backgrounding",
			"--disable-ipc-flooding-protection",
			"--enable-webgl",
			"--enable-3d-apis",
			"--use-gl=egl",
			"--ignore-certificate-errors",
			"--ignore-certificate-errors-spki-list",
			"--disable-web-security",
			"--allow-running-insecure-content",
			fmt.Sprintf("--window-size=%d,%d", randWidth, randHeight),
			"--lang=en-US,en;q=0.9",
			"--timezone=America/New_York",
		},
		IgnoreDefaultArgs: []string{
			"--enable-automation",
			"--disable-background-networking",
		},
	}
	
	browser, err := pw.Chromium.Launch(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to launch stealth browser: %v", err)
	}
	
	return browser, nil
}

// AddStealthToContext injects stealth scripts into browser context
func AddStealthToContext(context playwright.BrowserContext) error {
	stealthScript := `
		// Overwrite navigator properties
		Object.defineProperty(navigator, 'webdriver', {
			get: () => false,
		});
		
		Object.defineProperty(navigator, 'plugins', {
			get: () => [1, 2, 3, 4, 5],
		});
		
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en'],
		});
		
		// Overwrite chrome property
		window.chrome = {
			runtime: {},
			loadTimes: function() {},
			csi: function() {},
			app: {}
		};
		
		// Overwrite permissions
		if (window.navigator.permissions) {
			const originalQuery = window.navigator.permissions.query;
			window.navigator.permissions.query = (parameters) => (
				parameters.name === 'notifications' ?
					Promise.resolve({ state: Notification.permission }) :
					originalQuery(parameters)
			);
		}
		
		// Spoof WebGL
		const getParameter = WebGLRenderingContext.prototype.getParameter;
		WebGLRenderingContext.prototype.getParameter = function(parameter) {
			if (parameter === 37445) {
				return 'Intel Inc.';
			}
			if (parameter === 37446) {
				return 'Intel Iris OpenGL Engine';
			}
			return getParameter(parameter);
		};
		
		// Spoof timezone
		Object.defineProperty(Intl.DateTimeFormat.prototype, 'resolvedOptions', {
			get: function() {
				const original = this.__resolvedOptions ? this.__resolvedOptions() : {};
				return {
					...original,
					timeZone: 'America/New_York'
				};
			}
		});
	`
	
	err := context.AddInitScript(playwright.Script{
		Content: playwright.String(stealthScript),
	})
	return err
}

// generateFingerprint creates a random browser fingerprint
func generateFingerprint() map[string]interface{} {
	timezones := []string{"America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles"}
	languages := []string{"en-US", "en-GB", "en-CA", "en-AU"}
	platforms := []string{"Win32", "MacIntel", "Linux x86_64"}
	
	randTimezone := timezones[randomInt(len(timezones))]
	randLanguage := languages[randomInt(len(languages))]
	randPlatform := platforms[randomInt(len(platforms))]
	
	return map[string]interface{}{
		"userAgent":      GetRandomUserAgent(),
		"platform":       randPlatform,
		"language":       randLanguage,
		"timezone":       randTimezone,
		"screenWidth":    1920,
		"screenHeight":   1080,
		"colorDepth":     24,
		"pixelRatio":     1,
		"hardwareConcurrency": 8,
		"deviceMemory":   8,
	}
}

func randomInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}