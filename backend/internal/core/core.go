package core

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	VersionMajor byte = 0
	VersionMinor byte = 1
	VersionPatch byte = 0
)

var (
	Build    = "Custom"
	codename = "HonePress, Write Once, Publish Everywhere."
	intro    = "A static-first publishing platform powered by Go."
)

func init() {
	if Build != "Custom" {
		return
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	var foundRevision bool
	var modified bool
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			if len(setting.Value) < 7 {
				return
			}
			Build = setting.Value[:7]
			foundRevision = true
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if foundRevision && modified {
		Build += "-dirty"
	}
}

func Version() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}

func VersionStatement() []string {
	return []string{
		fmt.Sprintf("HonePress %s (%s) %s (%s %s/%s)", Version(), codename, Build, runtime.Version(), runtime.GOOS, runtime.GOARCH),
		intro,
	}
}
