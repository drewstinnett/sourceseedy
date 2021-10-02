package sourceseedy_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/sourceseedy"
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
		got := sourceseedy.GetParentPath(tt.path)
		require.Equal(t, tt.want, got)
	}
}
