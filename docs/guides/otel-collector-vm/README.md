# DS-APM 리눅스 서버(VM) 설치 가이드 — OTel Collector + Java Agent

리눅스 서버 한 대를 DS-APM으로 관측하기 위한 전체 절차이다. 위에서부터 순서대로 명령을
실행하면 다음 세 가지가 수집된다.

| 수집 항목 | 담당 구성요소 | DS-APM UI 확인 위치 |
|-----------|--------------|---------------------|
| 애플리케이션 트레이스 (APM) | OpenTelemetry Java Agent → 로컬 Collector | Services / Traces |
| 호스트 지표 (CPU·메모리·디스크) | Collector `hostmetrics` | Infrastructure → Hosts |
| PostgreSQL 지표 (선택) | Collector `postgresql` 리시버 | Metrics Explorer → `postgresql` |

```
[Java 앱] --(javaagent, localhost:4317)--> [이 서버의 OTel Collector] --(4317)--> [DS-APM 서버]
[호스트 지표 / PostgreSQL 지표] ----------↗
```

## 설치 방법은 두 가지

Collector를 설치·설정하는 부분(아래 3단계)만 두 갈래이고, 나머지 단계는 공통이다.

- **방법 A — `install.sh` 한 줄 설치**: `.env` 파일 하나를 채우고 `sudo ./install.sh`를 실행한다.
  deb 설치·설정 생성·비밀번호 주입·검증·재시작을 자동으로 수행한다.
- **방법 B — 수동 설치**: 같은 작업을 명령 하나씩 수행한다. install.sh를 사용할 수 없는 환경이거나
  각 단계를 직접 확인해야 하는 경우에 사용한다.

두 방법 모두 1·2단계(사전 준비)를 먼저 하고, 3단계에서 A 또는 B 하나를 수행한 뒤, 4단계
이후(Java Agent·확인)를 공통으로 이어서 한다.

**실행 위치**: 별도 표기가 없는 한 모든 명령은 **관측 대상 리눅스 서버**에 SSH로 접속한
터미널에서 실행한다. (DS-APM 서버 자체에서 실행하는 명령은 없다. DS-APM 서버 자신은 이미
수신용 Collector가 있으므로 이 절차의 대상이 아니다.)

**지원 환경**: Ubuntu/Debian 계열(x86_64/arm64) + systemd. RPM 계열(RHEL/Rocky 등)은 방법 B의
deb 설치만 rpm으로 바꾸면 나머지는 동일하다(install.sh는 deb 전용).

---

## 1단계. DS-APM 도달성 확인 (공통)

Collector는 이 서버에서 DS-APM 수신부(OTLP gRPC **4317** 포트)로 아웃바운드 연결을 만든다.
`<DS_APM_HOST>`는 DS-APM 서버의 IP/도메인이다.

```bash
nc -zv <DS_APM_HOST> 4317
```

실행 결과: `... 4317 port [tcp/*] succeeded!` 또는 `Connection ... succeeded`가 출력된다.

- `nc`가 없으면: `timeout 3 bash -c '</dev/tcp/<DS_APM_HOST>/4317' && echo OPEN`
- 연결이 실패하면 DS-APM 서버 주소·방화벽(4317 인바운드)을 먼저 해결한다.

## 2단계. PostgreSQL 모니터링 계정 준비 (DB 지표를 쓸 때만, 공통)

DB 지표가 필요 없으면 이 단계 전체를 건너뛴다. 이 계정 준비는 방법 A/B 모두 자동화되지 않으므로
직접 수행한다.

### 2-1. 모니터링 전용 계정 생성

애플리케이션 계정과 별개인 읽기 전용 계정을 만든다. `pg_monitor` 롤이 통계 뷰 읽기 권한을
부여한다. `<모니터링비번>`을 비밀번호로 바꿔 실행한다.

```bash
sudo -u postgres psql <<'SQL'
CREATE USER monitoring WITH PASSWORD '<모니터링비번>';
GRANT pg_monitor TO monitoring;
SQL
```

> 비밀번호에 작은따옴표(`'`)가 들어가면 위 SQL이 실패한다. 해당 문자를 피하거나 `''`로 이스케이프한다.

### 2-2. 모니터링 계정 TCP 접속 검증

Collector는 이 계정으로 `localhost:5432`에 TCP + 비밀번호로 접속한다. 기본 `pg_hba.conf`가 이 경로를
막는 경우가 있으므로, Collector와 동일한 접속을 미리 실행해 확인한다. `<대상DB>`는 지표를 수집할
DB 이름이다.

```bash
PGPASSWORD='<모니터링비번>' psql -h localhost -U monitoring -d <대상DB> -tAc \
  "SELECT 1 FROM pg_stat_database LIMIT 1;"
# → '1' 이 출력된다.
```

`1`이 출력되지 않으면 다음을 점검한다.

1. `pg_hba.conf`에 `host all all 127.0.0.1/32 scram-sha-256` 같은 host 라인이 있는지
   (없으면 추가 후 `sudo systemctl reload postgresql`)
2. PostgreSQL이 TCP를 듣고 있는지 (`postgresql.conf`의 `listen_addresses`)
3. 계정명·비밀번호 오타

권한도 별도로 확인한다 (`pg_stat_database`는 PUBLIC 읽기가 가능하여 위 검증만으로는 권한이 확인되지
않는다):

```bash
PGPASSWORD='<모니터링비번>' psql -h localhost -U monitoring -d <대상DB> -tAc \
  "SELECT pg_has_role(current_user, 'pg_monitor', 'member');"
# → 't' 가 출력된다.
```

---

## 3단계. Collector 설치 — 방법 A 또는 방법 B (하나만)

### 방법 A — `install.sh` 한 줄 설치

#### A-1. 설치기 파일을 서버로 가져오기

`install.sh`와 `dsapm-collector.env.example`은 이 저장소의
`docs/guides/otel-collector-vm/`에 있다. 대상 서버에 저장소를 클론하거나 폴더만 복사한다.

```bash
# (예: 저장소 클론)
git clone <ds-apm-repo-url> && cd ds-apm/docs/guides/otel-collector-vm

# 또는 로컬 PC에서 폴더만 서버로 전송
#   scp -r docs/guides/otel-collector-vm  user@<서버>:~/otel-collector-vm
```

#### A-2. `.env` 작성

예시 파일을 복사해 값을 채운다. 권한 600으로 잠가 비밀번호 평문을 보호한다.

```bash
cp dsapm-collector.env.example dsapm-collector.env
chmod 600 dsapm-collector.env
nano dsapm-collector.env
```

최소 입력 값 (별도 DS-APM 서버 + DB 지표 사용 예시):

```bash
DS_APM_HOST=<DS_APM_HOST>          # 호스트만 (포트 제외 — install.sh가 :4317을 붙임)
OTELCOL_VERSION=0.139.0            # 버전 고정 (빈 값이면 매 실행 최신 추적 — 멱등 아님)
ENABLE_POSTGRESQL=true             # DB 지표 안 쓰면 false (아래 PG_* 무시됨)
PG_ENDPOINT=localhost:5432
PG_MONITORING_USER=monitoring
PG_MONITORING_PASSWORD=<모니터링비번>   # 2-1에서 만든 비밀번호
PG_DATABASES=<대상DB>              # 쉼표 구분, 빈 값이면 전체 DB
```

#### A-3. 실행

```bash
sudo ./install.sh
```

실행 결과:

- `✅ 설치 성공`: deb 설치·`/etc/otelcol-contrib/config.yaml` 생성·비밀번호 주입·검증·재시작이
  완료된 것이다. 4단계로 진행한다.
- `⚠️ 부분 실패`: 서비스는 기동됐으나 전송/스크레이프에 문제가 있는 상태다. 메시지가 원인 위치를
  알려준다. 6단계 문제 해결 표를 참고해 조치 후 다시 실행한다. install.sh는 멱등하므로 값을 고쳐
  재실행할 수 있다.

install.sh는 소스 가능한 형태이며(함수 정의 + 가드된 main), 상세 동작과 설정 값은 파일 자체의
주석과 `dsapm-collector.env.example`에 기술돼 있다.

---

### 방법 B — 수동 설치

방법 A를 수행했다면 이 절을 건너뛰고 4단계로 간다.

#### B-1. 환경 변수 준비

이후 명령들이 참조할 값을 현재 터미널에 설정한다. 값을 환경에 맞게 고쳐 실행한다.

```bash
DSAPM_HOST="10.0.0.5"        # ← DS-APM 서버의 IP 또는 도메인 (포트 붙이지 말 것)
OTELCOL_VERSION="0.139.0"    # ← Collector 버전
```

> 이 변수는 현재 터미널에서만 유효하다. 터미널을 새로 열었다면 다시 실행하고 이어서 한다.

#### B-2. Collector deb 설치

```bash
ARCH=$(dpkg --print-architecture)    # amd64 또는 arm64 자동 감지
curl -fL -o /tmp/otelcol-contrib.deb \
  "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v${OTELCOL_VERSION}/otelcol-contrib_${OTELCOL_VERSION}_linux_${ARCH}.deb"
sudo dpkg -i /tmp/otelcol-contrib.deb
rm -f /tmp/otelcol-contrib.deb

systemctl is-active otelcol-contrib && otelcol-contrib --version
```

#### B-3. 설정 파일 작성

설정 파일 위치는 `/etc/otelcol-contrib/config.yaml`이다. PostgreSQL 지표 수집 여부에 따라
**B-3-가와 B-3-나 중 하나만** 실행한다.

**B-3-가. 기본 (앱 트레이스 + 호스트 지표)**

```bash
sudo tee /etc/otelcol-contrib/config.yaml > /dev/null <<EOF
receivers:
  otlp:                      # 이 서버의 앱(javaagent)이 보내는 텔레메트리 수신
    protocols:
      grpc: { endpoint: 0.0.0.0:4317 }
      http: { endpoint: 0.0.0.0:4318 }

  hostmetrics:               # 호스트 CPU/메모리/디스크/네트워크 지표
    collection_interval: 60s
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

processors:
  resourcedetection:
    detectors: [env, system]
  batch: {}

exporters:
  otlp:                      # DS-APM 서버로 전송
    endpoint: "${DSAPM_HOST}:4317"
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resourcedetection, batch]
      exporters: [otlp]
    metrics:
      receivers: [otlp, hostmetrics]
      processors: [resourcedetection, batch]
      exporters: [otlp]
    logs:
      receivers: [otlp]
      processors: [resourcedetection, batch]
      exporters: [otlp]
EOF
```

→ **B-4로 진행** (B-3-나는 건너뛴다).

**B-3-나. PostgreSQL 지표 포함**

`PG_DATABASE`만 대상 DB 이름으로 고쳐 실행한다. 비밀번호는 이 파일에 쓰지 않고 B-4에서 별도
파일로 주입한다. YAML의 `\${env:POSTGRES_PASSWORD}`는 리터럴 그대로 기록된다.

```bash
PG_DATABASE="petclinic"      # ← 지표를 수집할 DB 이름 (여러 개면 쉼표 구분: db1, db2)

sudo tee /etc/otelcol-contrib/config.yaml > /dev/null <<EOF
receivers:
  otlp:                      # 이 서버의 앱(javaagent)이 보내는 텔레메트리 수신
    protocols:
      grpc: { endpoint: 0.0.0.0:4317 }
      http: { endpoint: 0.0.0.0:4318 }

  hostmetrics:               # 호스트 CPU/메모리/디스크/네트워크 지표
    collection_interval: 60s
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

  postgresql:                # PostgreSQL 지표
    endpoint: localhost:5432
    username: monitoring
    password: \${env:POSTGRES_PASSWORD}
    databases: [${PG_DATABASE}]
    collection_interval: 60s
    tls:
      insecure: true

processors:
  resourcedetection:
    detectors: [env, system]
  batch: {}

exporters:
  otlp:                      # DS-APM 서버로 전송
    endpoint: "${DSAPM_HOST}:4317"
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resourcedetection, batch]
      exporters: [otlp]
    metrics:
      receivers: [otlp, hostmetrics, postgresql]
      processors: [resourcedetection, batch]
      exporters: [otlp]
    logs:
      receivers: [otlp]
      processors: [resourcedetection, batch]
      exporters: [otlp]
EOF
```

비밀번호 주입 (systemd EnvironmentFile). config.yaml의 `${env:POSTGRES_PASSWORD}`가 기동 시
이 값으로 치환된다. `<모니터링비번>`을 바꿔 실행한다.

```bash
echo 'POSTGRES_PASSWORD=<모니터링비번>' | sudo tee -a /etc/otelcol-contrib/otelcol-contrib.conf > /dev/null
sudo chmod 600 /etc/otelcol-contrib/otelcol-contrib.conf
```

#### B-4. 검증 및 재시작

재시작 전에 설정 문법·참조를 검증한다.

```bash
sudo bash -c 'set -a; [ -f /etc/otelcol-contrib/otelcol-contrib.conf ] && . /etc/otelcol-contrib/otelcol-contrib.conf; \
  otelcol-contrib validate --config /etc/otelcol-contrib/config.yaml' && echo "✅ config 유효"
```

`✅ config 유효`가 출력되면 재시작한다. 오류가 출력되면 B-3 설정 파일을 다시 확인한다
(들여쓰기 2칸, 탭 금지).

```bash
sudo systemctl restart otelcol-contrib
sudo systemctl enable otelcol-contrib    # 부팅 시 자동 기동
```

동작 확인:

```bash
systemctl is-active otelcol-contrib                                       # → active
sudo journalctl -u otelcol-contrib --since '2 min ago' --no-pager | grep -i 'everything is ready'
sudo journalctl -u otelcol-contrib --since '2 min ago' --no-pager | grep -iE 'failed|refused|denied' \
  || echo "✅ 오류 없음"
```

---

## 4단계. Java Agent 설치 및 애플리케이션 계측 (공통)

여기까지(방법 A 또는 B)로 호스트 지표 수집이 완료된다. 1~2분 뒤 DS-APM UI →
**Infrastructure → Hosts**에 이 서버가 나타난다.

애플리케이션 트레이스(Services/Traces)를 위해 앱 JVM에 OpenTelemetry Java Agent를 붙인다.
코드 수정은 필요 없다. 이 단계는 install.sh 범위 밖이므로 방법 A로 설치한 경우에도 직접 수행한다.

에이전트 다운로드 (서버 공용 위치에 1회):

```bash
sudo mkdir -p /opt/opentelemetry
sudo curl -fL -o /opt/opentelemetry/opentelemetry-javaagent.jar \
  https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
```

> 여러 서버에 배포한다면 `latest` 대신 버전 고정 URL을 쓴다:
> `https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v2.14.0/opentelemetry-javaagent.jar`

### 4-A. 터미널에서 직접 실행하는 앱

기존 `java -jar ...` 실행 명령에 `-javaagent`와 OTel 옵션을 추가한다. `<서비스명>`은 DS-APM
Services 화면에 표시될 이름, `<your-app>.jar`는 실제 jar로 바꾼다.

```bash
java -javaagent:/opt/opentelemetry/opentelemetry-javaagent.jar \
  -Dotel.service.name=<서비스명> \
  -Dotel.exporter.otlp.endpoint=http://localhost:4317 \
  -Dotel.exporter.otlp.protocol=grpc \
  -jar <your-app>.jar
```

### 4-B. systemd 서비스로 실행되는 앱

앱의 유닛 파일을 고치지 않고 drop-in으로 환경변수만 얹는다. `<앱서비스명>`을 실제 서비스
이름으로, `my-app`을 표시할 서비스명으로 바꾼다.

```bash
sudo systemctl edit <앱서비스명>
```

열린 편집기에 아래를 입력하고 저장한다:

```ini
[Service]
Environment="JAVA_TOOL_OPTIONS=-javaagent:/opt/opentelemetry/opentelemetry-javaagent.jar"
Environment="OTEL_SERVICE_NAME=my-app"
Environment="OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317"
Environment="OTEL_EXPORTER_OTLP_PROTOCOL=grpc"
```

```bash
sudo systemctl daemon-reload
sudo systemctl restart <앱서비스명>
```

> Collector의 OTLP 수신 포트(4317/4318)를 바꿨다면 endpoint도 같은 포트로 맞춘다.

## 5단계. 최종 확인 (공통)

앱에 트래픽을 몇 번 발생시킨 뒤(브라우저 접속 또는 `curl http://localhost:<앱포트>/...`),
DS-APM UI에서 확인한다. 반영까지 1~2분 걸릴 수 있다.

| 확인 위치 | 기대 결과 |
|-----------|----------|
| **Services** | 지정한 서비스명이 목록에 표시 |
| **Traces** | 방금 발생시킨 요청의 트레이스 |
| **Infrastructure → Hosts** | 이 서버의 호스트네임과 CPU/메모리 지표 |
| **Metrics Explorer → `postgresql` 검색** | (PG 활성 시) `postgresql.*` 지표 |

세 축이 모두 표시되면 설치가 완료된 것이다.

## 6단계. 문제 해결 (공통)

먼저 Collector 로그를 확인한다.

```bash
sudo journalctl -u otelcol-contrib -f    # 실시간 로그 (Ctrl+C로 종료)
```

| 증상 / 로그 | 원인 | 조치 |
|-------------|------|------|
| `failed to export` / `connection refused` / `context deadline exceeded` | Collector → DS-APM 전송 실패 | 1단계 도달성 재확인 (DS-APM 주소, 4317 방화벽), config의 `exporters.otlp.endpoint` 값 확인 |
| `authentication failed` (postgresql 관련) | 모니터링 계정 접속 거부 | 2-2 검증 재수행 — `pg_hba.conf` host 라인, 비밀번호, 주입 파일 확인 |
| `postgresql` 줄에 `error`/`failed` | 접속은 되나 스크레이프 실패 | 대상 DB 존재 여부, `pg_monitor` 권한(2-2 두 번째 검증) |
| 서비스가 `active`가 아님 / 재시작 반복 | config 문법 오류 | B-4 validate를 다시 실행해 오류 메시지 확인 |
| Hosts는 보이는데 Services가 안 보임 | 앱 계측 문제 | 앱 기동 로그에 javaagent 배너가 있는지, endpoint 포트가 맞는지 확인 |
| UI에 아무것도 안 나옴 (로그는 깨끗) | 방화벽이 패킷을 조용히 DROP | 수십 초 기다린 뒤 로그 재확인, DS-APM 서버 쪽 인바운드 4317 확인 |
| 프로세스 목록 지표가 비어 보임 | deb 패키지의 Collector가 저권한 유저로 실행됨 | 정상 동작. JVM 등 타 계정 프로세스의 실행경로·IO는 권한상 안 보일 수 있음 |

설정을 고친 뒤에는 항상 검증 → 재시작을 다시 한다(방법 A는 `sudo ./install.sh` 재실행,
방법 B는 B-4).

---

## 부록: 파일·서비스 위치 요약

| 항목 | 경로 |
|------|------|
| install.sh 설치기 | `docs/guides/otel-collector-vm/` (install.sh, dsapm-collector.env.example) |
| Collector 설정 | `/etc/otelcol-contrib/config.yaml` |
| Collector 환경 파일 (PG 비밀번호) | `/etc/otelcol-contrib/otelcol-contrib.conf` |
| Collector systemd 서비스 | `otelcol-contrib` (`systemctl status otelcol-contrib`) |
| Java Agent | `/opt/opentelemetry/opentelemetry-javaagent.jar` |
| Collector 로그 | `journalctl -u otelcol-contrib` |
