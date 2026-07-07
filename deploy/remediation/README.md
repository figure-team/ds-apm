# 자동대응 타겟 측 봉쇄 — ForceCommand 래퍼 설치

DS-APM 자동대응 전용 SSH 키를 고정 래퍼로 제한해, 키가 유출돼도 타겟에서
임의 셸을 얻지 못하게 한다. 래퍼는 감사 로그 + ulimit + timeout(SIGKILL)
아래에서 `bash -s`(stdin 프로토콜)만 실행한다.

## 설치 (타겟 1대당 1회)

1. 래퍼 복사 + 권한:

       scp deploy/remediation/dsapm-remed-exec <target>:/usr/local/bin/dsapm-remed-exec
       ssh <target> 'chmod 755 /usr/local/bin/dsapm-remed-exec'

2. 자동대응 계정의 `~/.ssh/authorized_keys`에서 DS-APM 공개키 행을 다음
   형태로 교체 (기존 키 문자열은 그대로):

       restrict,command="/usr/local/bin/dsapm-remed-exec" ssh-ed25519 AAAA... dsapm-remed

   `restrict`가 PTY·포워딩·X11 등을 일괄 차단하고, `command=`가 이 키의
   유일한 실행 대상을 래퍼로 고정한다.

3. 검증 — DS-APM 웹 UI의 자동대응 타겟 "연결 테스트" 실행:
   내부적으로 `echo ok`를 stdin으로 보내므로 래퍼 아래에서도 그대로 성공해야
   한다. 타겟의 `/var/log/dsapm-remed.log`에 호출 행이 남는지 확인.

## 환경 변수 (타겟 측, 선택)

| 변수 | 기본 | 의미 |
|---|---|---|
| `DSAPM_REMED_TIMEOUT` | 305 | 래퍼의 벽시계 상한(초). 서버 org 설정 ExecTimeoutSeconds(기본 300)+5 이상으로 유지 |
| `DSAPM_REMED_LOG` | /var/log/dsapm-remed.log | 감사 로그 경로 |

## 개발 환경 (remtgt-sshd)

remtgt-sshd는 수동 `docker run`(restart=no) 컨테이너다 — 스택 재시작 시
죽으므로 `docker start remtgt-sshd`로 복구. 래퍼 테스트:

    docker cp deploy/remediation/dsapm-remed-exec remtgt-sshd:/usr/local/bin/
    docker exec remtgt-sshd chmod 755 /usr/local/bin/dsapm-remed-exec
    # 컨테이너 내 authorized_keys에 restrict,command= 프리픽스 추가

주의: 래퍼는 타겟에 `timeout`(coreutils/busybox)과 `bash`가 있어야 한다.
없으면 실행이 exit 127로 실패한다(fail-closed — 조용한 우회 없음).
