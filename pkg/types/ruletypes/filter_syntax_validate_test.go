package ruletypes

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ruleWithFilterExpr는 metrics 빌더 쿼리에 filter.expression만 바꿔 끼운 규칙 JSON을 만든다.
func ruleWithFilterExpr(expression string) string {
	return fmt.Sprintf(`{
		"alert": "FilterSyntaxTest",
		"version": "v5",
		"condition": {
			"compositeQuery": {
				"queryType": "builder",
				"queries": [{
					"type": "builder_query",
					"spec": {
						"name": "A",
						"signal": "metrics",
						"aggregations": [{"metricName": "signoz_calls_total", "spaceAggregation": "sum"}],
						"stepInterval": "5m",
						"filter": {"expression": %q}
					}
				}]
			},
			"target": 10.0,
			"matchType": "1",
			"op": "1"
		}
	}`, expression)
}

func TestValidate_FilterExprSyntax(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "정상 표현식은 통과",
			expression: "service.name = 'frontend'",
		},
		{
			name:       "빈 표현식은 통과",
			expression: "",
		},
		{
			name:       "닫는 따옴표 없는 표현식은 거부 (T2-001 실사례)",
			expression: "service.name ='frontend",
			wantErr:    true,
		},
		{
			// 키 단독은 문법상 유효(존재/전문검색 조건)라 통과가 맞다.
			name:       "키 단독으로 끝나는 표현식은 문법상 유효",
			expression: "service.name ='frontend' and http.status_code ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rule PostableRule
			if err := json.Unmarshal([]byte(ruleWithFilterExpr(tt.expression)), &rule); err != nil {
				t.Fatalf("unmarshal 실패: %v", err)
			}
			err := rule.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expression %q: 검증 에러를 기대했으나 통과함", tt.expression)
				}
				if !strings.Contains(err.Error(), "syntax errors") {
					t.Fatalf("expression %q: syntax 에러 메시지를 기대했으나: %v", tt.expression, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expression %q: 통과를 기대했으나 에러: %v", tt.expression, err)
			}
		})
	}
}
