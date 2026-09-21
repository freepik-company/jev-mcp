package mcpserver

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersionIsTheSameForEveryInstallPath(t *testing.T) {
	buildInfo := func(main string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Main: debug.Module{Version: main}}, true }
	}
	noBuildInfo := func() (*debug.BuildInfo, bool) { return nil, false }
	cases := []struct {
		name, ldflag string
		info         func() (*debug.BuildInfo, bool)
		want         string
	}{
		{"release archive", "0.3.0", buildInfo("(devel)"), "0.3.0"},
		{"image built from the tag name", "v0.3.0", buildInfo("(devel)"), "0.3.0"},
		{"go install module@version", "dev", buildInfo("v0.3.0"), "0.3.0"},
		{"go install with an empty ldflag", "", buildInfo("v0.3.0"), "0.3.0"},
		{"local go build", "dev", buildInfo("(devel)"), "dev"},
		{"stripped binary without build info", "dev", noBuildInfo, "dev"},
	}
	for _, tc := range cases {
		if got := resolveVersion(tc.ldflag, tc.info); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
