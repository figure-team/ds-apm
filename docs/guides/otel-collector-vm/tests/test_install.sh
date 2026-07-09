#!/usr/bin/env bash
# 의존성 없는 단위/통합 테스트. Git Bash(개발기)에서 바로 실행.
# install.sh를 source하면 가드 덕분에 main은 안 돈다.
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$HERE/../install.sh"

# 실패는 변수가 아니라 파일에 기록한다. assert가 상태격리용 서브셸 `( )` 안에서
# 돌아도 실패가 부모로 전파되게 하려는 것 — 셸 변수는 서브셸 경계를 못 넘지만
# 파일 쓰기는 넘는다. (이전 설계: 서브셸 안 `fail_mark`이 삼켜져 오답이 ALL PASS로 통과)
FAILFILE="$(mktemp)"
fail_mark() { echo x >>"$FAILFILE"; }
assert_eq() { # $1=expected $2=actual $3=msg
  if [[ "$1" == "$2" ]]; then echo "ok: $3"; else
    echo "FAIL: $3"; echo "  expected: [$1]"; echo "  actual:   [$2]"; fail_mark; fi
}
assert_contains() { # $1=haystack $2=needle $3=msg
  if [[ "$1" == *"$2"* ]]; then echo "ok: $3"; else
    echo "FAIL: $3"; echo "  missing: [$2]"; echo "  in: [$1]"; fail_mark; fi
}
assert_not_contains() { # $1=haystack $2=needle $3=msg
  if [[ "$1" != *"$2"* ]]; then echo "ok: $3"; else
    echo "FAIL: $3 (unexpectedly present: [$2])"; fail_mark; fi
}

# shellcheck disable=SC1090
source "$SCRIPT"

# ── parse_env_file ─────────────────────────────────────
test_parse_env_file() {
  local tmp; tmp="$(mktemp)"
  printf '# comment\nDS_APM_HOST=10.0.0.5\n\nPG_MONITORING_PASSWORD=p@$$ w"x\r\nENABLE_POSTGRESQL=true\n' > "$tmp"
  ( parse_env_file "$tmp"
    assert_eq "10.0.0.5" "$DS_APM_HOST" "parse: 단순 값"
    assert_eq 'p@$$ w"x' "$PG_MONITORING_PASSWORD" "parse: 특수문자 비번 verbatim + CR 제거"
    assert_eq "true" "$ENABLE_POSTGRESQL" "parse: 후속 키" )
  rm -f "$tmp"
}
test_parse_env_file_missing() {
  if parse_env_file "/no/such/file" 2>/dev/null; then
    echo "FAIL: parse: 없는 파일은 return 1 이어야"; fail_mark
  else echo "ok: parse: 없는 파일 return 1"; fi
}
test_parse_env_file_allowlist() {
  # .env가 PATH/IFS 같은 셸 내부변수를 덮어쓰지 못하고, 미지 키는 무시해야 한다.
  local tmp; tmp="$(mktemp)"
  printf 'PATH=/evil\nIFS=,\nDS_APM_HOST=10.0.0.5\nBOGUS_KEY=x\n' > "$tmp"
  if ( before="$PATH"
       parse_env_file "$tmp" 2>/dev/null
       [[ "$PATH" == "$before" ]]       || exit 1   # PATH 안 덮여야
       [[ "$DS_APM_HOST" == "10.0.0.5" ]] || exit 1  # allowlist 키는 설정돼야
       [[ -z "${BOGUS_KEY:-}" ]]        || exit 1 ); then  # 미지 키는 무시돼야
    echo "ok: parse: allowlist — PATH/IFS/미지키 무시, 기대 키만 설정"
  else
    echo "FAIL: parse: allowlist 위반(PATH 덮임 or 미지 키 설정 or 기대 키 누락)"; fail_mark
  fi
  rm -f "$tmp"
}

test_parse_env_file
test_parse_env_file_missing
test_parse_env_file_allowlist

# ── map_arch ───────────────────────────────────────────
assert_eq "amd64" "$(map_arch x86_64)"  "map_arch: x86_64→amd64"
assert_eq "arm64" "$(map_arch aarch64)" "map_arch: aarch64→arm64"
if map_arch ppc64le >/dev/null 2>&1; then
  echo "FAIL: map_arch: 미지원 아키텍처는 return 1"; fail_mark
else echo "ok: map_arch: 미지원 return 1"; fi

# ── resolve_endpoint ──────────────────────────────────
( DS_APM_HOST="10.0.0.5"; DS_APM_ENDPOINT=""
  assert_eq "10.0.0.5:4317" "$(resolve_endpoint)" "resolve: HOST에 :4317 부착" )
( DS_APM_HOST="10.0.0.5"; DS_APM_ENDPOINT="apm.local:14317"
  assert_eq "apm.local:14317" "$(resolve_endpoint)" "resolve: ENDPOINT 우선" )

# ── validate_inputs ───────────────────────────────────
( DS_APM_HOST=""; DS_APM_ENDPOINT=""; ENABLE_POSTGRESQL="false"
  if validate_inputs 2>/dev/null; then echo "FAIL: validate: 빈 HOST 통과 안 돼야"; fail_mark;
  else echo "ok: validate: 빈 HOST 거부"; fi )
( DS_APM_HOST="10.0.0.5:4317"; DS_APM_ENDPOINT=""; ENABLE_POSTGRESQL="false"
  if validate_inputs 2>/dev/null; then echo "FAIL: validate: 포트포함 HOST 거부해야"; fail_mark;
  else echo "ok: validate: 포트포함 HOST 거부"; fi )
( DS_APM_HOST="10.0.0.5"; DS_APM_ENDPOINT=""; ENABLE_POSTGRESQL="true"
  PG_ENDPOINT="localhost:5432"; PG_MONITORING_USER="monitoring"; PG_MONITORING_PASSWORD=""
  if validate_inputs 2>/dev/null; then echo "FAIL: validate: PG 켰는데 빈 비번 거부해야"; fail_mark;
  else echo "ok: validate: PG 활성+빈 비번 거부"; fi )
( DS_APM_HOST="10.0.0.5"; DS_APM_ENDPOINT=""; ENABLE_POSTGRESQL="true"
  PG_ENDPOINT="localhost:5432"; PG_MONITORING_USER="monitoring"; PG_MONITORING_PASSWORD="pw"
  if validate_inputs 2>/dev/null; then echo "ok: validate: 유효 입력 통과";
  else echo "FAIL: validate: 유효 입력인데 거부됨"; fail_mark; fi )

# ── render_pg_databases ───────────────────────────────
assert_eq "[petclinic, foo]" "$(render_pg_databases 'petclinic,foo')" "pgdb: csv→flow 배열"
assert_eq "[petclinic]"      "$(render_pg_databases 'petclinic')"     "pgdb: 단일"
assert_eq "[]"               "$(render_pg_databases '')"              "pgdb: 빈값→전체"

# ── render_config: PG 끄면 ────────────────────────────
cfg_off="$( DS_APM_HOST="10.0.0.5"; DS_APM_ENDPOINT=""; TLS_INSECURE="true"
  ENABLE_POSTGRESQL="false"; COLLECTION_INTERVAL="60s"
  OTLP_GRPC_ENDPOINT="0.0.0.0:4317"; OTLP_HTTP_ENDPOINT="0.0.0.0:4318"
  render_config )"
assert_contains     "$cfg_off" "endpoint: \"10.0.0.5:4317\"" "cfg-off: exporter endpoint"
assert_contains     "$cfg_off" "hostmetrics:"                "cfg-off: hostmetrics 항상"
assert_contains     "$cfg_off" "mute_process_name_error: true" "cfg-off: process mute_*"
assert_not_contains "$cfg_off" "postgresql:"                 "cfg-off: postgresql 없음"

# ── render_config: PG 켜면 ────────────────────────────
cfg_on="$( DS_APM_HOST="10.0.0.5"; DS_APM_ENDPOINT=""; TLS_INSECURE="true"
  ENABLE_POSTGRESQL="true"; COLLECTION_INTERVAL="60s"
  OTLP_GRPC_ENDPOINT="0.0.0.0:4317"; OTLP_HTTP_ENDPOINT="0.0.0.0:4318"
  PG_ENDPOINT="localhost:5432"; PG_MONITORING_USER="monitoring"; PG_DATABASES="petclinic"
  render_config )"
assert_contains     "$cfg_on" "postgresql:"                    "cfg-on: postgresql receiver"
assert_contains     "$cfg_on" "username: monitoring"           "cfg-on: PG_MONITORING_USER 템플릿"
assert_contains     "$cfg_on" 'password: ${env:POSTGRES_PASSWORD}' "cfg-on: 비번은 env 참조만"
assert_contains     "$cfg_on" "databases: [petclinic]"         "cfg-on: databases 배열"
assert_contains     "$cfg_on" "receivers: [otlp, hostmetrics, postgresql]" "cfg-on: metrics 파이프라인에 postgresql"
# 비번 원문이 config에 새어들지 않는지(하드가드)
assert_not_contains "$cfg_on" "<CHANGE_ME>"                    "cfg-on: 비번 원문 없음"

# ── classify_smoke ────────────────────────────────────
assert_eq "success" "$(classify_smoke 'Everything is ready. Begin running and processing data.')" \
  "smoke: 깨끗한 로그→success"
out="$(classify_smoke 'error exporting items ... connection refused' || true)"
assert_contains "$out" "partial" "smoke: export 실패→partial"
assert_contains "$out" "DS_APM"  "smoke: export 실패는 DS_APM 도달성 지목"
out="$(classify_smoke 'postgresql scraper: pq: authentication failed for user monitoring' || true)"
assert_contains "$out" "partial" "smoke: PG 인증 실패→partial"
assert_contains "$out" "pg_hba"  "smoke: PG 인증 실패는 pg_hba/pg_monitor 지목"
# 기동 마커 '이후'의 export 실패는 진짜 지속 실패 → partial (거짓 success 방지)
out="$(classify_smoke "$(printf 'Everything is ready. Begin running and processing data.\nfailed to export metrics: connection refused; retrying\n')" || true)"
assert_contains "$out" "partial" "smoke: ready 뒤(기동 후) export 실패→partial(거짓성공 방지)"
assert_contains "$out" "DS_APM"  "smoke: 기동 후 실패는 DS_APM 도달성 지목"

# 기동 마커 '이전'의 과도기 오류만 있고 이후엔 깨끗 → success (부팅 재시도 노이즈 무시)
out="$(classify_smoke "$(printf 'connection refused; retrying\nEverything is ready. Begin running and processing data.\n')")"
assert_eq "success" "$out" "smoke: 기동 전 과도기 오류는 무시→success"

# 마커 없이 지속 export 실패 → partial
out="$(classify_smoke "$(printf 'connection refused\nfailed to export\ncontext deadline exceeded\n')" || true)"
assert_contains "$out" "partial" "smoke: 미기동+지속 export 실패→partial"
# return code 계약
if classify_smoke 'failed to export' >/dev/null; then
  echo "FAIL: smoke: partial은 return 3 이어야"; fail_mark
else echo "ok: smoke: partial return 비0"; fi

# 실 UAT 재현: 마커 이전의 postgresql config에러 + feature-gate warn만 있고 마커가 마지막 → success (오탐 방지)
out="$(classify_smoke "$(printf 'warn Configuration references unset environment variable POSTGRES_PASSWORD\nError: invalid configuration: receivers::postgresql: invalid config: missing password\nwarn postgresqlreceiver scraper.go Feature gate receiver.postgresql.separateSchemaAttr is not enabled\ninfo Everything is ready. Begin running and processing data.\n')")"
assert_eq "success" "$out" "smoke: 마커이전 postgresql config에러+feature-gate warn→success(오탐 방지)"

# 런타임(마커 이후) postgresql scrape 에러 → partial + postgresql 지목
out="$(classify_smoke "$(printf 'info Everything is ready. Begin running and processing data.\nerror scraper otelcol.component.id postgresql query failed: relation missing\n')" || true)"
assert_contains "$out" "partial"    "smoke: 마커이후 postgresql 쿼리에러→partial"
assert_contains "$out" "postgresql" "smoke: postgresql scrape 브랜치가 원인 지목"

# ── 부수효과: fakebin으로 외부 명령 mock ──────────────
make_fakebin() { # $1=dir  가짜 명령들을 PATH 앞에 얹는다
  local d="$1"; mkdir -p "$d"
  cat >"$d/dpkg-query" <<'SH'
#!/usr/bin/env bash
# FAKE_INSTALLED_VERSION 로 설치 버전 흉내
[[ -n "${FAKE_INSTALLED_VERSION:-}" ]] && { printf '%s' "$FAKE_INSTALLED_VERSION"; exit 0; }
exit 1
SH
  cat >"$d/dpkg" <<'SH'
#!/usr/bin/env bash
echo "fake dpkg $*" >>"$FAKE_LOG"
SH
  cat >"$d/curl" <<'SH'
#!/usr/bin/env bash
echo "fake curl $*" >>"$FAKE_LOG"
# -o 대상에 더미 .deb 생성
p=""; while [[ $# -gt 0 ]]; do [[ "$1" == "-o" ]] && { p="$2"; shift; }; shift; done
[[ -n "$p" ]] && echo "dummy" > "$p"
SH
  cat >"$d/systemctl" <<'SH'
#!/usr/bin/env bash
echo "fake systemctl $*" >>"$FAKE_LOG"
case "$*" in
  *"show -p EnvironmentFiles"*) echo "$FAKE_ENVFILE" ;;
  *"is-active"*) echo "active" ;;
esac
exit 0
SH
  cat >"$d/journalctl" <<'SH'
#!/usr/bin/env bash
printf '%s' "${FAKE_JOURNAL:-Everything is ready.}"
SH
  cat >"$d/otelcol-contrib" <<'SH'
#!/usr/bin/env bash
echo "fake otelcol $*" >>"$FAKE_LOG"
exit "${FAKE_VALIDATE_RC:-0}"
SH
  chmod +x "$d"/*
}

# installed_version 멱등 skip
( work="$(mktemp -d)"; make_fakebin "$work/bin"
  export PATH="$work/bin:$PATH" FAKE_LOG="$work/log" FAKE_INSTALLED_VERSION="0.139.0"
  OTELCOL_VERSION="0.139.0"
  install_collector >/dev/null 2>&1
  if grep -q "fake dpkg" "$work/log" 2>/dev/null; then
    echo "FAIL: install: 버전 일치인데 재설치함"; fail_mark
  else echo "ok: install: 버전 일치 시 skip"; fi
  rm -rf "$work" )

# inject_password: 재실행해도 줄 하나만(멱등) + 기존 EnvironmentFile 라인 보존
( work="$(mktemp -d)"; ef="$work/env.conf"
  # deb 기본 EnvironmentFile엔 OTELCOL_OPTIONS 같은 기존 라인이 있다 — 이게 보존돼야 한다.
  printf 'OTELCOL_OPTIONS="--config=/etc/otelcol-contrib/config.yaml"\n' > "$ef"
  PG_MONITORING_PASSWORD="secret1"; inject_password "$ef"
  PG_MONITORING_PASSWORD="secret2"; inject_password "$ef"
  n="$(grep -c '^POSTGRES_PASSWORD=' "$ef")"
  assert_eq "1" "$n" "inject: 비번 줄 하나만(교체)"
  assert_eq "POSTGRES_PASSWORD=secret2" "$(grep '^POSTGRES_PASSWORD=' "$ef")" "inject: 최신 값으로 교체"
  assert_eq "1" "$(grep -c '^OTELCOL_OPTIONS=' "$ef")" "inject: 기존 OTELCOL_OPTIONS 라인 보존"
  rm -rf "$work" )

# write_config → validate 실패 → rollback
( work="$(mktemp -d)"; make_fakebin "$work/bin"
  export PATH="$work/bin:$PATH" FAKE_LOG="$work/log"
  target="$work/config.yaml"; echo "OLD_CONFIG" > "$target"
  DS_APM_HOST="10.0.0.5"; DS_APM_ENDPOINT=""; TLS_INSECURE="true"; ENABLE_POSTGRESQL="false"
  COLLECTION_INTERVAL="60s"; OTLP_GRPC_ENDPOINT="0.0.0.0:4317"; OTLP_HTTP_ENDPOINT="0.0.0.0:4318"
  BACKUP_PATH=""
  write_config "$target" >/dev/null 2>&1
  # validate 실패를 흉내 → rollback. 가짜 otelcol은 자식 프로세스라 반드시 export해야
  # 값을 본다(바로 이 export 누락이 롤백 테스트를 무력화하는 대표 함정).
  export FAKE_VALIDATE_RC=1
  if run_validate "$target" >/dev/null 2>&1; then :; else rollback_config "$target"; fi
  assert_eq "OLD_CONFIG" "$(cat "$target")" "rollback: validate 실패 시 원본 복원"
  rm -rf "$work" )

# ERR 트랩 배선: main의 안전장치를 손 롤백이 아니라 실제 `trap ... ERR`로 검증.
# (set -e + trap 아래에서 run_validate가 실패하면 트랩이 rollback을 발화해야 한다)
( work="$(mktemp -d)"; make_fakebin "$work/bin"
  export PATH="$work/bin:$PATH" FAKE_LOG="$work/log"
  target="$work/config.yaml"; echo "OLD_CONFIG" > "$target"
  DS_APM_HOST="10.0.0.5"; DS_APM_ENDPOINT=""; TLS_INSECURE="true"; ENABLE_POSTGRESQL="false"
  COLLECTION_INTERVAL="60s"; OTLP_GRPC_ENDPOINT="0.0.0.0:4317"; OTLP_HTTP_ENDPOINT="0.0.0.0:4318"
  BACKUP_PATH=""; export FAKE_VALIDATE_RC=1   # 자식(가짜 otelcol)이 보게 export
  ( set -eE                                  # main과 동일: -E 없으면 함수내부 실패에 ERR 트랩 안 뜸
    write_config "$target"
    trap 'rollback_config "$target"' ERR   # main과 동일한 배선
    run_validate "$target"                 # 함수 내부에서 실패 → (errtrace 덕에) ERR 트랩 발화
    trap - ERR ) >/dev/null 2>&1
  # 함정 2가지가 이 테스트에 다 걸린다:
  #  (a) 서브셸에 `|| true`를 붙이면 `||` 좌변이라 내부 set -e가 억제돼 트랩이 안 뜬다.
  #  (b) `set -e`만 쓰면 run_validate(함수) 내부 실패엔 ERR 트랩이 안 뜬다 → `-E` 필수.
  # 바깥 테스트엔 set -e가 없으니 서브셸 비0 종료는 무해.
  assert_eq "OLD_CONFIG" "$(cat "$target")" "trap: ERR 트랩이 validate 실패 시 자동 롤백 발화"
  rm -rf "$work" )

# ── normalize_bool ────────────────────────────────────
assert_eq "true"  "$(normalize_bool True)"  "normbool: True→true"
assert_eq "true"  "$(normalize_bool TRUE)"  "normbool: TRUE→true"
assert_eq "true"  "$(normalize_bool yes)"   "normbool: yes→true"
assert_eq "true"  "$(normalize_bool true)"  "normbool: true→true"
assert_eq "false" "$(normalize_bool false)" "normbool: false→false"
assert_eq "false" "$(normalize_bool '')"    "normbool: 빈값→false"
assert_eq "false" "$(normalize_bool ture 2>/dev/null)" "normbool: 오타→false(경고)"

echo "----"
if [[ ! -s "$FAILFILE" ]]; then echo "ALL PASS"; rc=0; else echo "SOME FAILED"; rc=1; fi
rm -f "$FAILFILE"
exit "$rc"
