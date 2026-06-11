package ruletypes

import (
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
// E1 STUB: returns nil → round-trip + validation assertions fail (RED).
func FromDomainCodebaseServiceMap(m CodebaseServiceMap) (*StorableCodebaseServiceMap, error) {
	return nil, nil
}

// ToDomain converts the storable form back to the domain mapping.
//
// E1 STUB: returns the zero value → round-trip assertion fails (RED).
func (s *StorableCodebaseServiceMap) ToDomain() CodebaseServiceMap {
	return CodebaseServiceMap{}
}
