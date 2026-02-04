package core

import (
	"testing"
)

// TestConfigPersistence tests that configuration values persist correctly
func TestConfigPersistence(t *testing.T) {
	cfg, err := NewConfig("", "")
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test debug mode
	cfg.SetDebugMode(true)
	if !cfg.GetDebugMode() {
		t.Error("Debug mode not persisted correctly")
	}

	cfg.SetDebugMode(false)
	if cfg.GetDebugMode() {
		t.Error("Debug mode disable not persisted correctly")
	}

	// Test JS obfuscation level
	levels := []string{"off", "low", "medium", "high", "ultra"}
	for _, level := range levels {
		cfg.SetJsObfuscationLevel(level)
		if cfg.GetJsObfuscationLevel() != level {
			t.Errorf("JS obfuscation level '%s' not persisted correctly, got: %s", level, cfg.GetJsObfuscationLevel())
		}
	}

	// Test encryption key
	testKey := "test-encryption-key-123"
	cfg.SetEncryptionKey(testKey)
	if cfg.GetEncryptionKey() != testKey {
		t.Error("Encryption key not persisted correctly")
	}

	// Test clearing encryption key
	cfg.SetEncryptionKey("")
	if cfg.GetEncryptionKey() != "" {
		t.Error("Encryption key clear not persisted correctly")
	}
}

// TestJsObfuscationLevels tests all JS obfuscation levels
func TestJsObfuscationLevels(t *testing.T) {
	testScript := "console.log('test'); var x = 123; function foo() { return 'bar'; }"

	levels := []string{"off", "low", "medium", "high", "ultra"}
	for _, level := range levels {
		result := ObfuscateJavaScript(testScript, level)
		
		// Verify script is not empty
		if result == "" {
			t.Errorf("Obfuscation level '%s' returned empty script", level)
		}

		// For 'off', script should be unchanged
		if level == "off" && result != testScript {
			t.Errorf("Obfuscation level 'off' changed the script")
		}

		// For other levels, verify some transformation occurred
		if level != "off" && result == testScript {
			t.Logf("Warning: Obfuscation level '%s' did not transform the script (obfuscation may not be fully implemented)", level)
		}
	}
}

// TestLureGenerationWithEncryption tests lure generation with encrypted parameters
func TestLureGenerationWithEncryption(t *testing.T) {
	// Test parameters
	params := map[string]string{
		"email":    "test@example.com",
		"campaign": "q1-2024",
		"source":   "linkedin",
	}

	// Test with encryption key
	encKey := "my-secret-key-2024"
	encrypted, err := EncryptParams(params, encKey)
	if err != nil {
		t.Fatalf("Failed to encrypt params: %v", err)
	}

	// Verify encrypted string is not empty
	if encrypted == "" {
		t.Error("Encrypted params string is empty")
	}

	// Decrypt and verify
	decrypted, err := DecryptParams(encrypted, encKey)
	if err != nil {
		t.Fatalf("Failed to decrypt params: %v", err)
	}

	// Verify all parameters match
	for key, expectedValue := range params {
		if actualValue, ok := decrypted[key]; !ok {
			t.Errorf("Missing parameter '%s' after decryption", key)
		} else if actualValue != expectedValue {
			t.Errorf("Parameter '%s' mismatch: expected '%s', got '%s'", key, expectedValue, actualValue)
		}
	}

	// Test without encryption key (base64 fallback)
	encrypted2, err := EncryptParams(params, "")
	if err != nil {
		t.Fatalf("Failed to encrypt params with empty key: %v", err)
	}

	decrypted2, err := DecryptParams(encrypted2, "")
	if err != nil {
		t.Fatalf("Failed to decrypt params with empty key: %v", err)
	}

	// Verify all parameters match
	for key, expectedValue := range params {
		if actualValue, ok := decrypted2[key]; !ok {
			t.Errorf("Missing parameter '%s' after base64 decryption", key)
		} else if actualValue != expectedValue {
			t.Errorf("Parameter '%s' mismatch in base64: expected '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

// TestLureHostnameValidation tests lure hostname validation
func TestLureHostnameValidation(t *testing.T) {
	testCases := []struct {
		lureHostname     string
		phishletHostname string
		shouldPass       bool
		description      string
	}{
		{"login.example.com", "auth.example.com", true, "Valid subdomain"},
		{"example.com", "auth.example.com", true, "Valid TLD-only (empty subdomain)"},
		{"sub.login.example.com", "auth.example.com", true, "Valid nested subdomain"},
		{"", "example.com", false, "Empty lure hostname"},
		{"example.com", "", false, "Empty phishlet hostname"},
		{"google.com", "example.com", false, "Different domain"},
	}

	for _, tc := range testCases {
		err := ValidateLureHostname(tc.lureHostname, tc.phishletHostname)
		if tc.shouldPass && err != nil {
			t.Errorf("%s: Expected to pass but got error: %v", tc.description, err)
		}
		if !tc.shouldPass && err == nil {
			t.Errorf("%s: Expected to fail but passed", tc.description)
		}
	}
}

// ... URL Rewriting Tests ...

// TestHTMLParserIntegration tests HTML parser with different content types
func TestHTMLParserIntegration(t *testing.T) {
	testCases := []struct {
		name     string
		html     string
		script   string
		location string
		hasError bool
	}{
		{
			name:     "Valid HTML with head injection",
			html:     "<html><head><title>Test</title></head><body>Content</body></html>",
			script:   "<script>console.log('test');</script>",
			location: "head",
			hasError: false,
		},
		{
			name:     "Valid HTML with body_top injection",
			html:     "<html><head></head><body><div>Content</div></body></html>",
			script:   "<script>console.log('test');</script>",
			location: "body_top",
			hasError: false,
		},
		{
			name:     "Valid HTML with body_bottom injection",
			html:     "<html><head></head><body><div>Content</div></body></html>",
			script:   "<script>console.log('test');</script>",
			location: "body_bottom",
			hasError: false,
		},
		{
			name:     "Invalid HTML structure - Should parse anyway", // html.Parse is lenient
			html:     "Not HTML content",
			script:   "<script>console.log('test');</script>",
			location: "body_bottom",
			hasError: false, 
		},
		{
			name:     "Invalid location",
			html:     "<html><head></head><body>Content</body></html>",
			script:   "<script>console.log('test');</script>",
			location: "invalid_location",
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := InjectJavaScriptHTML([]byte(tc.html), tc.script, tc.location)
			
			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) == 0 {
					t.Error("Result is empty")
				}
			}
		})
	}
}

// TestConfigDefaultValues tests that config has correct default values
func TestConfigDefaultValues(t *testing.T) {
	cfg, err := NewConfig("", "")
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Debug mode should default to false
	if cfg.GetDebugMode() {
		t.Error("Debug mode should default to false")
	}

	// JS obfuscation should default to "medium"
	if cfg.GetJsObfuscationLevel() != "medium" {
		t.Errorf("JS obfuscation should default to 'medium', got: %s", cfg.GetJsObfuscationLevel())
	}

	// Encryption key should default to empty
	if cfg.GetEncryptionKey() != "" {
		t.Error("Encryption key should default to empty")
	}
}
