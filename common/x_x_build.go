package common

import (
	_ "embed"
	"runtime/debug"
	"strings"
)

// The embedded version is the release identity of this binary, independent of
// legacy VERSION environment overrides left in a deployment's Compose file.
//
//go:embed x_x_version.txt
var xxVersion string

// XXBuildCommit is set by the custom Docker build; local go builds can use VCS metadata.
var XXBuildCommit string

type XXBuildMetadata struct {
	Version         string `json:"version"`
	UpstreamVersion string `json:"upstream_version"`
	Commit          string `json:"commit,omitempty"`
}

func GetXXBuildMetadata() XXBuildMetadata {
	version := strings.TrimSpace(xxVersion)
	upstream, _, custom := strings.Cut(version, "+lqy007700+v")
	if !custom {
		upstream, _, _ = strings.Cut(version, "-x-x.")
	}
	commit := XXBuildCommit
	if commit == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
					break
				}
			}
		}
	}
	return XXBuildMetadata{Version: version, UpstreamVersion: upstream, Commit: commit}
}
