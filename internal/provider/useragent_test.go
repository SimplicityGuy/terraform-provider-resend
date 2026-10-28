package provider

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"runtime/debug"
	"testing"
)

func TestResendGoVersionFromDeps(t *testing.T) {
	deps := []*debug.Module{
		{Path: "example.com/other", Version: "v9.9.9"},
		{Path: "github.com/resend/resend-go/v4", Version: "v4.2.0"},
	}
	if got := resendGoVersionFromDeps(deps); got != "4.2.0" {
		t.Errorf("got %q, want 4.2.0", got)
	}

	replaced := []*debug.Module{{
		Path: "github.com/resend/resend-go/v3", Version: "v3.1.1",
		Replace: &debug.Module{Path: "../resend-go", Version: "(devel)"},
	}}
	if got := resendGoVersionFromDeps(replaced); got != unknownVersion {
		t.Errorf("replaced with devel: got %q, want %q", got, unknownVersion)
	}

	if got := resendGoVersionFromDeps(nil); got != unknownVersion {
		t.Errorf("no deps: got %q, want %q", got, unknownVersion)
	}
}

func TestUserAgentFormat(t *testing.T) {
	re := regexp.MustCompile(`^terraform-provider-resend/0\.1\.0 resend-go/(\d+\.\d+\.\d+\S*|unknown)$`)
	for _, in := range []string{"0.1.0", "v0.1.0"} {
		if got := userAgent(in); !re.MatchString(got) {
			t.Errorf("userAgent(%q) = %q, does not match %s", in, got, re)
		}
	}
	if got := userAgent(""); got[:len("terraform-provider-resend/dev ")] != "terraform-provider-resend/dev " {
		t.Errorf("empty version: got %q", got)
	}
}

func TestClientSendsUserAgent(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer srv.Close()

	client := newResendClient("key", "0.1.0")
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")
	_, _ = client.ApiKeys.List()

	if want := userAgent("0.1.0"); got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
}
