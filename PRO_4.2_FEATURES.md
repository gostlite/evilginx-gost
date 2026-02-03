# Evilginx Pro 4.2 Features - Usage Guide

This guide covers the new Pro 4.2 features that have been integrated into your Evilginx installation.

## Table of Contents
1. [Configuration Options](#configuration-options)
2. [JavaScript Injection with Location Control](#javascript-injection-with-location-control)
3. [Encrypted Lure Parameters](#encrypted-lure-parameters)
4. [URL Rewriting (Anti-Detection)](#url-rewriting-anti-detection)
5. [Examples](#examples)

---

## Configuration Options

### Debug Mode

Enable verbose debug logging for troubleshooting.

```bash
# Enable debug mode
config debug on

# Disable debug mode
config debug off

# View current setting
config
```

**When to use:** Troubleshooting phishlet issues, testing new configurations, or investigating proxy behavior.

---

### JavaScript Obfuscation

Control the level of JavaScript obfuscation applied to injected scripts.

```bash
# Set obfuscation level
config js_obfuscation <level>

# Available levels:
config js_obfuscation off      # No obfuscation
config js_obfuscation low      # Basic minification
config js_obfuscation medium   # Minification + string encoding (recommended)
config js_obfuscation high     # Medium + variable renaming
config js_obfuscation ultra    # Maximum obfuscation (may slow page loads)
```

**Default:** `medium`

**Recommendation:** Use `medium` for most scenarios. Use `high` or `ultra` only if you're experiencing detection issues.

---

### Encryption Key for Lure Parameters

Set an AES-256 encryption key for lure custom parameters.

```bash
# Set encryption key
config enc_key my-secret-passphrase-123

# Clear encryption key (fallback to base64)
config enc_key ""

# View current setting (key is masked)
config
```

**How it works:**
- When set, custom lure parameters are encrypted using AES-256-GCM
- When empty, parameters use base64 encoding (backward compatible)
- Encryption key is hashed with SHA-256 before use

**Example encrypted URL:**
```
https://phish.example.com/login?yui=Ag3kF9mP...encrypted_data...
```

---

## JavaScript Injection with Location Control

### Overview

The HTML parser replaces regex-based JavaScript injection with proper HTML parsing, allowing you to specify where scripts are injected.

### Phishlet Configuration

Add the `location` field to your `js_inject` configuration:

```yaml
js_inject:
  - trigger_domains: ['login.example.com']
    trigger_paths: ['/login']
    location: 'body_bottom'  # Optional: head, body_top, body_bottom
    script: |
      console.log('Injected at body bottom');
```

### Injection Locations

| Location | Description | Use Case |
|----------|-------------|----------|
| `head` | End of `<head>` tag | Early initialization, load external libraries |
| `body_top` | Beginning of `<body>` tag | DOM manipulation before content loads |
| `body_bottom` | End of `<body>` tag (default) | Standard scripts, analytics, tracking |

### Example: Multiple Injection Points

```yaml
js_inject:
  # Inject tracking at head
  - trigger_domains: ['login.example.com']
    trigger_paths: ['/.*']
    location: 'head'
    script: |
      window.trackingId = 'session-{id}';
  
  # Inject form handler at body_bottom
  - trigger_domains: ['login.example.com']
    trigger_paths: ['/login']
    location: 'body_bottom'
    script: |
      document.querySelector('form').addEventListener('submit', function(e) {
        console.log('Form submitted');
      });
```

### Features

✅ **Proper HTML Parsing** - Uses `golang.org/x/net/html` instead of regex  
✅ **CSP Compatibility** - Preserves `nonce` attributes for Content Security Policy  
✅ **Graceful Fallback** - Falls back to regex if HTML parsing fails  
✅ **Automatic Obfuscation** - Applies JS obfuscation based on config level

---

## Encrypted Lure Parameters

### Setup

1. Set an encryption key:
```bash
config enc_key my-super-secret-key-2024
```

2. Create a lure with custom parameters:
```bash
lures create <phishlet> <redirect_url>
lures get-url <lure_id> email=victim@example.com campaign=q1-2024
```

3. The generated URL will have encrypted parameters:
```
https://phish.example.com/login?yui=Ag3kF9mP2x...encrypted...
```

### How It Works

**Without encryption key:**
```
Parameters: {"email": "victim@example.com"}
Encoded: base64({"email": "victim@example.com"})
URL: ?yui=eyJlbWFpbCI6InZpY3RpbUBleGFtcGxlLmNvbSJ9
```

**With encryption key:**
```
Parameters: {"email": "victim@example.com"}
Encrypted: AES-256-GCM(JSON, SHA256(key))
URL: ?yui=Ag3kF9mP2x...encrypted_data...
```

### Decryption

Evilginx automatically decrypts parameters when the victim visits the lure URL. The decrypted parameters are available in the session.

### Backward Compatibility

- If encryption key is not set, parameters use base64 encoding
- If decryption fails, Evilginx falls back to base64 decoding
- Existing lures continue to work without changes

---

## URL Rewriting (Anti-Detection)

### Overview

URL rewriting dynamically changes phishing URLs to evade Safe Browsing pattern detection by modifying paths and query parameters.

### Phishlet Configuration

Add `rewrite_urls` section to your phishlet:

```yaml
rewrite_urls:
  - trigger:
      domains: ['login.example.com']
      paths: ['/auth/login']
    rewrite:
      path: '/secure/verify'
      query:
        - key: 'session'
          value: 'token-{id}'
        - key: 'ref'
          value: 'portal-{id}'
      exclude_keys: ['debug', 'test']
```

### Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `trigger.domains` | ✅ | List of domains to trigger rewriting |
| `trigger.paths` | ✅ | Regex patterns for paths to rewrite |
| `rewrite.path` | ✅ | New path to use |
| `rewrite.query` | ❌ | Query parameters to add |
| `rewrite.exclude_keys` | ❌ | Original query keys to exclude |

### Query Parameter Placeholder

The `{id}` placeholder in query values is replaced with a unique session identifier:

```yaml
query:
  - key: 'session'
    value: 'token-{id}'  # Becomes: token-abc123def456
```

### Example: M365 URL Rewriting

**Original URL:**
```
https://login.microsoftonline.com/common/oauth2/authorize?client_id=...
```

**Phishlet config:**
```yaml
rewrite_urls:
  - trigger:
      domains: ['login.microsoftonline.com']
      paths: ['/common/oauth2/authorize']
    rewrite:
      path: '/secure/portal/verify'
      query:
        - key: 'sid'
          value: 'session-{id}'
        - key: 'app'
          value: 'office-{id}'
      exclude_keys: ['client_id', 'redirect_uri']
```

**Rewritten URL:**
```
https://login.microsoftonline.com/secure/portal/verify?sid=session-abc123&app=office-abc123
```

### How It Works

1. **Trigger Detection**: When a victim visits a URL matching the trigger
2. **URL Rewriting**: Evilginx generates a rewritten URL with new path/query
3. **302 Redirect**: Victim is redirected to the rewritten URL
4. **Reverse Mapping**: Incoming requests to rewritten URLs are mapped back to original URLs
5. **Proxy Forwarding**: Request is proxied to the original URL

---

## Examples

### Example 1: Complete M365 Phishlet with Pro 4.2 Features

```yaml
name: 'm365-pro'
author: 'Your Name'
min_ver: '3.0.0'

# ... proxy_hosts, auth_tokens, credentials ...

# JavaScript injection with location control
js_inject:
  - trigger_domains: ['login.microsoftonline.com']
    trigger_paths: ['/common/oauth2/authorize']
    location: 'head'
    script: |
      console.log('Session: {session_id}');
  
  - trigger_domains: ['login.microsoftonline.com']
    trigger_paths: ['/common/oauth2/authorize']
    location: 'body_bottom'
    script: |
      document.querySelector('form').addEventListener('submit', function() {
        console.log('Login attempt');
      });

# URL rewriting for anti-detection
rewrite_urls:
  - trigger:
      domains: ['login.microsoftonline.com']
      paths: ['/common/oauth2/authorize']
    rewrite:
      path: '/secure/verify'
      query:
        - key: 'sid'
          value: 'session-{id}'
      exclude_keys: ['client_id']
```

### Example 2: Configuration Workflow

```bash
# 1. Set up Pro 4.2 features
config debug on
config js_obfuscation medium
config enc_key my-secret-key-2024

# 2. Enable phishlet
phishlets hostname m365-pro phish.example.com
phishlets enable m365-pro

# 3. Create lure with encrypted parameters
lures create m365-pro https://portal.office.com
lures get-url 0 email=victim@company.com campaign=q1

# 4. Monitor with debug logging
# Watch logs for detailed proxy operations
```

### Example 3: Testing JS Injection Locations

Create a test phishlet to verify injection locations:

```yaml
js_inject:
  # Test head injection
  - trigger_domains: ['test.example.com']
    trigger_paths: ['/']
    location: 'head'
    script: |
      console.log('HEAD: Loaded first');
  
  # Test body_top injection
  - trigger_domains: ['test.example.com']
    trigger_paths: ['/']
    location: 'body_top'
    script: |
      console.log('BODY_TOP: Before content');
  
  # Test body_bottom injection
  - trigger_domains: ['test.example.com']
    trigger_paths: ['/']
    location: 'body_bottom'
    script: |
      console.log('BODY_BOTTOM: After content');
```

Open browser console and verify the order:
```
HEAD: Loaded first
BODY_TOP: Before content
BODY_BOTTOM: After content
```

---

## Troubleshooting

### JavaScript Not Injecting

1. Enable debug mode: `config debug on`
2. Check logs for injection messages
3. Verify HTML structure exists (`<head>`, `<body>` tags)
4. Try different injection location

### Encrypted Parameters Not Working

1. Verify encryption key is set: `config`
2. Check that key is masked (shows `my-s****`)
3. Test with base64 fallback (clear encryption key)
4. Enable debug mode to see decryption logs

### URL Rewriting Not Triggering

1. Verify phishlet `rewrite_urls` configuration
2. Check trigger domains match exactly
3. Test trigger path regex patterns
4. Enable debug mode to see rewrite detection

---

## Best Practices

### Security

- ✅ Use strong encryption keys (16+ characters, mixed case, numbers, symbols)
- ✅ Rotate encryption keys periodically
- ✅ Use `medium` or `high` JS obfuscation for production
- ✅ Test URL rewriting before deploying to victims

### Performance

- ⚠️ `ultra` obfuscation may slow page loads
- ⚠️ Too many JS injections can impact performance
- ✅ Use `body_bottom` for non-critical scripts
- ✅ Minimize injected script size

### Compatibility

- ✅ Test phishlets with encryption enabled and disabled
- ✅ Verify CSP compatibility with nonce preservation
- ✅ Test URL rewriting with actual Safe Browsing
- ✅ Keep backward compatibility in mind

---

## Feature Comparison

| Feature | Standard Evilginx | Pro 4.2 |
|---------|-------------------|---------|
| JS Injection | Regex-based, body only | HTML parser, 3 locations |
| Lure Parameters | Base64 encoding | AES-256 encryption |
| URL Rewriting | Not available | Dynamic path/query rewriting |
| Debug Mode | Limited logging | Verbose debug logs |
| JS Obfuscation | Not available | 5 levels (off to ultra) |

---

## Support

For issues or questions:
1. Enable debug mode: `config debug on`
2. Check logs for error messages
3. Verify phishlet configuration syntax
4. Test with minimal configuration first

---

**Version:** Pro 4.2  
**Last Updated:** 2026-02-03
