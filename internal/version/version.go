package version

import (
	"regexp"
	"runtime/debug"
)

const fallbackVersion = "0.0.0-dev"

// Version is set by a release linker flag. If no linker flag sets it, it
// comes from the build information that Go embeds at build time. The run-time
// environment (working directory, executable location, PATH) has no effect.
var Version string

// releaseTag matches an exact release version, such as v1.2.3. It does not
// match pseudo-versions, pre-releases, "+dirty" builds, or "(devel)".
var releaseTag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func init() {
	if Version != "" {
		return
	}

	Version = fallbackVersion
	if info, ok := debug.ReadBuildInfo(); ok {
		if version, ok := releaseVersion(info.Main.Version); ok {
			Version = version
		}
	}
}

// releaseVersion maps the main module version that `go build` stamps from Git
// to a release version without the "v" prefix. Go stamps a tag only when the
// checkout has no tracked or untracked changes. Go stamps tags v2.0.0 and
// higher only if the module path has a /vN suffix, so those builds report the
// fallback.
func releaseVersion(mainVersion string) (string, bool) {
	if !releaseTag.MatchString(mainVersion) {
		return "", false
	}
	return mainVersion[1:], true
}
