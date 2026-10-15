package common

import (
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
	require.True(t, strings.HasPrefix(metadata.Version, metadata.UpstreamVersion+"-x-x."))
	require.Equal(t, XXBuildCommit, metadata.Commit)
	require.NotEqual(t, Version, metadata.Version)
	encoded, err := Marshal(metadata)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"upstream_version"`)
}
