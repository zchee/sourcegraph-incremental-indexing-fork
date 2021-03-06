package dbstore

import (
	"context"
	"database/sql"

	"github.com/keegancsmith/sqlf"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// IndexConfiguration stores the index configuration for a repository.
type IndexConfiguration struct {
	ID           int    `json:"id"`
	RepositoryID int    `json:"repository_id"`
	Data         []byte `json:"data"`
}

// scanIndexConfigurations scans a slice of index configurations from the return value of `*Store.query`.
func scanIndexConfigurations(rows *sql.Rows, queryErr error) (_ []IndexConfiguration, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	var indexConfigurations []IndexConfiguration
	for rows.Next() {
		var indexConfiguration IndexConfiguration
		if err := rows.Scan(
			&indexConfiguration.ID,
			&indexConfiguration.RepositoryID,
			&indexConfiguration.Data,
		); err != nil {
			return nil, err
		}

		indexConfigurations = append(indexConfigurations, indexConfiguration)
	}

	return indexConfigurations, nil
}

// scanFirstIndexConfiguration scans a slice of index configurations from the return value of `*Store.query`
// and returns the first.
func scanFirstIndexConfiguration(rows *sql.Rows, err error) (IndexConfiguration, bool, error) {
	indexConfigurations, err := scanIndexConfigurations(rows, err)
	if err != nil || len(indexConfigurations) == 0 {
		return IndexConfiguration{}, false, err
	}
	return indexConfigurations[0], true, nil
}

// GetIndexConfigurationByRepositoryID returns the index configuration for a repository.
func (s *Store) GetIndexConfigurationByRepositoryID(ctx context.Context, repositoryID int) (_ IndexConfiguration, _ bool, err error) {
	ctx, endObservation := s.operations.getIndexConfigurationByRepositoryID.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
	}})
	defer endObservation(1, observation.Args{})

	return scanFirstIndexConfiguration(s.Store.Query(ctx, sqlf.Sprintf(getIndexConfigurationByRepositoryIDQuery, repositoryID)))
}

const getIndexConfigurationByRepositoryIDQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/configuration.go:GetIndexConfigurationByRepositoryID
SELECT
	c.id,
	c.repository_id,
	c.data
FROM lsif_index_configuration c WHERE c.repository_id = %s
`

// UpdateIndexConfigurationByRepositoryID updates the index configuration for a repository.
func (s *Store) UpdateIndexConfigurationByRepositoryID(ctx context.Context, repositoryID int, data []byte) (err error) {
	ctx, endObservation := s.operations.updateIndexConfigurationByRepositoryID.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
	}})
	defer endObservation(1, observation.Args{})

	return s.Store.Exec(ctx, sqlf.Sprintf(updateIndexConfigurationByRepositoryIDQuery, repositoryID, data, data))
}

const updateIndexConfigurationByRepositoryIDQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/configuration.go:UpdateIndexConfigurationByRepositoryID
INSERT INTO lsif_index_configuration (repository_id, data) VALUES (%s, %s)
	ON CONFLICT (repository_id) DO UPDATE SET data = %s
`
