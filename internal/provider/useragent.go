package provider

import (
	"runtime/debug"
	"strings"

	"github.com/resend/resend-go/v3"
)

const (
	resendGoModulePrefix = "github.com/resend/resend-go/"
	unknownVersion       = "unknown"
)

// readBuildInfo is a variable so tests can simulate a binary without embedded
// build info.
var readBuildInfo = debug.ReadBuildInfo

// resendGoVersion returns the resend-go module version this binary was built
// with, read from the embedded build info. It matches any major version of
// the module so it keeps working across SDK upgrades.
func resendGoVersion() string {
	info, ok := readBuildInfo()
	if !ok {
		return unknownVersion
	}
	return resendGoVersionFromDeps(info.Deps)
}

func resendGoVersionFromDeps(deps []*debug.Module) string {
	for _, dep := range deps {
		if !strings.HasPrefix(dep.Path, resendGoModulePrefix) {
			continue
		}
		mod := dep
		if mod.Replace != nil {
			mod = mod.Replace
		}
		if v := strings.TrimPrefix(mod.Version, "v"); v != "" && v != "(devel)" {
			return v
		}
	}
	return unknownVersion
}

// userAgent builds the User-Agent sent to Resend:
// terraform-provider-resend/<provider version> resend-go/<resend-go version>.
func userAgent(providerVersion string) string {
	v := strings.TrimPrefix(providerVersion, "v")
	if v == "" {
		v = "dev"
	}
	return "terraform-provider-resend/" + v + " resend-go/" + resendGoVersion()
}

// newResendClient returns a Resend client whose User-Agent names the provider
// and resend-go versions.
func newResendClient(apiKey, providerVersion string) *resend.Client {
	client := resend.NewClient(apiKey)
	client.UserAgent = userAgent(providerVersion)
	return client
}
