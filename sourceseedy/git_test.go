package sourceseedy_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/sourceseedy"
	"github.com/stretchr/testify/require"
)

func TestDetectProperPathFromURL(t *testing.T) {
	tests := []struct {
		remote  string
		want    string
		wanterr bool
	}{
		{"git@github.com:drewstinnett/sourceseedy.git", "github.com/drewstinnett/sourceseedy", false},
		{"bad", "", true},
	}
	for _, tt := range tests {
		got, err := sourceseedy.DetectProperPathFromURL(tt.remote)
		require.Equal(t, tt.want, got)
		if tt.wanterr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
