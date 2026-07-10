package alertmanagertypes

import (
	"testing"

	"github.com/prometheus/alertmanager/template"
)

func TestResolveSOPBoundNotification(t *testing.T) {
	cases := []struct {
		name       string
		status     string
		ann        template.KV
		wantOK     bool
		wantTitle  string
		wantBody   string
		wantNotice string
	}{
		{
			name: "full",
			ann: template.KV{
				IncidentAnnotationNotificationBody: "## 현황\n- 5xx",
				IncidentAnnotationSopTitle:         "Shipping 5xx 대응",
				IncidentAnnotationCustomerUpdate:   "[안내] 점검 중",
			},
			wantOK: true, wantTitle: "Shipping 5xx 대응", wantBody: "## 현황\n- 5xx", wantNotice: "[안내] 점검 중",
		},
		{
			name: "title falls back to ai_headline",
			ann: template.KV{
				IncidentAnnotationNotificationBody: "body",
				IncidentAnnotationAIHeadline:       "헤드라인",
			},
			wantOK: true, wantTitle: "헤드라인", wantBody: "body", wantNotice: "",
		},
		{
			name: "no title candidate keeps ok with empty title",
			ann: template.KV{
				IncidentAnnotationNotificationBody: "body",
			},
			wantOK: true, wantTitle: "", wantBody: "body", wantNotice: "",
		},
		{
			name:   "resolved status prefixes title",
			status: "resolved",
			ann: template.KV{
				IncidentAnnotationNotificationBody: "body",
				IncidentAnnotationSopTitle:         "Shipping 5xx 대응",
			},
			wantOK: true, wantTitle: "✅ 해소 · Shipping 5xx 대응", wantBody: "body", wantNotice: "",
		},
		{
			name:   "resolved status with no title keeps title empty",
			status: "resolved",
			ann: template.KV{
				IncidentAnnotationNotificationBody: "body",
			},
			wantOK: true, wantTitle: "", wantBody: "body", wantNotice: "",
		},
		{
			name:   "no body -> not ok (gate)",
			ann:    template.KV{IncidentAnnotationCustomerUpdate: "[안내]"},
			wantOK: false,
		},
		{
			name: "whitespace body -> not ok",
			ann:  template.KV{IncidentAnnotationNotificationBody: "   \n  "},
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ResolveSOPBoundNotification(tc.status, tc.ann)
			if ok != tc.wantOK {
				t.Fatalf("ok: want %v got %v", tc.wantOK, ok)
			}
			if !tc.wantOK {
				return
			}
			if got.Title != tc.wantTitle || got.Body != tc.wantBody || got.CustomerNotice != tc.wantNotice {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
