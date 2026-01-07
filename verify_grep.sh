#!/bin/bash
DOMAIN="login.office365-security.com"
echo "Creating dummy log..."
cat > dummy.log << EOF
[inf] unwanted line
https://evilginx.com
https://login.office365-security.com/AbCdEf
EOF

echo "Testing extraction..."
grep -i "https://[^[:space:]]*$DOMAIN[^[:space:]]*" dummy.log
