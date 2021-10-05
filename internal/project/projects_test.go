package project_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/stretchr/testify/require"
)

func TestProjectFullID(t *testing.T) {
	tests := []struct {
		project project.Project
		want    string
	}{
		{
			project.Project{Name: "foo", Host: "github.com", Namespace: "mygroup"},
			"github.com/mygroup/foo",
		},
	}
	for _, test := range tests {
		got := test.project.FullID()
		require.Equal(t, test.want, got)
	}
}

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
		got, err := project.DetectProperPathFromURL(tt.remote)
		require.Equal(t, tt.want, got)
		if tt.wanterr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
