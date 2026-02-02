package core

import (
	"testing"
)

func TestEncryptDecryptParams(t *testing.T) {
	key := "test-encryption-key-123"
	params := map[string]string{
		"email":     "test@example.com",
		"from_name": "John Doe",
		"campaign":  "Q1-2026",
	}

	// Test encryption
	encrypted, err := EncryptParams(params, key)
	if err != nil {
		t.Fatalf("EncryptParams failed: %v", err)
	}

	if encrypted == "" {
		t.Fatal("Encrypted string is empty")
	}

	// Test decryption
	decrypted, err := DecryptParams(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptParams failed: %v", err)
	}

	// Verify all params match
	if len(decrypted) != len(params) {
		t.Fatalf("Decrypted params count mismatch: got %d, want %d", len(decrypted), len(params))
	}

	for key, value := range params {
		if decrypted[key] != value {
			t.Errorf("Param mismatch for key %s: got %s, want %s", key, decrypted[key], value)
		}
	}
}

func TestBase64Fallback(t *testing.T) {
	params := map[string]string{
		"email": "test@example.com",
		"name":  "Test User",
	}

	// Encode with base64 (legacy)
	encoded, err := EncodeParamsBase64(params)
	if err != nil {
		t.Fatalf("EncodeParamsBase64 failed: %v", err)
	}

	// Decrypt should fallback to base64 decoding
	decrypted, err := DecryptParams(encoded, "some-key")
	if err != nil {
		t.Fatalf("DecryptParams with base64 fallback failed: %v", err)
	}

	// Verify params match
	for key, value := range params {
		if decrypted[key] != value {
			t.Errorf("Param mismatch for key %s: got %s, want %s", key, decrypted[key], value)
		}
	}
}

func TestEncryptWithEmptyKey(t *testing.T) {
	params := map[string]string{
		"test": "value",
	}

	// Empty key should use base64
	encrypted, err := EncryptParams(params, "")
	if err != nil {
		t.Fatalf("EncryptParams with empty key failed: %v", err)
	}

	// Should be decodable with base64
	decrypted, err := DecodeParamsBase64(encrypted)
	if err != nil {
		t.Fatalf("DecodeParamsBase64 failed: %v", err)
	}

	if decrypted["test"] != "value" {
		t.Errorf("Param mismatch: got %s, want value", decrypted["test"])
	}
}

func TestValidateLureHostname(t *testing.T) {
	tests := []struct {
		name            string
		lureHostname    string
		phishletHostname string
		expectError     bool
	}{
		{
			name:            "exact match",
			lureHostname:    "accounts.phishing.com",
			phishletHostname: "accounts.phishing.com",
			expectError:     false,
		},
		{
			name:            "subdomain change",
			lureHostname:    "login.phishing.com",
			phishletHostname: "accounts.phishing.com",
			expectError:     false,
		},
		{
			name:            "empty subdomain (TLD-only)",
			lureHostname:    "phishing.com",
			phishletHostname: "accounts.phishing.com",
			expectError:     false,
		},
		{
			name:            "different base domain",
			lureHostname:    "accounts.evil.com",
			phishletHostname: "accounts.phishing.com",
			expectError:     true,
		},
		{
			name:            "nested subdomain",
			lureHostname:    "www.accounts.phishing.com",
			phishletHostname: "accounts.phishing.com",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLureHostname(tt.lureHostname, tt.phishletHostname)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
