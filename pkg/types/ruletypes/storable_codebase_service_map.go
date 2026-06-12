package ruletypes

import (
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

// StorableCodebaseServiceMap is the at-rest form of CodebaseServiceMap. PK is
// (org_id, service_name) for tenant isolation; one service maps to one repo.
type StorableCodebaseServiceMap struct {
	bun.BaseModel `bun:"table:ds_codebase_service_map"`

	OrgID       string `bun:"org_id,pk,notnull,type:text"`
	ServiceName string `bun:"service_name,pk,notnull,type:text"`
	RepoID      string `bun:"repo_id,notnull,type:text"`
	Subpath     string `bun:"subpath,notnull,default:'',type:text"`
}

// FromDomainCodebaseServiceMap validates and converts a mapping to its storable
// form. org_id, service_name, and repo_id are required.
//
func FromDomainCodebaseServiceMap(m CodebaseServiceMap) (*StorableCodebaseServiceMap, error) {
	if strings.TrimSpace(m.OrgID) == "" {
		return nil, fmt.Errorf("storable codebase service map: orgID must not be empty")
	}
	if strings.TrimSpace(m.ServiceName) == "" {
		return nil, fmt.Errorf("storable codebase service map: serviceName must not be empty")
	}
	if strings.TrimSpace(m.RepoID) == "" {
		return nil, fmt.Errorf("storable codebase service map: repoID must not be empty")
	}
	return &StorableCodebaseServiceMap{
		OrgID:       m.OrgID,
		ServiceName: m.ServiceName,
		RepoID:      m.RepoID,
		Subpath:     m.Subpath,
	}, nil
}

// ToDomain converts the storable form back to the domain mapping.
func (s *StorableCodebaseServiceMap) ToDomain() CodebaseServiceMap {
	return CodebaseServiceMap{
		OrgID:       s.OrgID,
		ServiceName: s.ServiceName,
		RepoID:      s.RepoID,
		Subpath:     s.Subpath,
	}
}
