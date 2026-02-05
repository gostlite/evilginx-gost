#!/bin/bash
# setup-evilginx-localhost.sh

set -e

echo "=== Setting up Evilginx with fake.localhost domains ==="

# Define your phishing target brands
BRANDS=("xfinity" "google" "microsoft" "apple" "facebook")

# Backup hosts file
BACKUP_FILE="/etc/hosts.backup.$(date +%Y%m%d_%H%M%S)"
sudo cp /etc/hosts "$BACKUP_FILE"
echo "Backup created: $BACKUP_FILE"

# Add entries for each brand
for brand in "${BRANDS[@]}"; do
    echo -e "\n# Evilginx: $brand" | sudo tee -a /etc/hosts > /dev/null
    
    # Common subdomains for each brand
    case $brand in
        "xfinity")
            SUBDOMAINS=("login" "account" "auth" "signin" "mail" "secure" "my" "oauth" "idp" "login.xfinity.com" "idp.xfinity.com" "oauth.xfinity.com")
            ;;
        "google")
            SUBDOMAINS=("accounts" "drive" "mail" "docs" "photos" "myaccount")
            ;;
        "microsoft")
            SUBDOMAINS=("login" "account" "outlook" "office" "onedrive")
            ;;
        *)
            SUBDOMAINS=("login" "account" "auth" "secure" "www")
            ;;
    esac
    
    for sub in "${SUBDOMAINS[@]}"; do
        DOMAIN="$sub.$brand.fake.localhost"
        if ! grep -q "$DOMAIN" /etc/hosts; then
            echo "127.0.0.1    $DOMAIN" | sudo tee -a /etc/hosts
            echo "::1          $DOMAIN" | sudo tee -a /etc/hosts
            echo "Added: $DOMAIN"
        fi
    done
    
    # Wildcard entry (catch-all)
    WILDCARD="*.$brand.fake.localhost"
    if ! grep -q "\\*\.$brand\.fake\.localhost" /etc/hosts; then
        echo "127.0.0.1    $WILDCARD" | sudo tee -a /etc/hosts
        echo "Added wildcard: $WILDCARD"
    fi
done

# Flush DNS
echo -e "\n=== Flushing DNS Cache ==="
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder

# Generate SSL certificates if needed
echo -e "\n=== SSL Certificate Setup ==="
read -p "Generate SSL certificates for *.fake.localhost? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    mkdir -p ssl
    cat > ssl/cert.conf << 'EOF'
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_req

[dn]
C = US
ST = California
L = San Francisco
O = Evilginx Development
CN = *.fake.localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment, keyCertSign, cRLSign
extendedKeyUsage = serverAuth
basicConstraints = critical, CA:TRUE
subjectAltName = @alt_names

[alt_names]
DNS.1 = *.fake.localhost
DNS.2 = fake.localhost
DNS.3 = *.xfinity.fake.localhost
DNS.4 = xfinity.fake.localhost
DNS.5 = *.google.fake.localhost
DNS.6 = google.fake.localhost
EOF

    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout ssl/fake.localhost.key \
        -out ssl/fake.localhost.crt \
        -config ssl/cert.conf
    
    # Trust certificate on macOS
    echo "Adding certificate to macOS keychain..."
    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ssl/fake.localhost.crt
    
    echo "SSL certificates generated in ssl/ folder"
    echo "Key: ssl/fake.localhost.key"
    echo "Cert: ssl/fake.localhost.crt"
fi

# Test setup
echo -e "\n=== Testing Setup ==="
echo "Testing domains:"
TEST_DOMAINS=(
    "login.xfinity.fake.localhost"
    "account.xfinity.fake.localhost"
    "accounts.google.fake.localhost"
    "login.microsoft.fake.localhost"
    "idp.xfinity.fake.localhost"
    "oauth.xfinity.fake.localhost"
)

for domain in "${TEST_DOMAINS[@]}"; do
    echo -n "Testing $domain... "
    if ping -c 1 -t 1 "$domain" &>/dev/null; then
        echo "✓ OK"
    else
        echo "✗ FAILED"
    fi
done

# Evilginx configuration helper
echo -e "\n=== Evilginx Configuration ==="
cat > evilginx-config.md << 'EOF'
# Evilginx Configuration for fake.localhost

## Hostnames to use in Evilginx:
- login.xfinity.fake.localhost
- account.xfinity.fake.localhost
- secure.xfinity.fake.localhost

## SSL Configuration (if using HTTPS):