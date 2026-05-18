#!/usr/bin/env bash
set -euo pipefail

TRANSPORT_URL="${TRANSPORT_URL:-http://localhost:8081}"
AGENT_URL="${AGENT_URL:-http://localhost:8082}"
APP_URL="${APP_URL:-http://localhost:8080}"

echo "== health =="
curl -sf "$TRANSPORT_URL/health"; echo
curl -sf "$AGENT_URL/health"; echo
curl -sf "$APP_URL/health" 2>/dev/null && echo || echo "(app optional, not running on $APP_URL)"

SCHEMA='{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string","format":"email"}}}'
SENT_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "== send =="
SEND_RESP=$(curl -sf -X POST "$TRANSPORT_URL/v1/messages/send" \
  -H 'Content-Type: application/json' \
  -d "{\"sender\":\"smoke\",\"sent_at\":\"$SENT_AT\",\"json_schema\":$SCHEMA,\"sample_count\":2}")
echo "$SEND_RESP"
MSG_ID=$(echo "$SEND_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['message_id'])")

echo "== wait receive (up to 30s) =="
for i in $(seq 1 10); do
  RECV=$(curl -sf -X POST "$TRANSPORT_URL/v1/messages/receive" \
    -H 'Content-Type: application/json' \
    -d '{"sender":"smoke","wait_ms":3000}')
  if echo "$RECV" | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if d.get('sender') and d.get('payload') else 1)"; then
    echo "$RECV" | python3 -m json.tool | head -30
    exit 0
  fi
  echo "attempt $i: empty, retry..."
done
echo "FAILED: no payload received"
exit 1
