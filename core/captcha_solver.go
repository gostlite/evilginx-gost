package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kgretzky/evilginx2/log"
	"github.com/playwright-community/playwright-go"
)

// CaptchaSolver handles CAPTCHA solving
type CaptchaSolver struct {
	service string
	apiKey  string
	client  *http.Client
}

// CaptchaInfo contains CAPTCHA details
type CaptchaInfo struct {
	Type      string // "recaptcha_v2", "recaptcha_v3", "hcaptcha"
	SiteKey   string
	PageURL   string
	Version   string
	Action    string // For reCAPTCHA v3
	Invisible bool
}

// NewCaptchaSolver creates a new solver instance
func NewCaptchaSolver(apiKey, service string) *CaptchaSolver {
	return &CaptchaSolver{
		service: service,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Solve solves a CAPTCHA
func (cs *CaptchaSolver) Solve(info *CaptchaInfo) (string, error) {
	log.Debug("[CAPTCHA] Solving %s for %s", info.Type, info.PageURL)
	
	switch cs.service {
	case "2captcha":
		return cs.solve2Captcha(info)
	case "anti-captcha":
		return cs.solveAntiCaptcha(info)
	case "capsolver":
		return cs.solveCapSolver(info)
	default:
		return "", fmt.Errorf("unsupported CAPTCHA service: %s", cs.service)
	}
}

// solve2Captcha solves using 2Captcha service
func (cs *CaptchaSolver) solve2Captcha(info *CaptchaInfo) (string, error) {
	var method string
	var params map[string]string
	
	switch info.Type {
	case "recaptcha_v2":
		method = "userrecaptcha"
		params = map[string]string{
			"key":       cs.apiKey,
			"method":    method,
			"googlekey": info.SiteKey,
			"pageurl":   info.PageURL,
			"json":      "1",
		}
	case "recaptcha_v3":
		method = "userrecaptcha"
		params = map[string]string{
			"key":       cs.apiKey,
			"method":    method,
			"googlekey": info.SiteKey,
			"pageurl":   info.PageURL,
			"version":   "v3",
			"action":    info.Action,
			"min_score": "0.3",
			"json":      "1",
		}
	case "hcaptcha":
		method = "hcaptcha"
		params = map[string]string{
			"key":       cs.apiKey,
			"method":    method,
			"sitekey":   info.SiteKey,
			"pageurl":   info.PageURL,
			"json":      "1",
		}
	default:
		return "", fmt.Errorf("unsupported CAPTCHA type for 2captcha: %s", info.Type)
	}
	
	// Submit CAPTCHA
	inURL := "http://2captcha.com/in.php"
	resp, err := cs.client.PostForm(inURL, convertToURLValues(params))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var inResult map[string]interface{}
	json.Unmarshal(body, &inResult)
	
	if inResult["status"] != float64(1) {
		return "", fmt.Errorf("failed to submit CAPTCHA: %v", inResult["request"])
	}
	
	captchaID := inResult["request"].(string)
	
	// Poll for solution
	for i := 0; i < 60; i++ {
		time.Sleep(5 * time.Second)
		
		resURL := "http://2captcha.com/res.php"
		resParams := url.Values{
			"key":    {cs.apiKey},
			"action": {"get"},
			"id":     {captchaID},
			"json":   {"1"},
		}
		
		resp, err := cs.client.Get(resURL + "?" + resParams.Encode())
		if err != nil {
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		var resResult map[string]interface{}
		json.Unmarshal(body, &resResult)
		
		if resResult["status"] == float64(1) {
			return resResult["request"].(string), nil
		}
		
		if resResult["request"] == "ERROR_CAPTCHA_UNSOLVABLE" {
			return "", fmt.Errorf("CAPTCHA unsolvable")
		}
	}
	
	return "", fmt.Errorf("timeout waiting for CAPTCHA solution")
}

// solveAntiCaptcha solves using Anti-Captcha service
func (cs *CaptchaSolver) solveAntiCaptcha(info *CaptchaInfo) (string, error) {
	// Similar implementation for Anti-Captcha
	// (Implementation omitted for brevity, similar to 2Captcha)
	return "", fmt.Errorf("anti-captcha not implemented in this example")
}

// solveCapSolver solves using CapSolver service
func (cs *CaptchaSolver) solveCapSolver(info *CaptchaInfo) (string, error) {
	// Similar implementation for CapSolver
	return "", fmt.Errorf("capsolver not implemented in this example")
}

// detectReCaptcha detects reCAPTCHA on page
func (pm *PuppetMaster) detectReCaptcha(page playwright.Page) (*CaptchaInfo, error) {
	// Check for reCAPTCHA iframe
	iframeSelector := "iframe[src*='google.com/recaptcha'], iframe[src*='gstatic.com/recaptcha']"
	iframeCount, _ := page.Locator(iframeSelector).Count()
	
	if iframeCount > 0 {
		// Extract sitekey
		jsScript := `
			(() => {
				// Check for reCAPTCHA v2
				const recaptchaDiv = document.querySelector('.g-recaptcha');
				if (recaptchaDiv && recaptchaDiv.dataset.sitekey) {
					return {
						type: 'recaptcha_v2',
						sitekey: recaptchaDiv.dataset.sitekey,
						pageurl: window.location.href
					};
				}
				
				// Check for reCAPTCHA v3
				if (typeof grecaptcha !== 'undefined' && grecaptcha.render) {
					const containers = document.querySelectorAll('[id^="g-recaptcha"]');
					for (const container of containers) {
						const widgetId = grecaptcha.render(container);
						const sitekey = grecaptcha.getResponse(widgetId);
						if (sitekey) {
							return {
								type: 'recaptcha_v3',
								sitekey: container.dataset.sitekey || '',
								pageurl: window.location.href,
								action: 'submit'
							};
						}
					}
				}
				
				// Check for invisible reCAPTCHA
				const invisibleDivs = document.querySelectorAll('[data-callback]');
				for (const div of invisibleDivs) {
					if (div.innerHTML.includes('recaptcha')) {
						return {
							type: 'recaptcha_v2',
							sitekey: div.dataset.sitekey || '',
							pageurl: window.location.href,
							invisible: true
						};
					}
				}
				
				return null;
			})()
		`
		
		result, err := page.Evaluate(jsScript)
		if err == nil && result != nil {
			if resultMap, ok := result.(map[string]interface{}); ok {
				return &CaptchaInfo{
					Type:    resultMap["type"].(string),
					SiteKey: resultMap["sitekey"].(string),
					PageURL: resultMap["pageurl"].(string),
					Version: strings.Replace(resultMap["type"].(string), "recaptcha_", "", 1),
				}, nil
			}
		}
	}
	
	return nil, fmt.Errorf("reCAPTCHA not detected")
}

// detectHCaptcha detects hCaptcha on page
func (pm *PuppetMaster) detectHCaptcha(page playwright.Page) (*CaptchaInfo, error) {
	// Check for hCaptcha iframe
	iframeSelector := "iframe[src*='hcaptcha.com'], iframe[src*='hcaptcha.com/captcha']"
	iframeCount, _ := page.Locator(iframeSelector).Count()
	
	if iframeCount > 0 {
		jsScript := `
			(() => {
				// Look for hCaptcha div
				const hcaptchaDiv = document.querySelector('[data-hcaptcha-widget-id], [id^="hcaptcha"]');
				if (hcaptchaDiv) {
					// Try to extract sitekey
					let sitekey = '';
					const scriptTags = document.querySelectorAll('script');
					for (const script of scriptTags) {
						if (script.innerHTML.includes('hcaptcha') && script.innerHTML.includes('sitekey')) {
							const match = script.innerHTML.match(/sitekey['"]?\\s*[:=]\\s*['"]([^'"]+)['"]/);
							if (match) {
								sitekey = match[1];
								break;
							}
						}
					}
					
					return {
						type: 'hcaptcha',
						sitekey: sitekey,
						pageurl: window.location.href
					};
				}
				return null;
			})()
		`
		
		result, err := page.Evaluate(jsScript)
		if err == nil && result != nil {
			if resultMap, ok := result.(map[string]interface{}); ok {
				return &CaptchaInfo{
					Type:    "hcaptcha",
					SiteKey: resultMap["sitekey"].(string),
					PageURL: resultMap["pageurl"].(string),
				}, nil
			}
		}
	}
	
	return nil, fmt.Errorf("hCaptcha not detected")
}

func convertToURLValues(m map[string]string) url.Values {
	values := url.Values{}
	for k, v := range m {
		values.Set(k, v)
	}
	return values
}