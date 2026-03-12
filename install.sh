#!/bin/sh

set -e

BASE_PORT=50000
CONF="/etc/danted.conf"
WHITELIST_FILE="whitelist.txt"
REVERSE_FILE="reverse.txt"

if [ ! -f "$WHITELIST_FILE" ]; then
  echo "whitelist.txt not found"
  exit 1
fi

echo "Installing dante-server..."
apt update
apt install -y dante-server curl

echo "Installing gost..."

# remove wrong gost if exists
apt remove -y gost 2>/dev/null || true

# install official gost
curl -L https://github.com/go-gost/gost/releases/download/v3.2.6/gost_3.2.6_linux_amd64.tar.gz \
| tar -xz

mv gost /usr/local/bin/gost
chmod +x /usr/local/bin/gost

# make compatible path
ln -sf /usr/local/bin/gost /usr/bin/gost

INTERFACE=$(ip route | awk '/default/ {print $5}' | head -n1)

TMP_CLIENT=$(mktemp)
TMP_SOCKS=$(mktemp)
TMP_WL=$(mktemp)

cleanup_reverse_services() {
  if ls /etc/systemd/system/reverse-socks-*.service >/dev/null 2>&1; then
    for U in /etc/systemd/system/reverse-socks-*.service; do
      UNIT_NAME=$(basename "$U")
      PORT=$(printf "%s" "$UNIT_NAME" | sed -n 's/^reverse-socks-\([0-9]\+\)\.service$/\1/p')
      [ -n "$PORT" ] || continue

      systemctl disable --now "$UNIT_NAME" 2>/dev/null || true
      rm -f "$U"

      while iptables -S INPUT | awk '/--dport '"$PORT"' / {print}' | grep -q .; do
        iptables -S INPUT | awk '/--dport '"$PORT"' / {print}' | while read -r RULE; do
          iptables $(printf "%s" "$RULE" | sed 's/^-A /-D /')
        done
      done
    done
    systemctl daemon-reload
  fi
}

while read -r IP; do
  IP=$(printf "%s" "$IP" | tr -d ' \t\r')
  [ -z "$IP" ] && continue
  case "$IP" in
    \#*) continue ;;
  esac

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

  printf "%s\n" "$IP" >> "$TMP_WL"

done < "$WHITELIST_FILE"

cat > "$CONF" <<EOF
logoutput: syslog

internal: 0.0.0.0 port = $BASE_PORT
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

rm -f "$TMP_CLIENT" "$TMP_SOCKS"

systemctl restart danted
systemctl enable danted

echo "SOCKS5 proxy started on port $BASE_PORT"

cleanup_reverse_services

if [ -f "$REVERSE_FILE" ]; then
  echo "Setting up reverse forwards..."

  FORWARD_PORT=$((BASE_PORT + 1))
  UNITS=""

  while read -r RPORT; do

    RPORT=$(printf "%s" "$RPORT" | tr -d ' \t\r')
    [ -z "$RPORT" ] && continue
    case "$RPORT" in
      \#*) continue ;;
    esac

    UNIT="reverse-socks-${FORWARD_PORT}.service"

cat > "/etc/systemd/system/${UNIT}" <<EOF
[Unit]
Description=Reverse SOCKS forward ${FORWARD_PORT} -> 127.0.0.1:${RPORT}
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/gost -L tcp://0.0.0.0:${FORWARD_PORT} -F tcp://127.0.0.1:${RPORT}
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF

    while read -r WIP; do
      [ -z "$WIP" ] && continue
      iptables -C INPUT -p tcp --dport "$FORWARD_PORT" -s "$WIP" -j ACCEPT 2>/dev/null || \
      iptables -A INPUT -p tcp --dport "$FORWARD_PORT" -s "$WIP" -j ACCEPT
    done < "$TMP_WL"

    iptables -C INPUT -p tcp --dport "$FORWARD_PORT" -j DROP 2>/dev/null || \
    iptables -A INPUT -p tcp --dport "$FORWARD_PORT" -j DROP

    UNITS="$UNITS $UNIT"
    FORWARD_PORT=$((FORWARD_PORT + 1))

  done < "$REVERSE_FILE"

  systemctl daemon-reload

  for U in $UNITS; do
    systemctl enable --now "$U"
  done

else
  echo "$REVERSE_FILE not found"
fi

rm -f "$TMP_WL"
