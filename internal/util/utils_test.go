package util_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/util"
	"github.com/stretchr/testify/require"
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
		got := util.GetParentPath(tt.path)
		require.Equal(t, tt.want, got)
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
		got := util.IsDir(test.dir)
		require.Equal(t, test.want, got)
	}
}
