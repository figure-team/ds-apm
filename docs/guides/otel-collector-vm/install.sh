#!/usr/bin/env bash
# DS-APM VM용 OTel Collector 원샷 설치기.
# 사용법:  cp dsapm-collector.env.example dsapm-collector.env
#          nano dsapm-collector.env    # 값 채움
#          sudo ./install.sh
# 설계: docs/superpowers/specs/2026-07-08-otel-collector-vm-installer-design.md
# 주의: 이 파일은 소스 가능하다(함수만 정의, main은 가드됨). set -e 는 main에서만.

ENV_FILE="${ENV_FILE:-dsapm-collector.env}"

# 기대 키 allowlist. source 대신 라인파싱으로 안전을 얻는 만큼, printf -v로도
# PATH/IFS/BASH_ENV 같은 셸 내부변수를 .env가 덮어쓰지 못하게 이 목록으로 게이트한다.
DSAPM_ALLOWED_KEYS=" DS_APM_HOST DS_APM_ENDPOINT OTELCOL_VERSION ENABLE_POSTGRESQL PG_ENDPOINT PG_MONITORING_USER PG_MONITORING_PASSWORD PG_DATABASES OTLP_GRPC_ENDPOINT OTLP_HTTP_ENDPOINT COLLECTION_INTERVAL TLS_INSECURE "

# .env를 source하지 않고 KEY=VALUE 라인만 파싱(비번 특수문자 안전).
# 값 내부는 verbatim, 값의 앞뒤 공백은 트림(따옴표 없는 .env 관례).
parse_env_file() {
  local path="$1" line key val
  if [[ ! -f "$path" ]]; then
    echo "오류: 설정 파일이 없습니다: $path" >&2
    echo "      cp dsapm-collector.env.example dsapm-collector.env 후 값을 채우세요." >&2
    return 1
  fi
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"                    # Windows CR 제거
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" =~ ^[[:space:]]*$ ]] && continue
    [[ "$line" != *"="* ]] && continue
    key="${line%%=*}"; val="${line#*=}"
    key="${key#"${key%%[![:space:]]*}"}"    # key 앞 공백
    key="${key%"${key##*[![:space:]]}"}"    # key 뒤 공백
    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || continue
    if [[ "$DSAPM_ALLOWED_KEYS" != *" $key "* ]]; then
      echo "경고: 알 수 없는 설정 키 무시: $key (오타이거나 예약 변수)" >&2
      continue                              # PATH/IFS 등 셸 내부변수 오염 차단
    fi
    val="${val#"${val%%[![:space:]]*}"}"    # val 앞 공백(트림 — 값 내부는 verbatim)
    val="${val%"${val##*[![:space:]]}"}"    # val 뒤 공백
    printf -v "$key" '%s' "$val"            # eval 없이 전역 설정(allowlist라 안전)
  done < "$path"
}

# .env 불리언 값 정규화: True/TRUE/yes/1/on → true, false/no/0/off/빈값 → false.
# 대소문자·표기 흔들림으로 옵션이 조용히 꺼지는 사고(예: ENABLE_POSTGRESQL=True인데
# PG 미설정 + 스모크는 success)를 막는다. 인식 못 한 값은 경고 후 false 처리(무성 off 방지).
normalize_bool() {
  case "$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]')" in
    true|yes|1|on)      echo "true" ;;
    false|no|0|off|"")  echo "false" ;;
    *) echo "경고: '$1'을(를) 불리언으로 해석할 수 없어 false로 처리합니다." >&2; echo "false" ;;
  esac
}

map_arch() {
  case "$1" in
    x86_64)          echo "amd64" ;;
    aarch64|arm64)   echo "arm64" ;;
    *) echo "오류: 미지원 아키텍처: $1 (amd64/arm64만 지원)" >&2; return 1 ;;
  esac
}

resolve_endpoint() {
  if [[ -n "${DS_APM_ENDPOINT:-}" ]]; then
    printf '%s' "$DS_APM_ENDPOINT"
  else
    printf '%s:4317' "${DS_APM_HOST:-}"
  fi
}

validate_inputs() {
  local ok=0
  if [[ -z "${DS_APM_ENDPOINT:-}" && -z "${DS_APM_HOST:-}" ]]; then
    echo "오류: DS_APM_HOST(또는 DS_APM_ENDPOINT)가 비었습니다." >&2; ok=1
  fi
  if [[ -z "${DS_APM_ENDPOINT:-}" && "${DS_APM_HOST:-}" == *:* ]]; then
    echo "오류: DS_APM_HOST에는 포트를 넣지 마세요(예: 10.0.0.5). install.sh가 :4317을 붙입니다." >&2
    echo "      포트/스킴을 직접 지정하려면 DS_APM_ENDPOINT를 쓰세요." >&2; ok=1
  fi
  if [[ "${ENABLE_POSTGRESQL:-false}" == "true" ]]; then
    [[ -z "${PG_ENDPOINT:-}" ]]          && { echo "오류: ENABLE_POSTGRESQL=true인데 PG_ENDPOINT가 비었습니다." >&2; ok=1; }
    [[ -z "${PG_MONITORING_USER:-}" ]]   && { echo "오류: ENABLE_POSTGRESQL=true인데 PG_MONITORING_USER가 비었습니다." >&2; ok=1; }
    [[ -z "${PG_MONITORING_PASSWORD:-}" || "${PG_MONITORING_PASSWORD:-}" == "<CHANGE_ME>" ]] \
      && { echo "오류: ENABLE_POSTGRESQL=true인데 PG_MONITORING_PASSWORD가 비었거나 <CHANGE_ME> 그대로입니다." >&2; ok=1; }
  fi
  return "$ok"
}

render_pg_databases() {
  local csv="$1"
  if [[ -z "$csv" ]]; then echo "[]"; return 0; fi
  local out="" item IFS=','
  local -                                   # 옵션 변경을 이 함수 안으로 격리(bash 4.4+)
  set -f                                    # DB명의 글롭 문자(*,?)가 파일로 전개되는 것 차단
  for item in $csv; do
    item="${item#"${item%%[![:space:]]*}"}"; item="${item%"${item##*[![:space:]]}"}"
    [[ -z "$item" ]] && continue
    out+="${out:+, }$item"
  done
  echo "[$out]"
}

render_config() {
  local endpoint tls interval
  endpoint="$(resolve_endpoint)"
  tls="${TLS_INSECURE:-true}"
  interval="${COLLECTION_INTERVAL:-60s}"

  cat <<YAML
# 이 파일은 install.sh가 생성했습니다. 직접 편집하면 다음 실행 때 덮어써집니다(백업됨).
receivers:
  otlp:
    protocols:
      grpc: { endpoint: ${OTLP_GRPC_ENDPOINT:-0.0.0.0:4317} }
      http: { endpoint: ${OTLP_HTTP_ENDPOINT:-0.0.0.0:4318} }

  hostmetrics:
    collection_interval: ${interval}
    scrapers:
      cpu: {}
      disk: {}
      load: {}
      filesystem: {}
      memory: {}
      network: {}
      paging: {}
      process:
        mute_process_name_error: true
        mute_process_exe_error: true
        mute_process_io_error: true
      processes: {}
YAML

  if [[ "${ENABLE_POSTGRESQL:-false}" == "true" ]]; then
    cat <<YAML

  postgresql:
    endpoint: ${PG_ENDPOINT}
    username: ${PG_MONITORING_USER}
    password: \${env:POSTGRES_PASSWORD}
    databases: $(render_pg_databases "${PG_DATABASES:-}")
    collection_interval: ${interval}
    tls:
      insecure: ${tls}
YAML
  fi

  cat <<YAML

processors:
  resourcedetection:
    detectors: [env, system]
  batch: {}

exporters:
  otlp:
    endpoint: "${endpoint}"
    tls:
      insecure: ${tls}

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resourcedetection, batch]
      exporters: [otlp]
    metrics:
      receivers: [otlp, hostmetrics$([[ "${ENABLE_POSTGRESQL:-false}" == "true" ]] && printf ', postgresql')]
      processors: [resourcedetection, batch]
      exporters: [otlp]
    logs:
      receivers: [otlp]
      processors: [resourcedetection, batch]
      exporters: [otlp]
YAML
}

classify_smoke() {
  local log="$1" scope
  # ── 판정은 기동 마커('Everything is ready') 이후(런타임)만 대상 ──
  # 마커 이전 로그(부팅 재시도, 이전 실패 인스턴스의 config 에러, feature-gate warn)는
  # 오탐이므로 제외한다. 마커가 없으면(미기동) 전체 로그가 대상.
  if grep -qi 'Everything is ready' <<<"$log"; then
    scope="$(awk 'f{print} /Everything is ready/{f=1}' <<<"$log")"
  else
    scope="$log"
  fi
  # ── 하드 실패(자가회복 안 됨): 인증/DNS/권한/인증서 ──
  if grep -qiE 'authentication failed|no such host|permission ?denied|x509|certificate.*(unknown|invalid|expired)' <<<"$scope"; then
    if grep -qiE 'authentication failed|pq:|postgresql' <<<"$scope"; then
      echo "partial: postgresql 인증/권한 실패 — pg_hba(monitoring localhost TCP 비번인증)/pg_monitor 권한/비번 주입을 확인하세요."
    else
      echo "partial: export 실패(하드) — DS_APM 도달성/포트(4317)/인증(ingestion-key)/TLS 인증서를 확인하세요."
    fi
    return 3
  fi
  # ── 소프트 export 실패(마커 이후 지속) ──
  if grep -qiE 'failed to export|connection refused|context deadline exceeded' <<<"$scope"; then
    echo "partial: export 실패(지속) — DS_APM_HOST 도달성/포트(4317)/방화벽/TLS를 확인하세요."
    return 3
  fi
  # ── postgresql scrape 실패(마커 이후, postgresql 관련 '동일 줄'에 에러) ──
  # feature-gate warn 등 무해한 postgresql 언급 줄은 에러 토큰이 없어 제외된다.
  if grep -iE 'postgresql' <<<"$scope" | grep -qiE 'error|failed|refused|denied'; then
    echo "partial: postgresql scrape 실패 — 5432 도달성/pg_monitor 권한/대상 DB 존재/비번 주입을 확인하세요."
    return 3
  fi
  echo "success"
  return 0
}

require_root() {
  if [[ "$(id -u)" != "0" ]]; then
    echo "오류: root 권한이 필요합니다. 'sudo ./install.sh'로 실행하세요." >&2; return 1
  fi
}

installed_version() {
  dpkg-query -W -f='${Version}' otelcol-contrib 2>/dev/null || true
}

install_collector() {
  local cur ver arch deb url
  cur="$(installed_version)"
  ver="${OTELCOL_VERSION:-}"
  if [[ -z "$ver" ]]; then
    echo "OTELCOL_VERSION 미지정 → GitHub 최신 릴리스 조회(재실행 시 자동 업그레이드될 수 있음)." >&2
    ver="$(curl -fsSL https://api.github.com/repos/open-telemetry/opentelemetry-collector-releases/releases/latest \
            | grep -oE '"tag_name":[[:space:]]*"v[^"]+"' | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)"
    if [[ -z "$ver" ]]; then
      echo "오류: 최신 버전 조회 실패(rate limit 가능). .env의 OTELCOL_VERSION을 직접 지정하세요." >&2
      return 1
    fi
  fi
  if [[ -n "$cur" && "$cur" == "$ver" ]]; then
    echo "otelcol-contrib $ver 이미 설치됨 → 설치 건너뜀."; return 0
  fi
  arch="$(map_arch "$(uname -m)")" || return 1
  deb="otelcol-contrib_${ver}_linux_${arch}.deb"
  url="https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v${ver}/${deb}"
  echo "otelcol-contrib ${ver} (${arch}) 다운로드..."
  curl -fSL -o "/tmp/${deb}" "$url"
  dpkg -i "/tmp/${deb}"
  rm -f "/tmp/${deb}"                       # 다운로드 잔여물 정리(재실행 누적 방지)
}

env_file_path() {
  local p
  p="$(systemctl show -p EnvironmentFiles --value otelcol-contrib 2>/dev/null | awk '{print $1}' | head -1)"
  # systemd는 "path (mode)" 형태를 낼 수 있어 경로만 취함
  p="${p%% *}"
  [[ -z "$p" ]] && p="/etc/otelcol-contrib/otelcol-contrib.conf"
  echo "$p"
}

write_config() {
  local target="$1" owner mode tmp ts
  if [[ -e "$target" ]]; then
    owner="$(stat -c '%U:%G' "$target" 2>/dev/null || echo 'root:root')"
    mode="$(stat -c '%a' "$target" 2>/dev/null || echo '640')"
    ts="$(date +%Y%m%d-%H%M%S)"
    BACKUP_PATH="${target}.bak-${ts}"
    cp -p "$target" "$BACKUP_PATH"
    echo "기존 config 백업: $BACKUP_PATH"
    # 백업 5개 초과분 정리
    ls -1t "${target}".bak-* 2>/dev/null | tail -n +6 | xargs -r rm -f
  else
    owner="root:root"; mode="640"; BACKUP_PATH=""
  fi
  # 원자적 교체: 대상과 같은 디렉터리에 임시 렌더 → mv(같은 파일시스템 rename=원자적).
  # install/cp는 open-truncate-write라 부분쓰기 창이 있어, 실패 시 깨진 config가 디스크에
  # 남아 다음 재시작/재부팅 때 죽는다. mv는 대상을 절대 부분쓰기 상태로 남기지 않는다.
  local dir; dir="$(dirname "$target")"
  tmp="$(mktemp "${dir}/.config.yaml.new.XXXXXX")"
  render_config > "$tmp"
  chmod "$mode" "$tmp"
  chown "$owner" "$tmp" 2>/dev/null || true   # 리눅스: 원본 owner/mode 보존. Git Bash엔 chown 무의미(폴백)
  mv -f "$tmp" "$target"                       # 원자적 rename
}

rollback_config() {
  local target="$1"
  if [[ -n "${BACKUP_PATH:-}" && -e "$BACKUP_PATH" ]]; then
    cp -p "$BACKUP_PATH" "$target"
    echo "config 롤백: $BACKUP_PATH → $target" >&2
  elif [[ -z "${BACKUP_PATH:-}" && -e "$target" ]]; then
    # 최초 설치라 백업이 없다 → 방금 쓴 불량 config가 디스크에 남지 않게 제거
    rm -f "$target"
    echo "config 롤백: 최초 설치(백업 없음) → 방금 쓴 불량 config 제거: $target" >&2
  fi
}

inject_password() {
  local ef="$1"
  [[ -e "$ef" ]] || : >"$ef"
  grep -v '^POSTGRES_PASSWORD=' "$ef" > "${ef}.tmp" 2>/dev/null || : >"${ef}.tmp"
  printf 'POSTGRES_PASSWORD=%s\n' "$PG_MONITORING_PASSWORD" >> "${ef}.tmp"
  mv "${ef}.tmp" "$ef"
  chmod 600 "$ef"
}

run_validate() {
  local cfg="$1"
  otelcol-contrib validate --config "$cfg"
}

main() {
  # -E(errtrace) 필수: run_validate는 함수라 실패가 함수 '내부'에서 난다.
  # errtrace 없으면 set -e는 종료시키지만 ERR 트랩이 안 떠서 config 롤백이 안 된다(검증됨).
  set -eEuo pipefail
  # config 경로는 deb + 시나리오 4·5장이 고정하는 상수라 의도적 가정(EnvironmentFile만 동적 조회).
  local cfg="/etc/otelcol-contrib/config.yaml" ef restart_ts smoke_out smoke_rc

  require_root
  parse_env_file "$ENV_FILE"
  # .env 권한 강제(평문 비번 보호)
  if [[ -f "$ENV_FILE" && "$(stat -c '%a' "$ENV_FILE" 2>/dev/null || echo 600)" != "600" ]]; then
    echo "경고: $ENV_FILE 권한을 600으로 강제합니다(평문 비번 보호)." >&2
    chmod 600 "$ENV_FILE" || true
  fi
  # 불리언 값 정규화(대소문자/표기 흔들림 흡수) — 게이트 이전에 1회.
  ENABLE_POSTGRESQL="$(normalize_bool "${ENABLE_POSTGRESQL:-false}")"
  validate_inputs

  install_collector
  write_config "$cfg"

  # validate 실패 시 config 자동 롤백
  trap 'rollback_config "$cfg"' ERR
  if [[ "${ENABLE_POSTGRESQL:-false}" == "true" ]]; then
    ef="$(env_file_path)"; inject_password "$ef"
    echo "모니터링 비번을 EnvironmentFile에 주입: $ef"
  fi
  POSTGRES_PASSWORD="${PG_MONITORING_PASSWORD:-}" run_validate "$cfg"
  trap - ERR

  restart_ts="$(date '+%Y-%m-%d %H:%M:%S')"
  systemctl restart otelcol-contrib
  sleep 3
  if [[ "$(systemctl is-active otelcol-contrib)" != "active" ]]; then
    echo "오류: 재시작 후 서비스가 active가 아닙니다. 최근 로그:" >&2
    journalctl -u otelcol-contrib -n 20 --no-pager >&2 || true
    exit 1
  fi

  # 사후 스모크(active-but-empty 탐지)
  sleep 5
  smoke_out="$(classify_smoke "$(journalctl -u otelcol-contrib --since "$restart_ts" --no-pager 2>/dev/null)")" \
    && smoke_rc=0 || smoke_rc=$?
  if [[ "$smoke_rc" -eq 0 ]]; then
    echo "✅ 설치 성공. DS-APM UI에서 세 축을 확인하세요:"
    echo "   - Infrastructure → Hosts (이 VM)"
    echo "   - Services / Traces (앱 실행 후)"
    [[ "${ENABLE_POSTGRESQL:-false}" == "true" ]] && echo "   - Metrics Explorer → 'postgresql'"
    echo "ℹ️  process 지표: deb 콜렉터는 저권한 유저라 JVM 등 타 계정 프로세스의 실행경로·IO가 비어 보일 수 있음."
    echo "   완전한 프로세스 지표가 필요하면 콜렉터를 root로 돌리거나 읽기 권한을 부여하세요."
  else
    echo "⚠️  서비스는 active이지만 스모크에서 문제가 감지됐습니다(부분 실패):" >&2
    echo "   $smoke_out" >&2
    echo "   트러블슈팅: docs/guides/dsapm-petclinic-vm-test-scenario.md 의 표 참조." >&2
    exit "$smoke_rc"
  fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
