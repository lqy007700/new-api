package common

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXXBuildMetadataPreservesBinaryIdentity(t *testing.T) {
	previousVersion, previousCommit := Version, XXBuildCommit
	t.Cleanup(func() { Version, XXBuildCommit = previousVersion, previousCommit })
	Version = "stale-deployment-override"
	XXBuildCommit = "0123456789abcdef0123456789abcdef01234567"
	metadata := GetXXBuildMetadata()
	require.NotEmpty(t, metadata.UpstreamVersion)
	require.Equal(t, strings.TrimSpace(xxVersion), metadata.Version)
	require.Regexp(t, "^"+regexp.QuoteMeta(metadata.UpstreamVersion)+`\+lqy007700\+v[1-9][0-9]*$`, metadata.Version)
	require.Equal(t, XXBuildCommit, metadata.Commit)
	require.NotEqual(t, Version, metadata.Version)
	encoded, err := Marshal(metadata)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"upstream_version"`)
}

func TestXXBuildMetadataParsesNewAndLegacyVersions(t *testing.T) {
	previousVersion := xxVersion
	t.Cleanup(func() { xxVersion = previousVersion })
	for _, version := range []string{"v1.0.0-rc.47+lqy007700+v1", "v1.0.0-rc.47+lqy007700+v10", "v1.0.0-rc.47-x-x.3"} {
		xxVersion = version
		metadata := GetXXBuildMetadata()
		require.Equal(t, version, metadata.Version)
		require.Equal(t, "v1.0.0-rc.47", metadata.UpstreamVersion)
	}
}
