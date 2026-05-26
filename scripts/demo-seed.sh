#!/usr/bin/env bash
# Seed the demo SOP documents (PAY/CART/AD) AND a couple of demo runbooks
# into a running SigNoz instance. Wraps runbook step 3
# (docs/demo/2026-05-21-runbook.md) and acts as the prefill for the
# Runbook Management v0.1 UI demo so the page is not empty on first open.
#
# Auth: logs in with email+password+orgUUID via /api/v2/sessions/email_password
# and uses the returned access token as Bearer for all writes. The legacy
# X-SigNoz-Org-Id header is dropped — org is carried in the JWT claims.
#
# Usage:
#   ./scripts/demo-seed.sh
#
#   SIGNOZ_URL=http://host:8080 \
#   SIGNOZ_EMAIL=admin@example.local \
#   SIGNOZ_PASSWORD=signoz123 \
#   SIGNOZ_ORG_UUID=019dc701-88c0-7d38-a10a-8da523562e50 \
#     ./scripts/demo-seed.sh
#
# Exit codes:
#   0 = all SOPs + runbooks seeded and verified
#   1 = curl/post failure or fixture missing
#   2 = verification failed (sopId or runbook not found after seed)
#   3 = signoz not reachable
#   4 = login failed (bad email/password/orgUUID)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SIGNOZ_URL="${SIGNOZ_URL:-http://localhost:8080}"
SIGNOZ_EMAIL="${SIGNOZ_EMAIL:-admin@example.local}"
SIGNOZ_PASSWORD="${SIGNOZ_PASSWORD:-signoz123}"
SIGNOZ_ORG_UUID="${SIGNOZ_ORG_UUID:-019dc701-88c0-7d38-a10a-8da523562e50}"

SOP_FILES=(
  "docs/demo/sop_pay.json"
  "docs/demo/sop_cart.json"
  "docs/demo/sop_ad.json"
)
EXPECTED_IDS=("SOP-PAY-001" "SOP-CART-001" "SOP-AD-001")

# Runbook fixtures, one entry per row: "file|sopId|version".
RUNBOOKS=(
  "docs/demo/runbook_pay_restart.json|SOP-PAY-001|2026-05-12.1"
  "docs/demo/runbook_cart_redis.json|SOP-CART-001|2026-05-20.1"
)

echo "==> Target SigNoz: ${SIGNOZ_URL} (user=${SIGNOZ_EMAIL})"

# Preflight: signoz reachable?
if ! curl -fsS -o /dev/null --max-time 5 "${SIGNOZ_URL}/api/v1/health"; then
  echo "ERROR: ${SIGNOZ_URL}/api/v1/health not reachable. Bring the stack up first (runbook Step 2)." >&2
  exit 3
fi

# Login → access token
login_payload=$(printf '{"email":"%s","password":"%s","orgID":"%s"}' \
  "${SIGNOZ_EMAIL}" "${SIGNOZ_PASSWORD}" "${SIGNOZ_ORG_UUID}")
login_response="$(curl -fsS -X POST "${SIGNOZ_URL}/api/v2/sessions/email_password" \
  -H "Content-Type: application/json" \
  -d "${login_payload}" 2>&1)" || {
  echo "ERROR: login failed for ${SIGNOZ_EMAIL}" >&2
  echo "${login_response}" >&2
  exit 4
}
TOKEN="$(printf '%s' "${login_response}" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["accessToken"])')"
if [[ -z "${TOKEN}" ]]; then
  echo "ERROR: login succeeded but accessToken missing in response" >&2
  exit 4
fi
echo "==> Authenticated (token len=${#TOKEN})"

# Seed SOPs
for f in "${SOP_FILES[@]}"; do
  abs="${ROOT_DIR}/${f}"
  if [[ ! -f "${abs}" ]]; then
    echo "ERROR: missing ${abs}" >&2
    exit 1
  fi
  echo "==> POST ${f}"
  curl -fsS -X POST "${SIGNOZ_URL}/api/v2/ds/sop/documents" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${TOKEN}" \
    -d @"${abs}" \
    | sed -e 's/^/    /'
  echo
done

# Seed runbooks (after their parent SOPs exist)
for row in "${RUNBOOKS[@]}"; do
  IFS='|' read -r rb_file rb_sop rb_ver <<<"${row}"
  abs="${ROOT_DIR}/${rb_file}"
  if [[ ! -f "${abs}" ]]; then
    echo "ERROR: missing ${abs}" >&2
    exit 1
  fi
  echo "==> POST ${rb_file} → ${rb_sop} ${rb_ver}"
  curl -fsS -X POST \
    "${SIGNOZ_URL}/api/v2/ds/sop/documents/${rb_sop}/versions/${rb_ver}/runbooks" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${TOKEN}" \
    -d @"${abs}" \
    | sed -e 's/^/    /'
  echo
done

# Verify SOPs
echo "==> Verifying seeded SOPs..."
listing="$(curl -fsS "${SIGNOZ_URL}/api/v2/ds/sop/documents" \
  -H "Authorization: Bearer ${TOKEN}")"

missing=()
for id in "${EXPECTED_IDS[@]}"; do
  if ! grep -q "\"${id}\"" <<<"${listing}"; then
    missing+=("${id}")
  fi
done

if (( ${#missing[@]} > 0 )); then
  echo "ERROR: missing after seed: ${missing[*]}" >&2
  echo "Response was:" >&2
  echo "${listing}" >&2
  exit 2
fi

# Verify runbooks attached to their parent SOPs
echo "==> Verifying seeded runbooks..."
for row in "${RUNBOOKS[@]}"; do
  IFS='|' read -r rb_file rb_sop rb_ver <<<"${row}"
  rb_title="$(grep -o '"title"[[:space:]]*:[[:space:]]*"[^"]*"' "${ROOT_DIR}/${rb_file}" \
              | head -1 | sed -E 's/.*"title"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')"
  rb_listing="$(curl -fsS \
    "${SIGNOZ_URL}/api/v2/ds/sop/documents/${rb_sop}/versions/${rb_ver}/runbooks" \
    -H "Authorization: Bearer ${TOKEN}")"
  if ! grep -qF "\"${rb_title}\"" <<<"${rb_listing}"; then
    echo "ERROR: runbook \"${rb_title}\" not found on ${rb_sop}/${rb_ver}" >&2
    echo "Response was:" >&2
    echo "${rb_listing}" >&2
    exit 2
  fi
  echo "  OK: ${rb_sop}/${rb_ver} ⇢ ${rb_title}"
done

echo "OK: ${EXPECTED_IDS[*]} + ${#RUNBOOKS[@]} runbooks"
