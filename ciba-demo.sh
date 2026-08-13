#!/usr/bin/env bash

set -euo pipefail

BASE="${BASE:-http://localhost:5173/realms/default}"
CLIENT_ID="${CLIENT_ID:-00000000-0000-4000-8000-000000000021}"
CLIENT_SECRET="${DEMO_CLIENT_SECRET:-demo-client-secret}"
LOGIN_HINT="${DEMO_USERNAME:-alice}"

token_request() {
  curl -sS -u "$CLIENT_ID:$CLIENT_SECRET" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -X POST "$BASE/token" "$@"
}

echo "CIBA 承認要求を起票します..."
CIBA_RESPONSE=$(curl -sS -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -X POST "$BASE/bc-authorize" \
  --data-urlencode "scope=openid profile" \
  --data-urlencode "login_hint=$LOGIN_HINT" \
  --data-urlencode "binding_message=demo-$(date +%H%M%S)" \
  --data-urlencode 'authorization_details=[{"type":"payment_initiation","actions":["initiate"],"fields":{"creditorAccount":"DEMO-001","instructedAmount":1000}}]')
printf '%s\n' "$CIBA_RESPONSE" | jq .

AUTH_REQ_ID=$(printf '%s' "$CIBA_RESPONSE" | jq -er '.auth_req_id')
INTERVAL=$(printf '%s' "$CIBA_RESPONSE" | jq -er '.interval')

echo
echo "承認前に一度 poll します（authorization_pending が正常です）:"
token_request \
  --data-urlencode "grant_type=urn:openid:params:grant-type:ciba" \
  --data-urlencode "auth_req_id=$AUTH_REQ_ID" | jq .

echo
echo "ブラウザで次を開き、alice として要求を承認してください:"
echo "$BASE/account/approvals"
echo "step-up を求められた場合の password: demo-password-1234"
read -r -p "承認後、Enter を押してください... "

echo "polling interval (${INTERVAL}s) を満たすまで待機します..."
sleep "$((INTERVAL + 1))"

echo
echo "承認後の token response:"
token_request \
  --data-urlencode "grant_type=urn:openid:params:grant-type:ciba" \
  --data-urlencode "auth_req_id=$AUTH_REQ_ID" | jq .
