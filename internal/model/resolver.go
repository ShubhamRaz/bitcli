// Package model defines BitCLI's model catalog, metadata, and persistence contracts.
package model

import (
	"fmt"

	"github.com/bitcli/bitcli/internal/utils"
)

// Resolver translates user input to a concrete artifact.
type Resolver struct {
	catalog Catalog
}

// NewResolver creates a resolver backed by the built-in catalog.
func NewResolver(catalog Catalog) Resolver {
	return Resolver{catalog: catalog}
}

// Resolve returns a known artifact or a helpful not-found error.
func (r Resolver) Resolve(id string) (Artifact, error) {
	if artifact, ok := r.catalog.Resolve(id); ok {
		return artifact, nil
	}
	return Artifact{}, utils.NewError(utils.CodeModelNotFound, fmt.Sprintf("unknown model %q", id))
}

