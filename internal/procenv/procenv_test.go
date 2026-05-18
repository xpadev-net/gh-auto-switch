package procenv

import "testing"

func TestWithGHHostReplacesHostAndStripsTokenEnv(t *testing.T) {
	got := WithGHHost([]string{
		"PATH=/bin",
		"GH_HOST=old.example",
		"GH_TOKEN=",
		"GITHUB_TOKEN=secret",
		"KEEP=yes",
	}, "github.com")
	want := []string{"PATH=/bin", "KEEP=yes", "GH_HOST=github.com"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; all=%#v", i, got[i], want[i], got)
		}
	}
}
