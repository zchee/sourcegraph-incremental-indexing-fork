package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/handlerutil"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
)

func serveRepoRefresh(db database.DB) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()

		repo, err := handlerutil.GetRepo(ctx, db, mux.Vars(r))
		if err != nil {
			return err
		}

		_, err = repoupdater.DefaultClient.EnqueueRepoUpdate(ctx, repo.Name)
		return err
	}
}
