package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/inconshreveable/log15"
	"golang.org/x/mod/module"
	modzip "golang.org/x/mod/zip"

	dependenciesStore "github.com/sourcegraph/sourcegraph/internal/codeintel/dependencies/store"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gomodproxy"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/sourcegraph/schema"
)

var placeholderGoDependency = func() *reposource.GoDependency {
	dep, err := reposource.ParseGoDependency("sourcegraph.com/placeholder@v0.0.0")
	if err != nil {
		panic(fmt.Sprintf("expected placeholder dependency to parse but got %v", err))
	}
	return dep
}()

type GoModulesSyncer struct {
	// Configuration object describing the connection to the npm registry.
	connection *schema.GoModulesConnection
	depsStore  repos.DependenciesStore
	client     *gomodproxy.Client
}

func NewGoModulesSyncer(
	connection *schema.GoModulesConnection,
	depStore repos.DependenciesStore,
	client *gomodproxy.Client,
) *GoModulesSyncer {
	return &GoModulesSyncer{connection, depStore, client}
}

var _ VCSSyncer = &GoModulesSyncer{}

func (s *GoModulesSyncer) Type() string {
	return "go_modules"
}

// IsCloneable always returns nil for Go dependency repos. We check which versions of a
// modules are cloneable in Fetch, and clone those, ignoring versions that are not
// cloneable.
func (s *GoModulesSyncer) IsCloneable(ctx context.Context, remoteURL *vcs.URL) error {
	return nil
}

func (s *GoModulesSyncer) CloneCommand(ctx context.Context, remoteURL *vcs.URL, bareGitDirectory string) (*exec.Cmd, error) {
	err := os.MkdirAll(bareGitDirectory, 0755)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "git", "--bare", "init")
	if _, err := runCommandInDirectory(ctx, cmd, bareGitDirectory, placeholderGoDependency); err != nil {
		return nil, err
	}

	// The Fetch method is responsible for cleaning up temporary directories.
	if err := s.Fetch(ctx, remoteURL, GitDir(bareGitDirectory)); err != nil {
		return nil, errors.Wrapf(err, "failed to fetch repo for %s", remoteURL)
	}

	// no-op command to satisfy VCSSyncer interface, see docstring for more details.
	return exec.CommandContext(ctx, "git", "--version"), nil
}

// Fetch adds git tags for newly added dependency versions and removes git tags
// for deleted versions.
func (s *GoModulesSyncer) Fetch(ctx context.Context, remoteURL *vcs.URL, dir GitDir) error {
	dep, err := reposource.ParseGoDependencyFromRepoName(remoteURL.Path)
	if err != nil {
		return errors.Wrapf(err, "failed to parse go dependency from repo name: %s", remoteURL.Path)
	}

	dependencies, err := s.moduleVersions(ctx, dep.PackageSyntax())
	if err != nil {
		return err
	}

	cloneable := dependencies[:0] // in place filtering
	for _, dep := range dependencies {
		_, err := s.client.GetVersion(ctx, dep.PackageSyntax(), dep.PackageVersion())
		if err != nil {
			if errcode.IsNotFound(err) {
				log15.Warn("skipping missing go dependency", "dep", dep.PackageManagerSyntax())
				continue
			}
			return err
		}
		cloneable = append(cloneable, dep)
	}

	dependencies = cloneable

	out, err := runCommandInDirectory(ctx, exec.CommandContext(ctx, "git", "tag"), string(dir), placeholderGoDependency)
	if err != nil {
		return err
	}

	tags := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if len(line) == 0 {
			continue
		}
		tags[line] = true
	}

	for i, dependency := range dependencies {
		if tags[dependency.GitTagFromVersion()] {
			continue
		}
		// the gitPushDependencyTag method is responsible for cleaning up temporary directories.
		if err := s.gitPushDependencyTag(ctx, string(dir), dependency, i == 0); err != nil {
			return errors.Wrapf(err, "error pushing dependency %q", dependency.PackageManagerSyntax())
		}
	}

	dependencyTags := make(map[string]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		dependencyTags[dependency.GitTagFromVersion()] = struct{}{}
	}

	for tag := range tags {
		if _, isDependencyTag := dependencyTags[tag]; !isDependencyTag {
			cmd := exec.CommandContext(ctx, "git", "tag", "-d", tag)
			if _, err := runCommandInDirectory(ctx, cmd, string(dir), placeholderGoDependency); err != nil {
				log15.Error("Failed to delete git tag", "error", err, "tag", tag)
				continue
			}
		}
	}

	return nil
}

// RemoteShowCommand returns the command to be executed for showing remote.
func (s *GoModulesSyncer) RemoteShowCommand(ctx context.Context, remoteURL *vcs.URL) (cmd *exec.Cmd, err error) {
	return exec.CommandContext(ctx, "git", "remote", "show", "./"), nil
}

// moduleVersions returns the list of Go module versions for the given module.
// The returned dependencies are sorted in descending semver order (newest first).
func (s *GoModulesSyncer) moduleVersions(ctx context.Context, mod string) (versions []*reposource.GoDependency, err error) {
	for _, d := range s.dependencies() {
		dep, err := reposource.ParseGoDependency(d)
		if err != nil {
			log15.Warn("skipping malformed go dependency", "dep", d, "error", err)
			continue
		}

		if dep.PackageSyntax() == mod {
			versions = append(versions, dep)
		}
	}

	depRepos, err := s.depsStore.ListDependencyRepos(ctx, dependenciesStore.ListDependencyReposOpts{
		Scheme:      dependenciesStore.GoModulesScheme,
		Name:        mod,
		NewestFirst: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list go dependencies from db")
	}

	for _, depRepo := range depRepos {
		dep := reposource.NewGoDependency(module.Version{
			Path:    depRepo.Name,
			Version: depRepo.Version,
		})
		versions = append(versions, dep)
	}

	return versions, nil
}

func (s *GoModulesSyncer) dependencies() []string {
	if s.connection.Dependencies == nil {
		return nil
	}
	return s.connection.Dependencies
}

// gitPushDependencyTag pushes a git tag to the given bareGitDirectory path. The
// tag points to a commit that adds all sources of given dependency. When
// isLatestVersion is true, the HEAD of the bare git directory will also be
// updated to point to the same commit as the git tag.
func (s *GoModulesSyncer) gitPushDependencyTag(ctx context.Context, bareGitDirectory string, dep *reposource.GoDependency, isLatestVersion bool) error {
	tmpDir, err := os.MkdirTemp("", "go-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	zipBytes, err := s.client.GetZip(ctx, dep.PackageSyntax(), dep.PackageVersion())
	if err != nil {
		return err
	}

	err = s.commitZip(ctx, dep, tmpDir, zipBytes)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "git", "remote", "add", "origin", bareGitDirectory)
	if _, err := runCommandInDirectory(ctx, cmd, tmpDir, dep); err != nil {
		return err
	}

	// Use --no-verify for security reasons. See https://github.com/sourcegraph/sourcegraph/pull/23399
	cmd = exec.CommandContext(ctx, "git", "push", "--no-verify", "--force", "origin", "--tags")
	if _, err := runCommandInDirectory(ctx, cmd, tmpDir, dep); err != nil {
		return err
	}

	if isLatestVersion {
		defaultBranch, err := runCommandInDirectory(ctx, exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD"), tmpDir, dep)
		if err != nil {
			return err
		}
		// Use --no-verify for security reasons. See https://github.com/sourcegraph/sourcegraph/pull/23399
		cmd = exec.CommandContext(ctx, "git", "push", "--no-verify", "--force", "origin", strings.TrimSpace(defaultBranch)+":latest", dep.GitTagFromVersion())
		if _, err := runCommandInDirectory(ctx, cmd, tmpDir, dep); err != nil {
			return err
		}
	}

	return nil
}

// commitZip initializes a git repository in the given working directory and creates
// a git commit in that contains all the file contents of the given zip archive.
func (s *GoModulesSyncer) commitZip(ctx context.Context, dep *reposource.GoDependency, workDir string, zipBytes []byte) error {
	zipFilename := path.Join(workDir, "mod.zip")
	err := os.WriteFile(zipFilename, zipBytes, 0666)
	if err != nil {
		return errors.Wrapf(err, "failed to create go module zip file %q on disk", zipFilename)
	}

	modDir := path.Join(workDir, "mod")
	err = modzip.Unzip(modDir, dep.Version, zipFilename)
	if err != nil {
		return errors.Wrapf(err, "failed to unzip go module zip file %q", zipFilename)
	}

	if err = os.RemoveAll(zipFilename); err != nil {
		return errors.Wrapf(err, "failed to remove module zip file %q", zipFilename)
	}

	if err = stripSingleOutermostDirectory(workDir); err != nil {
		return errors.Wrapf(err, "failed to strip single outer-most dir")
	}

	cmd := exec.CommandContext(ctx, "git", "init")
	if _, err := runCommandInDirectory(ctx, cmd, workDir, dep); err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "add", ".")
	if _, err := runCommandInDirectory(ctx, cmd, workDir, dep); err != nil {
		return err
	}

	// Use --no-verify for security reasons. See https://github.com/sourcegraph/sourcegraph/pull/23399
	cmd = exec.CommandContext(ctx, "git", "commit", "--no-verify",
		"-m", dep.PackageManagerSyntax(), "--date", stableGitCommitDate)
	if _, err := runCommandInDirectory(ctx, cmd, workDir, dep); err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "tag",
		"-m", dep.PackageManagerSyntax(), dep.GitTagFromVersion())
	if _, err := runCommandInDirectory(ctx, cmd, workDir, dep); err != nil {
		return err
	}

	return nil
}