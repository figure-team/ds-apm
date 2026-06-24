package alertmanagertypes

import (
	"bytes"
	"net/url"
	"time"

	tmplhtml "html/template"
	tmpltext "text/template"

	"github.com/SigNoz/signoz/pkg/errors"
	alertmanagertemplate "github.com/prometheus/alertmanager/template"
)

// kstZone is a fixed UTC+9 (Korea Standard Time) offset. A fixed zone is used
// instead of time.LoadLocation("Asia/Seoul") so the helper does not depend on
// the tzdata database being present in the (often minimal) runtime container.
var kstZone = time.FixedZone("KST", 9*60*60)

// FormatKST renders an alert timestamp in Korean Standard Time as
// "2006-01-02 15:04 KST". A zero time renders empty so an absent timestamp does
// not print a bogus date. Shared by the template "toKST" func and the notifiers
// that build incident bodies in Go (e.g. MS Teams), so KST formatting has a
// single source of truth.
func FormatKST(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(kstZone).Format("2006-01-02 15:04 KST")
}

func AdditionalFuncMap() tmpltext.FuncMap {
	return tmpltext.FuncMap{
		// urlescape escapes the string for use in a URL query parameter.
		// It returns tmplhtml.HTML to prevent the template engine from escaping the already escaped string.
		// url.QueryEscape escapes spaces as "+", and html/template escapes "+" as "&#43;" if tmplhtml.HTML is not used.
		"urlescape": func(value string) tmplhtml.HTML {
			return tmplhtml.HTML(url.QueryEscape(value))
		},
		// toKST renders an alert timestamp (e.g. {{ .StartsAt | toKST }}) in
		// Korean Standard Time as "2006-01-02 15:04 KST" so operators read the
		// incident time in local time instead of UTC. A zero time renders empty.
		"toKST": FormatKST,
	}
}

// customTemplateOption returns an Option that adds custom functions to the template.
func customTemplateOption() alertmanagertemplate.Option {
	return func(text *tmpltext.Template, html *tmplhtml.Template) {
		text.Funcs(AdditionalFuncMap())
		html.Funcs(AdditionalFuncMap())
	}
}

// FromGlobs overrides the default alertmanager template to add a ruleIdPath template.
// This is used to generate a link to the rule in the alertmanager.
//
// It checks for a ruleId label and generates a path to the rule.
// If testAlert=true label is present, it adds isTestAlert=true query parameter to the URL.
func FromGlobs(paths []string) (*alertmanagertemplate.Template, error) {
	t, err := alertmanagertemplate.FromGlobs(paths, customTemplateOption())
	if err != nil {
		return nil, err
	}

	if err := t.Parse(bytes.NewReader([]byte(`
	{{ define "__ruleIdPath" }}{{- $isTestAlert := "" -}}{{- range .CommonLabels.SortedPairs -}}{{- if eq .Name "testalert" -}}{{- if eq .Value "true" -}}{{- $isTestAlert = "true" -}}{{- end -}}{{- end -}}{{- end -}}{{- range .CommonLabels.SortedPairs -}}{{- if eq .Name "ruleId" -}}{{- if ne .Value "" -}}/edit?ruleId={{ .Value | urlescape }}{{- if $isTestAlert -}}&isTestAlert=true{{- end -}}{{- end -}}{{- end -}}{{- end -}}{{- end }}
	{{ define "__alertmanagerURL" }}{{ .ExternalURL }}/alerts{{ template "__ruleIdPath" . }}{{ end }}
	{{ define "msteamsv2.default.titleLink" }}{{ template "__alertmanagerURL" . }}{{ end }}
	`))); err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "error parsing alertmanager templates")
	}

	return t, nil
}
