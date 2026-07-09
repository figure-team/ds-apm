# DS-APM OTel Collector — VM 원샷 설치기 (Ubuntu/Debian 계열)

리눅스 서버 한 대를 DS-APM으로 관측하기 위한 Collector의 설치·설정·검증·재시작·스모크 절차를
`.env` 파일 작성과 단일 명령으로 수행한다. 전체 개념 및 아키텍처는
[dsapm-petclinic-vm-test-scenario.md](../dsapm-petclinic-vm-test-scenario.md)를 참조한다.

> 본 문서의 명령 절차는 실제 리눅스(Ubuntu + systemd + PostgreSQL) 환경에서 수동 절차를 한 차례
> 수행하여 traces·hosts·postgresql 세 축이 DS-APM에 수집됨을 확인하며 정리하였다. 다만 `install.sh`
> 자동화 경로는 mock 기반 테스트(`tests/`)까지 검증된 상태이며, 실 VM 원샷 실행 검증(UAT)은 예정이다.

## 요약

```bash
cp dsapm-collector.env.example dsapm-collector.env
chmod 600 dsapm-collector.env && nano dsapm-collector.env   # DS_APM_HOST 등 값 입력
sudo ./install.sh
```

정상 완료 시 `✅ 설치 성공` 메시지와 함께 확인 대상 세 축이 안내된다(PostgreSQL 비활성 시 두 축).
`is-active`는 정상이나 실제 export·scrape가 실패한 경우 대부분 `⚠️ 부분 실패`(exit 3)로 원인 위치를
제시한다. 스모크 판정 창은 재시작 후 약 8초이므로, 그보다 늦게 드러나는 실패(예: RST 없이 패킷을
DROP하는 방화벽의 export 타임아웃)는 `✅` 이후에 나타날 수 있으며, 최종 판단은 DS-APM UI의 실제
데이터 도착으로 한다. config validate 실패 및 서비스 미기동은 `⚠️`가 아닌 즉시 오류 종료(exit 1)이다.

---

## 리눅스 서버 온보딩 절차

빈 리눅스 서버를 DS-APM에 연결하는 전체 명령을 순서대로 기술한다. DB 지표가 필요하지 않은 경우
1·2단계를 생략하고 `.env`에서 `ENABLE_POSTGRESQL=false`로 설정한다.

### 0. 사전 확인 — DS-APM 도달성

Collector는 서버에서 DS-APM 수신부(OTLP gRPC **4317**)로 아웃바운드 연결을 생성한다. 사전에
연결 가능 여부를 확인한다.

```bash
# <DS_APM_HOST> = DS-APM 수신 호스트 IP/도메인 (별도 서버면 해당 주소, 동일 호스트면 localhost)
nc -zv <DS_APM_HOST> 4317   # 또는:  timeout 3 bash -c '</dev/tcp/<DS_APM_HOST>/4317' && echo OPEN
```

### 1. PostgreSQL 모니터링 계정 (DB 지표 사용 시)

애플리케이션이 사용하는 DB 계정과 별개인 읽기 전용 모니터링 계정을 생성한다. `pg_monitor` 롤이
통계 뷰 읽기 권한을 부여한다.

```bash
sudo -u postgres psql <<'SQL'
CREATE USER monitoring WITH PASSWORD '<모니터링비번>';
GRANT pg_monitor TO monitoring;
SQL
```

> 비밀번호에 작은따옴표(`'`)가 포함되면 위 SQL이 실패한다. 해당 문자를 피하거나 `''`로 이스케이프한다.

### 2. 설치 전 필수 검증 — 모니터링 계정 TCP 접속

리눅스 환경에서 DB 지표가 수집되지 않는 가장 흔한 원인이 이 단계이다. Collector의 `postgresql`
리시버는 모니터링 계정으로 `localhost:5432`에 TCP 및 비밀번호로 접속하며, 기본 `pg_hba.conf`가 해당
경로를 차단하거나 권한이 없으면 실패한다. 설치 전에 Collector가 수행할 접속 경로를 동일하게 재현하여
확인한다.

```bash
PGPASSWORD='<모니터링비번>' psql -h localhost -U monitoring -d <대상DB> -tAc \
  "SELECT 1 FROM pg_stat_database LIMIT 1;"
# → '1' 이 반환되어야 한다 (TCP 리슨 + pg_hba + 비밀번호 + 대상 DB 접속 확인).
#   반환되지 않는 경우 다음을 점검한다:
#     - pg_hba.conf 의 host 라인(예: host all all 127.0.0.1/32 scram-sha-256) 존재 여부
#       (없으면 추가 후 sudo systemctl reload postgresql)
#     - PostgreSQL의 TCP(localhost:5432) 리슨 여부 (listen_addresses)
#     - 계정명 및 비밀번호
```

> `pg_stat_database`는 PUBLIC 읽기가 가능하므로 위 검증만으로는 `pg_monitor` 권한까지 확인되지 않는다.
> 권한은 별도로 확인한다.
>
> ```bash
> PGPASSWORD='<모니터링비번>' psql -h localhost -U monitoring -d <대상DB> -tAc \
>   "SELECT pg_has_role(current_user, 'pg_monitor', 'member');"   # → 't' 여야 한다
> ```

### 3. `.env` 작성 및 설치 실행

```bash
cp dsapm-collector.env.example dsapm-collector.env
chmod 600 dsapm-collector.env
nano dsapm-collector.env
```

최소 입력 값 (별도 DS-APM 서버 + DB 지표 사용 예시):

```bash
DS_APM_HOST=<DS_APM_HOST>          # 호스트만 입력 (포트 제외 — :4317 자동 부착)
OTELCOL_VERSION=0.139.0            # 버전 고정 권장 (빈 값은 실행 시마다 최신 추적, 멱등하지 않음)
ENABLE_POSTGRESQL=true
PG_ENDPOINT=localhost:5432
PG_MONITORING_USER=monitoring
PG_MONITORING_PASSWORD=<모니터링비번>
PG_DATABASES=<대상DB>              # 쉼표 구분, 빈 값은 전체 DB
```

```bash
sudo ./install.sh
```

### 4. 애플리케이션 계측 (OpenTelemetry Java Agent)

Collector 기동 이후에 애플리케이션을 실행한다(애플리케이션은 로컬 Collector의 otlp 리시버로 전송한다).

```bash
curl -Lo opentelemetry-javaagent.jar \
  https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar

java -javaagent:./opentelemetry-javaagent.jar \
  -Dotel.service.name=<서비스명> \
  -Dotel.exporter.otlp.endpoint=http://localhost:4317 \
  -Dotel.exporter.otlp.protocol=grpc \
  -jar <your-app>.jar
```

> 리시버 포트를 `.env`(`OTLP_GRPC_ENDPOINT`/`OTLP_HTTP_ENDPOINT`)에서 변경한 경우 `endpoint`도 동일
> 포트로 지정한다(예: `http://localhost:14317`).

### 5. 확인

```bash
systemctl is-active otelcol-contrib                                       # active
journalctl -u otelcol-contrib --no-pager | grep -i 'everything is ready'  # 기동 마커 확인
# (-n 20 사용 시, 기동 후 시간이 경과하면 마커가 최근 20줄 범위를 벗어날 수 있다)
```

애플리케이션에 트래픽을 발생시킨 뒤 DS-APM UI에서 세 축을 확인한다.
**Services/Traces** · **Infrastructure → Hosts** · **Metrics Explorer → `postgresql`**.

---

## 스모크 판정 해석 (`✅` / `⚠️`)

설치기는 `is-active`만으로 성공을 판단하지 않는다. 재시작 후 로그를 기동 마커(`Everything is ready`)
이후 구간으로 판정하여, 서비스는 기동되었으나 실제 데이터가 유입되지 않는 "active-but-empty" 상태를
검출한다.

| 출력 | 의미 | 우선 점검 항목 |
|------|------|----------------|
| `✅ 설치 성공` | active 상태 + 마커 이후 export·scrape 오류 없음 | DS-APM UI에서 세 축 확인 |
| `⚠️ export 실패` | Collector → DS-APM 전송 실패 | `DS_APM_HOST` 도달성 / 4317 / 방화벽 / TLS |
| `⚠️ postgresql 인증/권한 실패` | 모니터링 접속 또는 권한 문제 | 2단계 검증 재확인 (pg_hba / pg_monitor / 비밀번호 주입) |
| `⚠️ postgresql scrape 실패` | 접속은 성립하나 스크레이프 오류 | 5432 도달성 / `pg_monitor` / 대상 DB 존재 |

`⚠️` 출력 시에도 config는 유효(validate 통과)하여 서비스는 계속 동작하므로, 원인 조치 후 필요 시
재실행한다.

판정 범위에 관한 두 가지 유의 사항:

- exit 1은 위 표의 대상이 아니다. config validate 실패 및 서비스 미기동은 스모크 판정에 도달하지
  않고 즉시 오류 종료(exit 1)한다.
- 판정 창은 재시작 후 약 8초이다. 그 이후에 드러나는 실패(예: RST 없이 패킷을 DROP하는 방화벽의
  export 타임아웃)는 `✅` 출력 이후에도 잔존할 수 있으며, 최종 판단은 DS-APM UI의 실제 데이터
  도착으로 한다.

---

## 동작 개요 (install.sh)

1. `otelcol-contrib` deb 설치(동일 버전이 이미 설치된 경우 생략 — 멱등).
2. `.env` 값으로 `/etc/otelcol-contrib/config.yaml` 생성(기존 파일은 타임스탬프 백업 후 원자적 교체,
   owner·mode 보존). `otlp`+`hostmetrics`는 항상 포함하며, `ENABLE_POSTGRESQL=true`인 경우
   `postgresql` 리시버 및 metrics 파이프라인을 추가한다.
3. `ENABLE_POSTGRESQL=true`인 경우 모니터링 비밀번호를 systemd EnvironmentFile에 주입한다
   (비밀번호는 config.yaml에 기록하지 않는다).
4. `validate` 통과 후에만 재시작한다(실패 시 ERR 트랩으로 config 자동 롤백).
5. 재시작 후 로그 스모크로 export·scrape 실제 동작을 확인한다.

## 적용 범위 밖 (사전 또는 수동 조치 필요)

- PostgreSQL 설치 및 `monitoring` 계정 생성(1단계), `pg_hba.conf` 비밀번호 인증 허용, `pg_monitor`
  권한, 5432 방화벽.
- 애플리케이션 실행(4단계), Windows Server, SigNoz 클라우드 exporter(자가호스팅 전용).
- RPM 계열(RHEL/Rocky/SUSE 등) — 설치기가 deb/dpkg 전용이므로 미지원.

## 설정 값

`dsapm-collector.env.example`의 주석을 참조한다. 주요 항목은 다음과 같다.

- `DS_APM_HOST` — 호스트만 입력한다(포트 제외, `:4317` 자동 부착). 포트·스킴을 직접 지정하려면
  `DS_APM_ENDPOINT`를 사용한다.
- `OTELCOL_VERSION` — 버전 고정을 권장한다(빈 값은 최신 추적으로, 실행 시마다 자동 업그레이드되어
  멱등하지 않다).
- `ENABLE_POSTGRESQL` / `PG_*` — DB 지표 수집 토글.
- `OTLP_GRPC_ENDPOINT` / `OTLP_HTTP_ENDPOINT` — 애플리케이션 텔레메트리 수신 포트. 기본값
  4317/4318이 다른 프로세스에 점유된 경우 변경하며, 애플리케이션의 exporter endpoint도 동일 포트로
  맞춘다.
- `TLS_INSECURE` — 기본값 `true`(Collector → DS-APM 구간 TLS 미적용/미검증). 신뢰할 수 없는 네트워크
  경로에서는 `false`로 설정하고 인증서 신뢰를 구성한다.

## 테스트 (개발자용)

```bash
bash tests/test_install.sh   # 순수 로직 및 mock 통합 테스트. Git Bash에서도 실행 가능.
```
