#!/bin/sh

set -e

PORT=50000
CONF="/etc/danted.conf"
WHITELIST_FILE="whitelist.txt"

if [ ! -f "$WHITELIST_FILE" ]; then
  echo "whitelist.txt not found"
  exit 1
fi

echo "Installing dante-server..."
apt update
apt install -y dante-server

INTERFACE=$(ip route | awk '/default/ {print $5}' | head -n1)

TMP_CLIENT=$(mktemp)
TMP_SOCKS=$(mktemp)

while read -r IP; do
  [ -z "$IP" ] && continue

  cat >> "$TMP_CLIENT" <<EOF
client pass {
    from: $IP/32 to: 0.0.0.0/0
}
EOF

  cat >> "$TMP_SOCKS" <<EOF
socks pass {
    from: $IP/32 to: 0.0.0.0/0
    command: connect bind udpassociate
}
EOF

done < "$WHITELIST_FILE"

cat > "$CONF" <<EOF
logoutput: syslog

internal: 0.0.0.0 port = $PORT
external: $INTERFACE

socksmethod: none
clientmethod: none

user.notprivileged: nobody

$(cat $TMP_CLIENT)

client block {
    from: 0.0.0.0/0 to: 0.0.0.0/0
}

$(cat $TMP_SOCKS)

socks block {
    from: 0.0.0.0/0 to: 0.0.0.0/0
}
EOF

rm $TMP_CLIENT $TMP_SOCKS

systemctl restart danted
systemctl enable danted

echo "SOCKS5 proxy started on port $PORT"
