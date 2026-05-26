// Package mockaigenerator returns canned AIStrategy responses keyed by alert
// labels. It exists so demos can show coherent "AI" answers without a real
// LLM. Rules are loaded from JSON files in a directory; the first file
// (lexicographic order) whose match keys all equal the request's labels
// wins. On no match, the mock delegates to a fallback generator so
// non-demo alerts keep working.
package mockaigenerator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Rule maps a set of alert-label matchers to a canned AIStrategy response.
// Keys in Match may use underscores (e.g. "service_name") as an ergonomic
// alias for the dotted label key ("service.name"); at match time an
// underscore key is tried first, then the dotted form as a fallback.
// The reverse does NOT apply: a dotted key in Match never falls back to its
// underscore form in the labels map.
type Rule struct {
	Match    map[string]string    `json:"match"`
	Strategy ruletypes.AIStrategy `json:"strategy"`
}

// Generator holds a sorted list of Rules and a fallback generator. It
// implements ruletypes.AIStrategyGenerator.
type Generator struct {
	rules    []Rule
	fallback ruletypes.AIStrategyGenerator
}

// New loads every *.json file in dir as a Rule and returns a Generator. dir
// may be empty (no rules) — useful for tests that want fallback-only
// behavior. Files are sorted lexicographically; the first match wins.
func New(dir string, fallback ruletypes.AIStrategyGenerator) (*Generator, error) {
	if fallback == nil {
		return nil, fmt.Errorf("mockaigenerator: fallback generator must not be nil")
	}
	g := &Generator{fallback: fallback}
	if dir == "" {
		return g, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return g, nil
		}
		return nil, fmt.Errorf("mockaigenerator: read dir %s: %w", dir, err)
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	for _, name := range files {
		full := filepath.Join(dir, name)
		raw, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("mockaigenerator: read %s: %w", full, err)
		}
		var r Rule
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("mockaigenerator: parse %s: %w", full, err)
		}
		g.rules = append(g.rules, r)
	}
	return g, nil
}

// Generate returns the first matching Rule's strategy, or delegates to the
// fallback generator when no rule matches.
func (g *Generator) Generate(ctx context.Context, req ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error) {
	for _, rule := range g.rules {
		if ruleMatches(rule.Match, req.Labels) {
			return rule.Strategy, nil
		}
	}
	return g.fallback.Generate(ctx, req)
}

// ruleMatches returns true when every key in match equals the corresponding
// label. Two label keys carry dots (e.g. "service.name"); the rule file
// uses underscores for ergonomics ("service_name"), so we normalize at
// match time.
func ruleMatches(match map[string]string, labels map[string]string) bool {
	if len(match) == 0 {
		return false
	}
	for k, v := range match {
		label, ok := labels[k]
		if !ok {
			// try the dotted form (service_name -> service.name)
			label = labels[strings.ReplaceAll(k, "_", ".")]
		}
		if label != v {
			return false
		}
	}
	return true
}
