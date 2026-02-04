# Pro 4.2 Features - Manual Verification Guide

This guide provides step-by-step procedures to manually verify all implemented Pro 4.2 features.

## Prerequisites

Build the project:
```bash
cd "c:\Users\LENOVO\app dev\evilginx2"
go build -mod=mod -o evilginx.exe main.go
```

## Test 1: Configuration Persistence

### Objective
Verify that all configuration options persist correctly.

### Steps
1. Start Evilginx:
   ```bash
   .\evilginx.exe
   ```

2. Test Debug Mode:
   ```
   config debug on
   config
   # Verify: debug = true
   
   config debug off
   config
   # Verify: debug = false
   ```

3. Test JS Obfuscation Levels:
   ```
   config js_obfuscation off
   config
   # Verify: js_obfuscation = off
   
   config js_obfuscation low
   config
   # Verify: js_obfuscation = low
   
   config js_obfuscation medium
   config
   # Verify: js_obfuscation = medium
   
   config js_obfuscation high
   config
   # Verify: js_obfuscation = high
   
   config js_obfuscation ultra
   config
   # Verify: js_obfuscation = ultra
   ```

4. Test Encryption Key:
   ```
   config enc_key my-secret-key-2024
   config
   # Verify: enc_key = my-s**** (masked)
   
   config enc_key ""
   config
   # Verify: enc_key = (empty)
   ```

### Expected Results
✅ All config values should persist  
✅ Encryption key should be masked when displayed  
✅ Config should survive restart (if saved to file)

---

## Test 2: Encrypted Lure Parameters

### Objective
Verify that lure parameters are encrypted when encryption key is set.

### Steps
1. Set encryption key:
   ```
   config enc_key test-encryption-key-123
   ```

2. Enable a phishlet (example with m365):
   ```
   phishlets hostname m365 phish.example.com
   phishlets enable m365
   ```

3. Create a lure:
   ```
   lures create m365 https://portal.office.com
   ```

4. Generate URL with custom parameters:
   ```
   lures get-url 0 email=victim@company.com campaign=q1-2024
   ```

5. Examine the generated URL:
   - Look for the `yui` parameter
   - Value should be encrypted (not readable base64)
   - Example: `?yui=Ag3kF9mP2x...` (encrypted)
   - NOT: `?yui=eyJlbWFpbCI6...` (base64)

6. Test without encryption key:
   ```
   config enc_key ""
   lures get-url 0 email=test@example.com
   ```
   - Value should be base64 encoded (readable if decoded)

### Expected Results
✅ With encryption key: Parameters are encrypted  
✅ Without encryption key: Parameters use base64 encoding  
✅ Both methods should work when victim visits the URL

---

## Test 3: JavaScript Injection with Location Control

### Objective
Verify that JavaScript can be injected at different locations (head, body_top, body_bottom).

### Prerequisites
Create a test phishlet with JS injection at different locations.

### Test Phishlet (test-js.yaml)
```yaml
name: 'test-js'
author: 'Test'
min_ver: '3.0.0'

proxy_hosts:
  - {phish_sub: 'login', orig_sub: 'login', domain: 'example.com', session: true, is_landing: true}

sub_filters:
  - {triggers_on: 'example.com', orig_sub: 'login', domain: 'example.com', search: 'example.com', replace: '{hostname}', mimes: ['text/html', 'application/json']}

js_inject:
  - trigger_domains: ['login.example.com']
    trigger_paths: ['/']
    location: 'head'
    script: |
      console.log('HEAD: Injected at head');
  
  - trigger_domains: ['login.example.com']
    trigger_paths: ['/']
    location: 'body_top'
    script: |
      console.log('BODY_TOP: Injected at body top');
  
  - trigger_domains: ['login.example.com']
    trigger_paths: ['/']
    location: 'body_bottom'
    script: |
      console.log('BODY_BOTTOM: Injected at body bottom');
```

### Steps
1. Place test phishlet in `phishlets/` directory

2. Enable the phishlet:
   ```
   phishlets hostname test-js test.phish.com
   phishlets enable test-js
   ```

3. Create a lure:
   ```
   lures create test-js https://example.com
   lures get-url 0
   ```

4. Visit the lure URL in a browser

5. Open browser Developer Tools (F12) → Console

6. Check console output:
   ```
   HEAD: Injected at head
   BODY_TOP: Injected at body top
   BODY_BOTTOM: Injected at body bottom
   ```

7. View page source (Ctrl+U) and verify script locations:
   - First script should be in `<head>` section
   - Second script should be at beginning of `<body>`
   - Third script should be at end of `<body>`

### Expected Results
✅ Scripts appear in console in correct order  
✅ Scripts are injected at specified locations in HTML  
✅ Page loads without errors

---

## Test 4: JS Obfuscation Levels

### Objective
Verify that JavaScript obfuscation works at different levels.

### Steps
1. Set obfuscation to 'off':
   ```
   config js_obfuscation off
   ```

2. Visit a page with JS injection

3. View page source and check injected script:
   - Should be readable, unobfuscated

4. Set obfuscation to 'medium':
   ```
   config js_obfuscation medium
   ```

5. Refresh the page and view source:
   - Script should be minified/transformed

6. Test with 'high' and 'ultra':
   ```
   config js_obfuscation high
   config js_obfuscation ultra
   ```

### Expected Results
✅ 'off': Script is readable  
✅ 'low': Basic minification  
✅ 'medium': Minification + encoding  
✅ 'high': Heavy obfuscation  
✅ 'ultra': Maximum obfuscation  
✅ Scripts still execute correctly at all levels

---

## Test 5: Debug Mode Logging

### Objective
Verify that debug mode provides verbose logging.

### Steps
1. Enable debug mode:
   ```
   config debug on
   ```

2. Perform various actions:
   - Create a lure
   - Visit a lure URL
   - Trigger JS injection

3. Check console output for detailed logs:
   - Should see "js_inject: injected script at location..."
   - Should see "js_inject: HTML parser failed..." (if applicable)
   - Should see obfuscation level logs

4. Disable debug mode:
   ```
   config debug off
   ```

5. Repeat actions:
   - Logs should be less verbose

### Expected Results
✅ Debug mode shows detailed logs  
✅ Normal mode shows standard logs  
✅ No errors or crashes

---

## Test 6: HTML Parser Fallback

### Objective
Verify that HTML parser gracefully falls back to regex if parsing fails.

### Steps
1. Enable debug mode:
   ```
   config debug on
   ```

2. Create a phishlet that returns non-HTML content (e.g., JSON API)

3. Trigger JS injection on non-HTML content

4. Check logs for:
   ```
   js_inject: content is not HTML, skipping injection
   ```
   OR
   ```
   js_inject: HTML parser failed (...), falling back to regex
   ```

### Expected Results
✅ Parser detects non-HTML content  
✅ Fallback to regex works  
✅ No crashes or errors

---

## Test 7: CSP Nonce Preservation

### Objective
Verify that Content Security Policy nonce attributes are preserved.

### Prerequisites
A target site with CSP that uses nonce attributes.

### Steps
1. Visit a page with CSP nonce in existing scripts

2. View page source and find existing script with nonce:
   ```html
   <script nonce="abc123xyz">...</script>
   ```

3. Check injected script:
   ```html
   <script nonce="abc123xyz">console.log('injected');</script>
   ```

4. Verify the nonce matches

### Expected Results
✅ Injected scripts inherit nonce from existing scripts  
✅ No CSP violations in browser console  
✅ Scripts execute successfully

---

## Test 8: Build Verification

### Objective
Verify that the project builds successfully with all changes.

### Steps
1. Clean build:
   ```bash
   go clean
   go build -mod=mod -o evilginx.exe main.go
   ```

2. Check for compilation errors

3. Run the executable:
   ```bash
   .\evilginx.exe
   ```

4. Verify it starts without errors

### Expected Results
✅ Build completes successfully  
✅ No compilation errors  
✅ Executable runs without crashes  
✅ All commands are available

---

## Test 9: Configuration File Persistence

### Objective
Verify that configuration persists across restarts.

### Steps
1. Set all config options:
   ```
   config debug on
   config js_obfuscation high
   config enc_key my-persistent-key
   ```

2. Exit Evilginx

3. Restart Evilginx

4. Check config:
   ```
   config
   ```

5. Verify all values persisted

### Expected Results
✅ Debug mode persists  
✅ JS obfuscation level persists  
✅ Encryption key persists (masked)

---

## Test 10: URL Rewriting Structures

### Objective
Verify that URL rewriting structures load correctly from phishlets.

### Test Phishlet
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
      exclude_keys: ['debug']
```

### Steps
1. Add `rewrite_urls` section to a phishlet

2. Load the phishlet:
   ```
   phishlets hostname test test.phish.com
   phishlets enable test
   ```

3. Check for errors in console

4. If no errors, URL rewrite structures loaded successfully

### Expected Results
✅ Phishlet loads without errors  
✅ No validation errors for `rewrite_urls`  
✅ `{id}` placeholder is validated

---

## Test 11: Custom Lure Hostnames
### Objective
Verify that custom lure hostnames work correctly.

### Steps
1. Configure a phishlet (e.g. `test-phishlet`) on `example.com`:
   ```bash
   phishlets hostname test-phishlet auth.example.com
   ```

2. Create a lure with a custom subdomain:
   ```bash
   lures create test-phishlet /login --hostname login.example.com
   ```
   *Should success: `login.example.com` has same base as `auth.example.com`*

3. Create a lure with TLD-only hostname:
   ```bash
   lures create test-phishlet /verify --hostname example.com
   ```
   *Should success*

4. Attempt to create invalid lure:
   ```bash
   lures create test-phishlet /fail --hostname google.com
   ```
   *Should fail: Different base domain*

5. Get generated URLs:
   ```bash
   lures get-url 0
   ```
   *Verify URL matches: `https://login.example.com/login`*

### Expected Results
✅ Valid custom hostnames are accepted
✅ Invalid hostnames are rejected
✅ Generated URLs reflect the custom hostname

---

## Test 12: URL Rewriting (Functional)
### Objective
Verify that URL rewriting logic correctly modifies requests and responses.

### Test Phishlet Configuration
Add this to your test phishlet yaml:
```yaml
rewrite_urls:
  - trigger:
      domains: ['auth.example.com']
      paths: ['^/secure/verify$']
    rewrite:
      path: '/auth/login'
      query:
         - key: 'token'
           value: '123'
```

### Steps
1. Enable phishlet on `auth.example.com`.
2. Visit `https://auth.example.com/secure/verify`
3. Check `debug` logs.
   *   Should see: `rewrite_url: matched trigger...`
   *   Should see: `rewrite_url: rewriting /secure/verify -> /auth/login`
4. The server receives request for `/auth/login`.
5. If server redirects back to `/auth/login`, verifying that the proxy rewrites `Location` header back to `/secure/verify`.

### Expected Results
✅ Request to `/secure/verify` is internally processed as `/auth/login`
✅ User stays on `/secure/verify` URL in browser
✅ Redirects are mapped back correctly

---

## Summary Checklist

After completing all tests, verify:

- [ ] All configuration options work
- [ ] Encryption key is set and masked
- [ ] Lure parameters are encrypted
- [ ] JS injection works at all locations
- [ ] JS obfuscation works at all levels
- [ ] Debug mode provides verbose logging
- [ ] HTML parser fallback works
- [ ] CSP nonce is preserved
- [ ] Build completes successfully
- [ ] Configuration persists across restarts
- [ ] URL rewriting structures load correctly
- [ ] Custom Lure Hostnames work
- [ ] URL Rewriting functional logic works

---

**Testing Date:** _____________  
**Tester:** _____________  
**Version:** Pro 4.2  
**Status:** _____________
