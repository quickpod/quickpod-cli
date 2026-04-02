package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const BaseVersion = "v0.1.0"

var (
	Version   = ""
	Commit    = ""
	BuildDate = ""
)

type Info struct {
	Version   string
	Commit    string
	BuildDate string
	Modified  bool
	Source    string
}

func Current() Info {
	info := Info{
		Version:   normalizeVersion(Version),
		Commit:    strings.TrimSpace(Commit),
		BuildDate: strings.TrimSpace(BuildDate),
		Source:    "ldflags",
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = shortCommit(setting.Value)
				}
			case "vcs.time":
				if info.BuildDate == "" {
					info.BuildDate = strings.TrimSpace(setting.Value)
				}
			case "vcs.modified":
				info.Modified = setting.Value == "true"
			}
		}
	}

	if info.Version == "" {
		info.Source = "build"
		info.Version = BaseVersion + "-dev"
		if info.Commit != "" {
			info.Version += "+" + info.Commit
			if info.Modified {
				info.Version += ".dirty"
			}
		} else if info.Modified {
			info.Version += "+dirty"
		}
	}

	if info.Commit != "" {
		info.Commit = shortCommit(info.Commit)
	}

	return info
}

func String() string {
	return Current().Version
}

func Multiline() string {
	info := Current()
	lines := []string{"quickpod-cli " + info.Version}
	if info.Commit != "" {
		lines = append(lines, "commit: "+info.Commit)
	}
	if info.BuildDate != "" {
		lines = append(lines, "built: "+info.BuildDate)
	}
	lines = append(lines, "source: "+info.Source)
	return strings.Join(lines, "\n")
}

func normalizeVersion(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "v") {
		return trimmed
	}
	return "v" + trimmed
}

func shortCommit(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 12 {
		return trimmed
	}
	return trimmed[:12]
}

func GoLDFlags(version string) string {
	return fmt.Sprintf("-X quickpod-cli/internal/version.Version=%s -X quickpod-cli/internal/version.Commit=%s -X quickpod-cli/internal/version.BuildDate=%s", version, shortCommit(Commit), BuildDate)
}
