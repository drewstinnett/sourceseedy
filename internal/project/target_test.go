package project_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/project"
)

func TestTargetFromRemote(t *testing.T) {
	for _, tt := range []struct {
		remote  string
		want    string
		wantErr bool
	}{
		{remote: "git@github.com:a/b.git", want: "github.com/a/b"},
		{remote: "https://github.com/a/b.git", want: "github.com/a/b"},
		{remote: "ssh://git@gitlab.com:2222/grp/sub/repo.git", want: "gitlab.com/grp/sub/repo"},
		{remote: "github.com/foo/bar", want: "github.com/foo/bar"},
		// Not a remote with a host
		{remote: "/tmp/some/local/repo", wantErr: true},
		{remote: "./relative/repo", wantErr: true},
		{remote: "bad", wantErr: true},
		// Would land outside base
		{remote: "https://evil.com/../../etc/passwd", wantErr: true},
		{remote: "https://evil.com/../x/y", wantErr: true},
		// Wouldn't be found again by list/fzf
		{remote: "https://localhost/a/b.git", wantErr: true},
		{remote: "https://github.com/onlyowner", wantErr: true},
		{remote: "https://github.com/.hidden/repo", wantErr: true},
	} {
		got, err := project.TargetFromRemote(tt.remote)
		if (err != nil) != tt.wantErr {
			t.Errorf("%v: err = %v, wantErr %v", tt.remote, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("%v: got %q, want %q", tt.remote, got, tt.want)
		}
	}
}
