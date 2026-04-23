#!/usr/bin/env bash
#
# generate-traffic.sh — Generate synthetic traffic against the FlowDesk API
# to populate Grafana dashboards with data.
#
# Usage:
#   ./scripts/generate-traffic.sh [BASE_URL] [DURATION_SECONDS]
#
# Examples:
#   ./scripts/generate-traffic.sh http://37.230.168.54 120
#   ./scripts/generate-traffic.sh http://localhost:8080 60
#
set -euo pipefail

BASE="${1:-http://37.230.168.54}"
DURATION="${2:-120}"
API="${BASE}/v1"

echo "=== FlowDesk Traffic Generator ==="
echo "  Target:   ${BASE}"
echo "  Duration: ${DURATION}s"
echo ""

# -------------------------------------------------------
# Helper functions
# -------------------------------------------------------
register_user() {
  local email="user-${RANDOM}@test.flowdesk.dev"
  local pass="TestPass123!"
  local name="Test User ${RANDOM}"

  local resp
  resp=$(curl -s -w "\n%{http_code}" -X POST "${API}/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${email}\",\"password\":\"${pass}\",\"full_name\":\"${name}\"}" 2>/dev/null) || true

  local code
  code=$(echo "$resp" | tail -1)
  local body
  body=$(echo "$resp" | sed '$d')

  if [[ "$code" == "201" ]] || [[ "$code" == "200" ]]; then
    echo "$email|$pass|$body"
  else
    echo ""
  fi
}

login_user() {
  local email="$1"
  local pass="$2"

  local resp
  resp=$(curl -s -w "\n%{http_code}" -X POST "${API}/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${email}\",\"password\":\"${pass}\"}" 2>/dev/null) || true

  local code
  code=$(echo "$resp" | tail -1)
  local body
  body=$(echo "$resp" | sed '$d')

  if [[ "$code" == "200" ]]; then
    # Extract access token
    echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo ""
  else
    echo ""
  fi
}

# -------------------------------------------------------
# Phase 1: Register users and get tokens
# -------------------------------------------------------
echo "--- Phase 1: Registering test users ---"
TOKENS=()
for i in $(seq 1 5); do
  result=$(register_user)
  if [[ -n "$result" ]]; then
    email=$(echo "$result" | cut -d'|' -f1)
    pass=$(echo "$result" | cut -d'|' -f2)
    token=$(login_user "$email" "$pass")
    if [[ -n "$token" ]]; then
      TOKENS+=("$token")
      echo "  ✓ Registered & logged in: ${email}"
    else
      echo "  ✗ Registered but login failed: ${email}"
    fi
  else
    echo "  ✗ Registration failed (user $i)"
  fi
done

echo ""
echo "  Active tokens: ${#TOKENS[@]}"

# Also try some failed logins to generate auth metrics
echo ""
echo "--- Phase 1b: Generating failed login attempts ---"
for i in $(seq 1 10); do
  curl -s -o /dev/null -X POST "${API}/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"nonexistent@test.dev","password":"wrong"}' 2>/dev/null || true
done
echo "  ✓ 10 failed login attempts"

# -------------------------------------------------------
# Phase 2: Hit various API endpoints
# -------------------------------------------------------
echo ""
echo "--- Phase 2: Generating API traffic (${DURATION}s) ---"

END_TIME=$((SECONDS + DURATION))
REQUEST_COUNT=0

get_random_token() {
  if [[ ${#TOKENS[@]} -gt 0 ]]; then
    echo "${TOKENS[$((RANDOM % ${#TOKENS[@]}))]}"
  else
    echo ""
  fi
}

while [[ $SECONDS -lt $END_TIME ]]; do
  # Pick a random endpoint pattern
  ENDPOINT=$((RANDOM % 10))
  TOKEN=$(get_random_token)
  AUTH_HEADER=""
  if [[ -n "$TOKEN" ]]; then
    AUTH_HEADER="-H \"Authorization: Bearer ${TOKEN}\""
  fi

  case $ENDPOINT in
    0|1)
      # GET /v1/floors — list floors
      eval curl -s -o /dev/null ${AUTH_HEADER} "${API}/floors" 2>/dev/null || true
      ;;
    2|3)
      # GET /v1/desks — list desks (may need floor_id param)
      eval curl -s -o /dev/null ${AUTH_HEADER} "${API}/desks" 2>/dev/null || true
      ;;
    4)
      # GET /v1/reservations — list reservations
      eval curl -s -o /dev/null ${AUTH_HEADER} "${API}/reservations" 2>/dev/null || true
      ;;
    5)
      # GET /v1/analytics/summary — analytics
      eval curl -s -o /dev/null ${AUTH_HEADER} "${API}/analytics/summary" 2>/dev/null || true
      ;;
    6)
      # GET /healthz — health check
      curl -s -o /dev/null "${BASE}/healthz" 2>/dev/null || true
      ;;
    7)
      # GET /metrics — prometheus metrics
      curl -s -o /dev/null "${BASE}/metrics" 2>/dev/null || true
      ;;
    8)
      # GET / — frontend (SPA)
      curl -s -o /dev/null "${BASE}/" 2>/dev/null || true
      ;;
    9)
      # GET /v1/nonexistent — 404 to generate error metrics
      eval curl -s -o /dev/null ${AUTH_HEADER} "${API}/nonexistent" 2>/dev/null || true
      ;;
  esac

  REQUEST_COUNT=$((REQUEST_COUNT + 1))

  # Print progress every 50 requests
  if [[ $((REQUEST_COUNT % 50)) -eq 0 ]]; then
    REMAINING=$((END_TIME - SECONDS))
    if [[ $REMAINING -lt 0 ]]; then REMAINING=0; fi
    echo "  Sent ${REQUEST_COUNT} requests (${REMAINING}s remaining)"
  fi

  # Small random delay (50-200ms)
  sleep "0.$(printf '%03d' $((RANDOM % 150 + 50)))"
done

echo ""
echo "============================================"
echo "  Traffic generation complete!"
echo "  Total requests: ${REQUEST_COUNT}"
echo "  Users created:  ${#TOKENS[@]}"
echo "============================================"
echo ""
echo "Check your Grafana dashboard at: ${BASE}/grafana"
echo ""
