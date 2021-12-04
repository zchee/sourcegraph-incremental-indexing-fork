package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func isSiteAdmin(ctx context.Context, db dbutil.DB) bool {
	user, err := database.Users(db).GetByCurrentAuthUser(ctx)
	if err != nil {
		if errcode.IsNotFound(err) || err == database.ErrNoCurrentUser {
			return false
		}

		log15.Error("precise-code-intel proxy: failed to get up current user", "error", err)
		return false
	}

	return user != nil && user.SiteAdmin
}

var DefaultValidatorByCodeHost = map[string]func(context.Context, *http.Request, string) (int, error){
	"github.com": enforceAuthViaGitHub,
}

type AuthValidatorMap = map[string]func(context.Context, *http.Request, string) (int, error)

func enforceAuth(ctx context.Context, w http.ResponseWriter, r *http.Request, repoName string, validators AuthValidatorMap) bool {
	for codeHost, validator := range validators {
		if !strings.HasPrefix(repoName, codeHost) {
			continue
		}

		if status, err := validator(ctx, r, repoName); err != nil {
			http.Error(w, err.Error(), status)
			return false
		}

		return true
	}

	http.Error(w, "verification not supported for code host - see https://github.com/sourcegraph/sourcegraph/issues/4967", http.StatusUnprocessableEntity)
	return false
}