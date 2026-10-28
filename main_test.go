package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// stubServe replaces serve for the test and records the options it receives.
func stubServe(t *testing.T, err error) *providerserver.ServeOpts {
	t.Helper()
	orig := serve
	t.Cleanup(func() { serve = orig })

	var got providerserver.ServeOpts
	serve = func(_ context.Context, factory func() provider.Provider, opts providerserver.ServeOpts) error {
		if factory() == nil {
			t.Error("expected a non-nil provider from the factory")
		}
		got = opts
		return err
	}
	return &got
}

func TestRun_defaults(t *testing.T) {
	got := stubServe(t, nil)

	if err := run(newFlagSet(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Address != "registry.terraform.io/SimplicityGuy/resend" {
		t.Errorf("unexpected address %q", got.Address)
	}
	if got.Debug {
		t.Error("expected debug to default to false")
	}
}

func TestRun_debugFlag(t *testing.T) {
	got := stubServe(t, nil)

	if err := run(newFlagSet(), []string{"-debug"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Debug {
		t.Error("expected -debug to enable debug mode")
	}
}

func TestRun_serveError(t *testing.T) {
	stubServe(t, errors.New("serve failed"))

	if err := run(newFlagSet(), nil); err == nil || err.Error() != "serve failed" {
		t.Errorf("expected the serve error, got %v", err)
	}
}

func TestMain_serveError(t *testing.T) {
	stubServe(t, errors.New("serve failed"))

	origFatal, origArgs := fatal, os.Args
	t.Cleanup(func() { fatal, os.Args = origFatal, origArgs })
	var msg string
	fatal = func(v ...any) { msg, _ = v[0].(string) }
	os.Args = []string{"terraform-provider-resend"}

	main()

	if msg != "serve failed" {
		t.Errorf("expected fatal to receive the serve error, got %q", msg)
	}
}

func TestMain_ok(t *testing.T) {
	stubServe(t, nil)

	origFatal, origArgs := fatal, os.Args
	t.Cleanup(func() { fatal, os.Args = origFatal, origArgs })
	fatal = func(v ...any) { t.Errorf("unexpected fatal: %v", v) }
	os.Args = []string{"terraform-provider-resend"}

	main()
}

// newFlagSet returns a flag set that reports parse errors instead of exiting.
func newFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func TestRun_badFlag(t *testing.T) {
	stubServe(t, nil)

	if err := run(newFlagSet(), []string{"-nope"}); err == nil {
		t.Error("expected an error for an unknown flag")
	}
}
