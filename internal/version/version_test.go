package version

import "testing"

func TestReleaseVersionFromBuildInfo(t *testing.T) {
	for _, tc := range []struct {
		mainVersion, want string
		ok                bool
	}{
		{"v1.2.3", "1.2.3", true},
		{"v0.8.0", "0.8.0", true},
		{"v10.20.30", "10.20.30", true},
		{"v0.8.1-0.20261112052643-64f72910671c", "", false},
		{"v0.8.0+dirty", "", false},
		{"v0.8.1-0.20261112052643-64f72910671c+dirty", "", false},
		{"v1.2.3-rc.1", "", false},
		{"(devel)", "", false},
		{"", "", false},
		{"v", "", false},
		{"1.2.3", "", false},
		{"v1.2", "", false},
		{"v1.2.3.4", "", false},
		{"v01.2.3", "", false},
		{"vX.Y.Z", "", false},
	} {
		got, ok := releaseVersion(tc.mainVersion)
		if got != tc.want || ok != tc.ok {
			t.Errorf("releaseVersion(%q) = %q, %v; want %q, %v", tc.mainVersion, got, ok, tc.want, tc.ok)
		}
	}
}
