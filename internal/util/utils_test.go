package util_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/util"
)

func TestGetParentPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/src/thing/repo", "/src/thing"},
		{"/src/thing/other-thing/repo", "/src/thing/other-thing"},
		{"relative/src/thing/repo", "relative/src/thing"},
	}
	for _, tt := range tests {
		if got := util.GetParentPath(tt.path); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestIsDir(t *testing.T) {
	tests := []struct {
		dir  string
		want bool
	}{
		{"./testdata/non-exist", false},
		{"./testdata/exists", true},
	}
	for _, test := range tests {
		if got := util.IsDir(test.dir); got != test.want {
			t.Errorf("%v: got %v, want %v", test.dir, got, test.want)
		}
	}
}
