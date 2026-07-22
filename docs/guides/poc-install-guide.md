# DS-APM PoC 설치 가이드

React(브라우저) + Node.js + Python/FastAPI 고객 환경에 DS-APM을 붙이기 위한 단계별 설치 가이드다.
구성은 두 덩어리로 나뉜다.

- **관제 서버 1대**: DS-APM 스택 전체(Docker Compose). 여기에만 설치한다.
- **고객 앱 서버**: DS-APM 설치 불필요. OTel 계측 라이브러리만 붙이고 관제 서버로 전송한다.

```
[고객 앱 서버들]                          [관제 서버]
FastAPI ──(OTLP)──┐                ┌──────────────────────────┐
Node.js ──(OTLP)──┼──→ :4317/:4318 │ otel-collector           │
React   ──(OTLP)──┘                │   ↓                      │
                                   │ ClickHouse ← ds-apm 백엔드│→ :8080 (웹 UI)
                                   └──────────────────────────┘
```

| 구분 | 포트 | 용도 |
|---|---|---|
| ds-apm | 8080 | 웹 UI + API 백엔드 |
| ds-apm-otel-collector | 4317 | OTLP gRPC 수신 |
| ds-apm-otel-collector | 4318 | OTLP HTTP 수신 (브라우저 계측은 이 포트) |

진행 순서: **1장(로컬 PoC 기동) → 2장(앱 계측) → 3장(검증) → 4장(폐쇄망 반입 번들)**.

---

## 1. 관제 서버: DS-APM 스택 설치 (로컬 PoC — Windows + Docker Desktop)

### 1-1. 사전 조건

- Windows 10/11 + **Docker Desktop** (WSL2 백엔드) 설치·실행 중
- 디스크 여유 20GB 이상 (이미지 + ClickHouse 데이터)
- 메모리 8GB 이상 권장 (ClickHouse가 메모리를 씀)
- 포트 8080, 4317, 4318이 비어 있을 것

확인 (PowerShell):

```powershell
docker version          # Client/Server 둘 다 출력되면 정상
docker compose version  # v2.x
netstat -ano | findstr ":8080 :4317 :4318"   # 아무것도 안 나오면 포트 비어있음
```

### 1-2. 소스 준비

ds-apm 레포 폴더를 통째로 복사(또는 clone)한다. 빌드에 `cmd/`, `pkg/`, `frontend/`, `deploy/`, `templates/`, `go.mod`, `Dockerfile.local`이 필요하므로 **레포 루트 전체**가 있어야 한다.

```powershell
git clone <ds-apm 레포 URL> C:\poc\ds-apm
cd C:\poc\ds-apm\deploy\docker
```

### 1-3. 환경 파일(.env) 구성

`deploy/docker/.env` 파일을 만든다(이미 있으면 내용 확인). `COMPOSE_FILE`을 지정해 두면 이후 `docker compose` 명령에 `-f`를 매번 붙일 필요가 없다.

```powershell
# 암호화 키 생성 (AI 설정 저장용, 32바이트 base64)
$key = [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))

@"
COMPOSE_FILE=docker-compose.poc.yaml
DS_APM_AI_CONFIG_ENCRYPTION_KEY=$key
"@ | Set-Content -Encoding ascii .env
```

> 키는 한 번 정하면 바꾸지 말 것. 바꾸면 UI에서 저장한 AI 설정(API 키 등)을 복호화하지 못한다.

### 1-4. compose 파일 선택

PoC/고객 반입에는 **`docker-compose.poc.yaml`**을 쓴다(위 `.env`의 `COMPOSE_FILE`이 이미 지정). 특징:

- 컨테이너·네트워크·볼륨·자체 빌드 이미지 이름이 전부 `ds-apm-*` / `ds-apm:poc`로 통일되어 `docker ps` 등에 signoz라는 이름이 노출되지 않는다.
- 개발 장비 전용 마운트(docker.sock 등)가 없어 어느 장비에서든 그대로 기동된다.
- init-clickhouse가 필요 바이너리를 이미 갖고 있으면 다운로드를 건너뛰므로 폐쇄망에서도 기동된다(4-3 참고).

`docker-compose.local.yaml`은 개발 장비 전용이므로 PoC/반입에는 쓰지 않는다.

> 컨테이너 간 통신용 내부 식별자(compose 서비스 키 `signoz`·`clickhouse`, 환경변수 접두사 `SIGNOZ_*`, ClickHouse DB명 `signoz_*`)와 서드파티 이미지 좌표(`signoz/signoz-otel-collector` 등)는 동작에 필요한 값이라 유지된다. 화면·명령 출력에 보이는 이름만 ds-apm이다.

### 1-5. 빌드 + 기동

첫 빌드는 Go 컴파일 + 프론트엔드 yarn build를 포함해 **10~30분** 걸린다. 인터넷이 필요하다(폐쇄망 설치는 4장의 이미지 tar 방식 사용).

```powershell
cd C:\poc\ds-apm\deploy\docker
docker compose up -d --build
```

cmd(명령 프롬프트)라면:

```bat
cd /d C:\poc\ds-apm\deploy\docker
docker compose up -d --build
```

### 1-6. 기동 확인

```powershell
docker compose ps
```

정상 상태의 기대값:

| 컨테이너 | 상태 |
|---|---|
| ds-apm | Up (healthy) |
| ds-apm-otel-collector | Up |
| ds-apm-clickhouse | Up (healthy) |
| ds-apm-zookeeper-1 | Up (healthy) |
| ds-apm-init-clickhouse | Exited (0) — 1회성 초기화 |
| ds-apm-migrator | Exited (0) — 1회성 마이그레이션 |

헬스 체크와 UI 접속:

```powershell
curl.exe -s http://localhost:8080/api/v1/health    # {"status":"ok"} 류 응답
Start-Process http://localhost:8080                # 브라우저 열기
```

첫 접속 시 관리자 계정 생성 화면이 나온다. 이메일/비밀번호를 등록하면 대시보드 진입.

문제가 있으면 로그 확인:

```powershell
docker logs ds-apm --tail 100
docker logs ds-apm-otel-collector --tail 100
```

---

## 2. 고객 앱 서버: OTel 자동 계측

앱 서버에는 **DS-APM을 설치하지 않는다**. 계측 라이브러리를 설치하고 환경변수로 관제 서버를 가리키게만 하면 된다. 아래에서 `<DSAPM_HOST>`는 관제 서버 IP(로컬 PoC면 `localhost`).

### 2-1. FastAPI (Python)

패키지 설치:

```powershell
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install     # 설치된 라이브러리(fastapi, requests 등)에 맞는 계측기 자동 설치
```

실행 — 기존 `uvicorn main:app ...` 명령 앞에 `opentelemetry-instrument`를 붙이는 것이 전부다.

PowerShell:

```powershell
$env:OTEL_SERVICE_NAME = "customer-api"
$env:OTEL_EXPORTER_OTLP_ENDPOINT = "http://<DSAPM_HOST>:4317"
$env:OTEL_EXPORTER_OTLP_PROTOCOL = "grpc"
$env:OTEL_RESOURCE_ATTRIBUTES = "deployment.environment=poc"
opentelemetry-instrument uvicorn main:app --host 0.0.0.0 --port 8000
```

cmd:

```bat
set OTEL_SERVICE_NAME=customer-api
set OTEL_EXPORTER_OTLP_ENDPOINT=http://<DSAPM_HOST>:4317
set OTEL_EXPORTER_OTLP_PROTOCOL=grpc
set OTEL_RESOURCE_ATTRIBUTES=deployment.environment=poc
opentelemetry-instrument uvicorn main:app --host 0.0.0.0 --port 8000
```

Linux(bash):

```bash
export OTEL_SERVICE_NAME=customer-api
export OTEL_EXPORTER_OTLP_ENDPOINT=http://<DSAPM_HOST>:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
opentelemetry-instrument uvicorn main:app --host 0.0.0.0 --port 8000
```

> 코드 수정 0줄. gunicorn을 쓴다면 `opentelemetry-instrument gunicorn ...`으로 동일하게 래핑한다.

### 2-2. Node.js

패키지 설치 (앱 프로젝트 루트에서):

```powershell
npm install @opentelemetry/api @opentelemetry/auto-instrumentations-node
```

실행 — `--require`로 자동 계측 로더를 먼저 올린다.

PowerShell:

```powershell
$env:OTEL_SERVICE_NAME = "customer-node"
$env:OTEL_EXPORTER_OTLP_ENDPOINT = "http://<DSAPM_HOST>:4317"
$env:OTEL_EXPORTER_OTLP_PROTOCOL = "grpc"
node --require @opentelemetry/auto-instrumentations-node/register server.js
```

cmd:

```bat
set OTEL_SERVICE_NAME=customer-node
set OTEL_EXPORTER_OTLP_ENDPOINT=http://<DSAPM_HOST>:4317
set OTEL_EXPORTER_OTLP_PROTOCOL=grpc
node --require @opentelemetry/auto-instrumentations-node/register server.js
```

pm2 등 프로세스 매니저를 쓰면 시작 스크립트를 바꾸는 대신 환경변수로도 가능하다:

```powershell
$env:NODE_OPTIONS = "--require @opentelemetry/auto-instrumentations-node/register"
```

### 2-3. React (브라우저) — 선택

브라우저 계측은 사용자 PC → 관제 서버 :4318(HTTP)로 직접 전송하므로 네트워크 경로와 collector CORS 설정이 필요해 난도가 높다. **PoC 1차 범위에서는 생략하고 백엔드(FastAPI/Node) 트레이스부터 시연하는 것을 권장**한다. 필요 시 `deploy/docker/otel-collector-config.yaml`의 `receivers.otlp.protocols.http`에 `cors.allowed_origins`를 추가하고 OTel Web SDK(`@opentelemetry/sdk-trace-web`)를 붙인다.

### 2-4. 로그 수집 — 선택 (앱 서버에 경량 collector 에이전트)

로그 파일·호스트 메트릭(CPU/메모리)까지 수집하려면 앱 서버(Linux VM)에 경량 otel-collector 에이전트를 하나 세운다. 레포에 원샷 설치기가 있다:

- 설치기: `docs/guides/otel-collector-vm/install.sh`
- 사용법·설정값: `docs/guides/otel-collector-vm/README.md` (환경값 예시는 `dsapm-collector.env.example`)

이 구성에서는 앱의 `OTEL_EXPORTER_OTLP_ENDPOINT`를 `http://localhost:4317`(로컬 에이전트)로 바꾸고, 에이전트가 관제 서버로 중계한다.

---

## 3. 검증

### 3-1. 트래픽 생성

계측된 앱에 요청을 몇 번 보낸다:

```powershell
1..20 | ForEach-Object { curl.exe -s http://localhost:8000/ | Out-Null; Start-Sleep -Milliseconds 300 }
```

### 3-2. UI 확인 체크리스트

`http://<DSAPM_HOST>:8080` 접속 후:

1. **서비스** 메뉴에 `customer-api`, `customer-node`가 나타나는가 (수 분 내)
2. 서비스 클릭 → RPS/지연시간/에러율 차트에 값이 흐르는가
3. **트레이스** 메뉴에서 개별 요청의 스팬 트리가 보이는가 (FastAPI→외부 호출 체인 포함)
4. (로그 에이전트 구성 시) **로그** 메뉴에 로그가 적재되는가

### 3-3. 데이터가 안 보일 때

| 증상 | 확인 |
|---|---|
| 서비스 목록이 비어 있음 | 앱 콘솔에 OTel export 에러가 있는지, `OTEL_EXPORTER_OTLP_ENDPOINT` 오타/방화벽(4317) 확인 |
| collector 수신 여부 | `docker logs ds-apm-otel-collector --tail 50` — export 실패/ClickHouse 연결 오류 확인 |
| 앱→관제서버 통신 | `Test-NetConnection <DSAPM_HOST> -Port 4317` (PowerShell) |
| 스택 자체 이상 | `docker compose ps`에서 Exited 상태 컨테이너 확인 → `docker logs <이름>` |

---

## 4. 폐쇄망 반입 번들 만들기

폐쇄망에서는 빌드가 불가능하다(Docker 빌드가 GitHub/npm/pip에 접근함). **인터넷 되는 장비에서 1~3장을 완주한 뒤**, 그 결과물을 tar로 떠서 반입한다.

### 4-1. 이미지 목록

| 이미지 | 역할 |
|---|---|
| `ds-apm:poc` | DS-APM 백엔드+프론트 (1-5에서 빌드된 것) |
| `signoz/signoz-otel-collector:v0.144.3` | 게이트웨이 collector + 마이그레이터 |
| `clickhouse/clickhouse-server:25.5.6` | 데이터 저장소 |
| `signoz/zookeeper:3.7.1` | ClickHouse 코디네이터 |

(서드파티 이미지 3종의 좌표는 pull·업데이트 경로가 깨지지 않도록 원래 이름을 유지한다.)

(태그는 `deploy/docker/.env`/compose의 `VERSION`, `OTELCOL_TAG` 값과 맞출 것.)

### 4-2. 번들 생성 (인터넷 되는 장비, PowerShell)

```powershell
mkdir C:\poc\bundle
cd C:\poc\bundle

docker save ds-apm:poc | gzip > dsapm-backend.tar.gz
docker save signoz/signoz-otel-collector:v0.144.3 | gzip > dsapm-collector.tar.gz
docker save clickhouse/clickhouse-server:25.5.6 | gzip > dsapm-clickhouse.tar.gz
docker save signoz/zookeeper:3.7.1 | gzip > dsapm-zookeeper.tar.gz

# 배포 파일: deploy 폴더 전체 (compose, collector 설정, clickhouse 설정 포함)
Compress-Archive -Path C:\poc\ds-apm\deploy -DestinationPath dsapm-deploy.zip
```

계측 패키지도 오프라인용으로 준비한다:

```powershell
# Python (고객 서버의 Python 버전·OS에 맞춰야 함 — Linux 서버면 Linux에서 받거나 --platform 지정)
pip download opentelemetry-distro opentelemetry-exporter-otlp -d .\wheelhouse
# 고객 앱의 requirements.txt 기준 계측기까지 포함하려면:
# opentelemetry-bootstrap 목록을 뽑아 함께 pip download

# Node (고객 앱 프로젝트에서 node_modules까지 설치된 상태로 압축하는 방식이 가장 단순)
npm pack @opentelemetry/auto-instrumentations-node @opentelemetry/api
```

### 4-3. 오프라인 기동 요건 — histogramQuantile 바이너리

`init-clickhouse`는 첫 기동 시 GitHub에서 `histogram-quantile` 바이너리를 내려받는다. `docker-compose.poc.yaml`에는 **"이미 있으면 건너뛰기" 로직이 내장**되어 있어, 바이너리가 미리 있으면 폐쇄망에서도 정상 기동된다.

따라서 확인할 것은 하나다: 온라인 장비에서 1-5를 완주하면 바이너리가 `deploy/common/clickhouse/user_scripts/histogramQuantile`에 남는다 → **4-2의 deploy zip에 이 파일이 포함됐는지 반입 전에 확인**한다. 이 파일이 없으면 폐쇄망에서 ClickHouse가 영원히 기동 대기 상태에 빠진다.

### 4-4. 폐쇄망 서버에서 설치 (고객 서버가 Linux인 경우, bash)

```bash
# 반입한 파일 배치
mkdir -p ~/ds-apm && cd ~/ds-apm
unzip dsapm-deploy.zip

# 이미지 적재
for f in dsapm-*.tar.gz; do gunzip -c "$f" | docker load; done
docker images    # 4개 이미지 확인

# .env 작성 (1-3과 동일, bash 버전)
cd deploy/docker
cat > .env <<EOF
COMPOSE_FILE=docker-compose.poc.yaml
DS_APM_AI_CONFIG_ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64)
EOF

# 기동 — 절대 --build 붙이지 말 것 (폐쇄망에선 빌드 불가, 반입 이미지 사용)
docker compose up -d --no-build
docker compose ps
curl -s http://localhost:8080/api/v1/health
```

폐쇄망 서버가 Windows라면 4-4의 적재를 PowerShell로:

```powershell
Get-ChildItem dsapm-*.tar.gz | ForEach-Object { docker load -i $_.FullName }
```

### 4-5. 폐쇄망에서의 AI 기능

- AI 공지 생성기는 기본값이 `local`(외부 호출 없음)이라 별도 설정 없이 알림 파이프라인이 동작한다.
- Code RCA·LLM 기반 공지 고도화는 외부 LLM이 필요하므로 폐쇄망 1차 범위에서 제외한다. 사내 OpenAI 호환 sLLM(vLLM 등)이 있으면 UI의 AI 설정에서 `provider=llm / transport=api` + 엔드포인트 오버라이드로 연동을 시도할 수 있다(별도 검증 필요).

---

## 5. 자주 쓰는 운영 명령 (관제 서버)

```powershell
docker compose ps                      # 상태
docker logs -f ds-apm                  # 백엔드 로그 팔로우
docker restart ds-apm                  # 백엔드만 재시작
docker compose down                    # 중지 (데이터 볼륨은 유지됨)
docker compose up -d                   # 재기동
docker compose down -v                 # ⚠ 데이터까지 전부 삭제 (초기화)
```

데이터는 도커 볼륨 `ds-apm-clickhouse`(텔레메트리), `ds-apm-sqlite`(계정·설정·알림)에 저장되며 `down`/재부팅에도 유지된다.
