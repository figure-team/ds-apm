// 로컬 실행 샌드박스: Alpine 컨테이너(root, systemd 없음) 호환의 최소 권한
// 봉쇄. runbook(사람 승인) 스크립트는 docker.sock 등 루트 권한을 정당하게
// 쓰므로 강등하지 않고 리소스 상한만 best-effort; llm-generated 스크립트는
// setpriv 비루트 강등 + no-new-privs + prlimit 필수(fail-closed).
package remediation

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// cliaudit sandbox 필드에 기록되는 프로파일명.
const (
	sandboxProfileOff        = "off"             // DS_APM_REMED_SANDBOX=off 명시 해제
	sandboxProfileNone       = "none"            // 도구 부재 — runbook만 무샌드박스 허용
	sandboxProfilePrlimit    = "prlimit"         // runbook: 리소스 상한만
	sandboxProfileRestricted = "setpriv+prlimit" // llm: 비루트 강등 + 상한
)

// sandboxUser는 llm 스크립트를 강등 실행할 전용 비루트 계정
// (Dockerfile.local: adduser -D -H). docker 그룹/소켓 접근 없음.
const sandboxUser = "remed"

// toolProbe는 도구/계정/환경 조회를 추상화해 테스트에서 부재 상황을
// 시뮬레이션할 수 있게 한다.
type toolProbe struct {
	lookPath   func(string) (string, error)
	lookupUser func(string) (*user.User, error)
	env        func(string) string
}

func defaultToolProbe() toolProbe {
	return toolProbe{lookPath: exec.LookPath, lookupUser: user.Lookup, env: os.Getenv}
}

// resolveLocalSandbox는 스크립트 소스를 ["bash","-c",script] 앞에 붙일 argv
// 프리픽스와 감사용 프로파일명으로 변환한다. llm 소스는 도구/계정 부재 시
// fail-closed(err), runbook 소스는 best-effort(무샌드박스 폴백 허용).
func resolveLocalSandbox(source string, p toolProbe) ([]string, string, error) {
	if p.env("DS_APM_REMED_SANDBOX") == "off" {
		return nil, sandboxProfileOff, nil
	}
	prlimitPath, prlimitErr := p.lookPath("prlimit")

	if source == ruletypes.RemediationSourceLLMGenerated {
		setprivPath, err := p.lookPath("setpriv")
		if err != nil {
			return nil, "", fmt.Errorf("llm 스크립트 로컬 샌드박스 불가: setpriv 없음 (fail-closed)")
		}
		if prlimitErr != nil {
			return nil, "", fmt.Errorf("llm 스크립트 로컬 샌드박스 불가: prlimit 없음 (fail-closed)")
		}
		if _, err := p.lookupUser(sandboxUser); err != nil {
			return nil, "", fmt.Errorf("llm 스크립트 로컬 샌드박스 불가: %s 계정 없음 (fail-closed)", sandboxUser)
		}
		return []string{
			setprivPath, "--reuid=" + sandboxUser, "--regid=" + sandboxUser,
			"--clear-groups", "--no-new-privs",
			prlimitPath, "--nproc=128", "--nofile=256", "--fsize=268435456", "--cpu=300",
		}, sandboxProfileRestricted, nil
	}

	// runbook: root 유지 특성상 nproc 상한은 상징적이지만 fsize/nofile은 유효.
	if prlimitErr != nil {
		return nil, sandboxProfileNone, nil
	}
	return []string{prlimitPath, "--nproc=256", "--nofile=1024", "--fsize=1073741824"}, sandboxProfilePrlimit, nil
}
