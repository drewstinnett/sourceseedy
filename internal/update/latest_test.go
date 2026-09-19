package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLatest(t *testing.T) {
	for _, tt := range []struct {
		name    string
		handler http.HandlerFunc
		want    string
		wantErr string
	}{
		{
			name: "redirect to the tag",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "https://example.com/o/r/releases/tag/v0.3.0", http.StatusFound)
			},
			want: "v0.3.0",
		},
		{
			name: "relative redirect",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/o/r/releases/tag/v1.2.3", http.StatusMovedPermanently)
			},
			want: "v1.2.3",
		},
		{
			// A repo with no releases goes to the releases page
			name: "redirect to something else",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "https://example.com/o/r/releases", http.StatusFound)
			},
			wantErr: "unexpected redirect",
		},
		{
			name: "no redirect",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: "200 OK",
		},
		{
			name: "not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantErr: "404",
		},
		{
			name: "redirect without a location",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusFound)
			},
			wantErr: "unexpected redirect",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var followed bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/tag/") {
					followed = true
				}
				tt.handler(w, r)
			}))
			defer srv.Close()
			got, err := Latest(context.Background(), srv.Client(), srv.URL+"/o/r/releases")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("Latest = %q, want %q", got, tt.want)
			}
			if followed {
				t.Error("followed the redirect")
			}
		})
	}
}
