package resolvers

import (
	"context"
	"os"

	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/internal/catalog"
)

func (r *componentResolver) Cyclonedx(ctx context.Context) (*string, error) {
	slocs, err := r.sourceSetResolver(context.TODO())
	if err != nil {
		return nil, err
	}
	return slocs.Cyclonedx(ctx)
}

func (r *rootResolver) GitTreeEntryCyclonedx(ctx context.Context, treeEntry *gql.GitTreeEntryResolver) (*string, error) {
	return sourceSetResolverFromTreeEntry(treeEntry, r.db).Cyclonedx(ctx)
}

func (r *sourceSetResolver) Cyclonedx(ctx context.Context) (*string, error) {
	data, err := os.ReadFile(catalog.CycloneDXSampleFile)
	if err != nil {
		return nil, err
	}
	s := string(data)
	return &s, nil
}