package cmd

import "testing"

func TestProjectFromPath(t *testing.T) {
	for _, tt := range []struct {
		name, base, dir, want string
		wantErr               bool
	}{
		{name: "inside", base: "/src", dir: "/src/github.com/a/b", want: "github.com/a/b"},
		{name: "trailing slash", base: "/src", dir: "/src/github.com/a/b/", want: "github.com/a/b"},
		{name: "base itself", base: "/src", dir: "/src", wantErr: true},
		{name: "outside", base: "/src", dir: "/elsewhere/a/b", wantErr: true},
		{name: "sibling with shared prefix", base: "/src", dir: "/src2/a/b", wantErr: true},
		{name: "parent escape", base: "/src", dir: "/src/../etc", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := projectFromPath(tt.base, tt.dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
