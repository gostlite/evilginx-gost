# Evilginx Pro 4.2 - Feature Usage Guide

This guide explains how to use the new advanced features introduced in Evilginx Pro 4.2.

## 1. URL Rewriting (Anti-Detection)

URL Rewriting allows you to hide typical phishing paths (like `/s/session_id`) and standard phishlet paths (like `/login/authorized`) behind custom, legitimate-looking URLs.

### Configuration
Add the `rewrite_urls` section to your phishlet `.yaml` file:

```yaml
rewrite_urls:
  - trigger:
      domains: ['login.example.com']  # The domain the victim visits
      paths: ['^/common/oauth2/authorize$'] # The "fake" path user sees
    rewrite:
      path: '/ppsecure/post.srf'      # The "real" path the phishlet uses
      query:
        - key: 'client_id'
          value: '{client_id}'        # Persist parameters
```

### How to Use
1.  Enable the phishlet with rewriting rules.
2.  The rewriting matches the *incoming* request from the victim.
3.  If the victim visits `https://phish.com/common/oauth2/authorize`, Evilginx will internally process it as `/ppsecure/post.srf`.
4.  If the backend application redirects the user to `/ppsecure/post.srf`, the proxy will catch it and rewrite the *Location* header back to `/common/oauth2/authorize`, keeping the user on the "safe" path.

---

## 2. Advanced JavaScript Injection

Pro 4.2 replaces the old regex-based injection with a robust HTML parser and adds location control.

### Configuration
Update the `js_inject` section in your phishlet:

```yaml
js_inject:
  - trigger_domains: ["login.example.com"]
    trigger_paths: ["/login"]
    
    # NEW: Specify injection location
    # Options: 'head', 'body_top', 'body_bottom' (default)
    location: "head" 
    
    script: |
      // Application logic here
      console.log('Injected in HEAD');
```

*   **head**: Injected just before `</head>`. Useful for anti-detection scripts, polyfills, or CSS.
*   **body_top**: Injected just after `<body>`. Useful for overlays or blocking UI.
*   **body_bottom**: Injected just before `</body>`. Standard location for logic scripts.

---

## 3. Lure Enhancements

### Custom Hostnames (Subdomains)
You can now create lures that use specific subdomains, or even the top-level domain (if configured), instead of the default randomization.

**Command:**
```bash
lures create <phishlet> <path> --hostname <custom.hostname>
```

**Examples:**
*   **Specific Subdomain:** `lures create m365 /login --hostname auth.phish.com`
*   **TLD Only:** `lures create m365 /login --hostname phish.com`

*Note: The custom hostname must be a valid subdomain of the configured phishlet domain.*

### AES-256 Parameter Encryption
All custom parameters in proper lure URLs are now encrypted using AES-256 if an encryption key is configured.

**Setup:**
```bash
config enc_key my-secret-password-123
```

**Usage:**
```bash
lures get-url <id> email=victim@company.com
```
The resulting URL will contain a single `?yui=...` parameter containing the encrypted data, instead of base64 encoded values.

---

## 4. Configuration Persistence & Debugging

New configuration commands are available:

*   **Debug Mode:** Toggle verbose logging.
    ```bash
    config debug on/off
    ```
*   **JS Obfuscation:** Control the level of obfuscation for injected scripts.
    ```bash
    config js_obfuscation off/low/medium/high/ultra
    ```

---
