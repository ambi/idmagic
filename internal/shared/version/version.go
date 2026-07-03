package version

import (
	"runtime/debug"
)

// ldflags で注入する変数
var (
	Version   = "0.0.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func init() {
	// ldflags で注入されていない場合、runtime/debug.BuildInfo から取得を試みる
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	var revision, timeStr, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			timeStr = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}

	if Version == "0.0.0-dev" && revision != "" {
		v := "0.0.0-dev+" + revision[:8]
		if modified == "true" {
			v += ".dirty"
		}
		Version = v
	}
	if GitCommit == "unknown" && revision != "" {
		GitCommit = revision
	}
	if BuildDate == "unknown" && timeStr != "" {
		BuildDate = timeStr
	}
}

// Info はバージョン情報を表す
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// Get は現在のバージョン情報を返す
func Get() Info {
	goVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		goVersion = info.GoVersion
	}
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: goVersion,
	}
}
