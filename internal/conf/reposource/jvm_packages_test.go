package reposource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sourcegraph/sourcegraph/internal/api"
)

func TestDecomposeMavenPath(t *testing.T) {
	obtained, _ := ParseMavenModule("maven/org.hamcrest/hamcrest-core")
	assert.Equal(t, obtained.GroupID, "org.hamcrest")
	assert.Equal(t, obtained.ArtifactID, "hamcrest-core")
	assert.Equal(t, api.RepoName("maven/org.hamcrest/hamcrest-core"), obtained.RepoName())
}

func ParseMavenDependencyOrPanic(t *testing.T, value string) *MavenDependency {
	dependency, err := ParseMavenDependency(value)
	if err != nil {
		t.Fatalf("error=%s", err)
	}
	return dependency
}

func TestGreaterThan(t *testing.T) {
	assert.True(t, versionGreaterThan("11.2.0", "1.2.0"))
	assert.True(t, versionGreaterThan("11.2.0", "2.2.0"))
	assert.True(t, versionGreaterThan("11.2.0", "11.2.0-M1"))
	assert.False(t, versionGreaterThan("11.2.0-M11", "11.2.0"))
}

func TestSortDependencies(t *testing.T) {
	dependencies := []*MavenDependency{
		ParseMavenDependencyOrPanic(t, "a:c:1.2.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0.Final"),
		ParseMavenDependencyOrPanic(t, "a:a:1.2.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.11.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0-M11"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0-M1"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0-RC11"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0-RC1"),
		ParseMavenDependencyOrPanic(t, "a:b:1.1.0"),
	}

	SortDependencies(dependencies)

	have := make([]string, 0, len(dependencies))
	for _, dep := range dependencies {
		have = append(have, dep.PackageManagerSyntax())
	}

	want := []string{
		"a:c:1.2.0",
		"a:b:1.11.0",
		"a:b:1.2.0",
		"a:b:1.2.0.Final",
		"a:b:1.2.0-RC11",
		"a:b:1.2.0-RC1",
		"a:b:1.2.0-M11",
		"a:b:1.2.0-M1",
		"a:b:1.1.0",
		"a:a:1.2.0",
	}

	assert.Equal(t, want, have)
}
