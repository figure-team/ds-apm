// Package sopstoretest provides an in-memory fake of ruletypes.SOPStore
// for tests that need a working store without a real database. Behavior
// mirrors sqlsopstore: every method is scoped by orgID, cross-tenant
// reads return ErrSOPDocumentNotFound, and Upsert is idempotent on
// (orgID, sopID, version).
package sopstoretest

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Fake is an in-memory ruletypes.SOPStore. Safe for concurrent use.
type Fake struct {
	mu   sync.RWMutex
	docs map[string]map[string]ruletypes.SOPDocument // orgID -> key(sopID,version) -> doc
}

// New returns an empty Fake.
func New() *Fake {
	return &Fake{docs: map[string]map[string]ruletypes.SOPDocument{}}
}

func key(sopID, version string) string { return sopID + "\x00" + version }

func (f *Fake) Upsert(_ context.Context, orgID string, doc ruletypes.SOPDocument) error {
	if strings.TrimSpace(orgID) == "" {
		return errors.New("sopstoretest: orgID must not be empty")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.docs[orgID] == nil {
		f.docs[orgID] = map[string]ruletypes.SOPDocument{}
	}
	f.docs[orgID][key(doc.SOPID, doc.Version)] = doc
	return nil
}

func (f *Fake) Get(_ context.Context, orgID, sopID, version string) (ruletypes.SOPDocument, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if doc, ok := f.docs[orgID][key(sopID, version)]; ok {
		return doc, nil
	}
	return ruletypes.SOPDocument{}, ruletypes.ErrSOPDocumentNotFound
}

func (f *Fake) GetLatest(_ context.Context, orgID, sopID string) (ruletypes.SOPDocument, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	matches := make([]ruletypes.SOPDocument, 0)
	for _, doc := range f.docs[orgID] {
		if doc.SOPID == sopID {
			matches = append(matches, doc)
		}
	}
	if len(matches) == 0 {
		return ruletypes.SOPDocument{}, ruletypes.ErrSOPDocumentNotFound
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Version > matches[j].Version })
	return matches[0], nil
}

func (f *Fake) List(_ context.Context, orgID string) ([]ruletypes.SOPDocument, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]ruletypes.SOPDocument, 0, len(f.docs[orgID]))
	for _, doc := range f.docs[orgID] {
		out = append(out, doc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SOPID != out[j].SOPID {
			return out[i].SOPID < out[j].SOPID
		}
		return out[i].Version < out[j].Version
	})
	return out, nil
}

func (f *Fake) Delete(_ context.Context, orgID, sopID, version string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.docs[orgID][key(sopID, version)]; !ok {
		return ruletypes.ErrSOPDocumentNotFound
	}
	delete(f.docs[orgID], key(sopID, version))
	return nil
}

func (f *Fake) UpsertRunbook(_ context.Context, orgID, sopID, version string, rb ruletypes.Runbook) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	doc, ok := f.docs[orgID][key(sopID, version)]
	if !ok {
		return ruletypes.ErrSOPDocumentNotFound
	}
	replaced := false
	for i := range doc.Runbooks {
		if doc.Runbooks[i].ID == rb.ID {
			doc.Runbooks[i] = rb
			replaced = true
			break
		}
	}
	if !replaced {
		doc.Runbooks = append(doc.Runbooks, rb)
	}
	f.docs[orgID][key(sopID, version)] = doc
	return nil
}

func (f *Fake) DeleteRunbook(_ context.Context, orgID, sopID, version, runbookID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	doc, ok := f.docs[orgID][key(sopID, version)]
	if !ok {
		return ruletypes.ErrSOPDocumentNotFound
	}
	found := false
	filtered := doc.Runbooks[:0]
	for _, r := range doc.Runbooks {
		if r.ID == runbookID {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return ruletypes.ErrSOPDocumentNotFound
	}
	doc.Runbooks = filtered
	f.docs[orgID][key(sopID, version)] = doc
	return nil
}

var _ ruletypes.SOPStore = (*Fake)(nil)
