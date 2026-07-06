package remediation

import (
	"fmt"
	"regexp"
)

// llmScriptDenylist는 LLM 생성 fallback 스크립트에서 명백히 파괴적이거나
// 원격 코드 주입성인 구문을 승인 카드에 올리기 전에 차단한다. 패턴 매칭은
// 우회 가능하지만, 뒤의 사람 승인 게이트 앞단에서 "명백한 사고"를 거르는
// 1차 필터다. 사람이 검토·승인한 Runbook 스크립트에는 적용하지 않는다.
// rmForce는 재귀/강제 삭제 플래그(-r/-f 계열)가 붙은 `rm` 호출을 공통 매칭하는
// 프리픽스다. 아래 세 rm 엔트리(루트 / 치명 시스템 트리 / 데이터 디렉토리 최상위)가
// 이 프리픽스에 서로 다른 타겟 경로를 이어붙인다. ⚠️ 한계(의도적 통과): `/var/lib/app`
// 같은 시스템 트리 하위 앱 데이터 경로와 `rm --recursive`(롱플래그)는 잡지 않는다 —
// 앱 데이터 정리는 정당하고, 이 게이트는 심층방어의 얇은 1차 필터일 뿐이다(실질
// 게이트는 사람 승인 + Task 3 강등 + Task 4 능력 제한).
const rmForce = `(?i)\brm\s+(-[a-zA-Z]+\s+)*-[a-zA-Z]*[rf][a-zA-Z]*\s+(--[a-z-]+\s+)*`

var llmScriptDenylist = []struct {
	re   *regexp.Regexp
	desc string
}{
	{regexp.MustCompile(rmForce + `/(\s|$|[*'"&|;])`), "루트 경로 재귀 삭제(rm -rf /)"},
	{regexp.MustCompile(rmForce + `/(etc|bin|sbin|lib|lib64|boot|dev|proc|sys|usr|root)(/|\s|$|[;&|'"*])`), "치명 시스템 트리 재귀 삭제(rm -rf /etc·/usr 등, 하위 경로 포함)"},
	{regexp.MustCompile(rmForce + `/(var|home|opt|srv|run|tmp|mnt|media)/?(\s|$|[;&|'"*])`), "데이터 디렉토리 최상위 재귀 삭제(rm -rf /var·/home 등; 앱 데이터 하위경로는 허용)"},
	{regexp.MustCompile(`--no-preserve-root`), "no-preserve-root"},
	{regexp.MustCompile(`(?i)\bmkfs(\.|\s)`), "파일시스템 포맷(mkfs)"},
	{regexp.MustCompile(`(?i)\bdd\b[^\n]*\bof=/dev/`), "블록 디바이스 덮어쓰기(dd of=/dev/)"},
	{regexp.MustCompile(`:\(\)\s*\{`), "포크밤"},
	{regexp.MustCompile(`(?i)\b(curl|wget)\b[^\n|]*\|\s*(sudo\s+)?(ba|z|da)?sh\b`), "원격 코드 파이프 실행(curl|bash)"},
	{regexp.MustCompile(`(?i)\bbase64\b[^\n|]*\|\s*(sudo\s+)?(ba|z|da)?sh\b`), "인코딩 우회 실행(base64|sh)"},
	{regexp.MustCompile(`(?i)\b(drop|truncate)\s+(table|database)\b`), "파괴적 SQL(DROP/TRUNCATE)"},
	{regexp.MustCompile(`(?i)(^|[;&|]\s*|\s)(shutdown|reboot|halt|poweroff)\b`), "시스템 전원 제어"},
	{regexp.MustCompile(`>\s*/dev/(sd|nvme|vd|xvd)`), "블록 디바이스 리다이렉션"},
	{regexp.MustCompile(`(?i)\bchmod\s+(-[a-zA-Z]+\s+)*777\s+/(\s|$)`), "루트 권한 개방(chmod 777 /)"},
}

// CheckLLMScript는 LLM 생성 스크립트가 denylist에 걸리면 non-nil 에러를
// 반환한다. 호출부는 fail-closed로 처리한다(제안 생성 거부 / 실행 거부).
// 스크립트를 수정하지 않는다 — ScriptSnapshot verbatim 불변식 유지.
func CheckLLMScript(script string) error {
	for _, d := range llmScriptDenylist {
		if d.re.MatchString(script) {
			return fmt.Errorf("llm 스크립트 정적 게이트 차단: %s", d.desc)
		}
	}
	return nil
}
