package sourceseedy_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/sourceseedy"
	"github.com/stretchr/testify/require"
)

func TestProjectFullID(t *testing.T) {
	tests := []struct {
		project sourceseedy.Project
		want    string
	}{
		{
			sourceseedy.Project{Name: "foo", Host: "github.com", Namespace: "mygroup"},
			"github.com/mygroup/foo",
		},
	}
	for _, test := range tests {
		got := test.project.FullID()
		require.Equal(t, test.want, got)
	}
}
