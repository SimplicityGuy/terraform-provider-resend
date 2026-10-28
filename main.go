package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/SimplicityGuy/terraform-provider-resend/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

// serve and fatal are variables so tests can run main without starting a
// provider server or exiting the test process.
var (
	serve = providerserver.Serve
	fatal = log.Fatal
)

func main() {
	// ExitOnError matches the flag.CommandLine behavior main always had.
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	if err := run(fs, os.Args[1:]); err != nil {
		fatal(err.Error())
	}
}

// run parses args with fs and serves the provider until the server stops.
func run(fs *flag.FlagSet, args []string) error {
	var debug bool

	fs.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/SimplicityGuy/resend",
		Debug:   debug,
	}

	return serve(context.Background(), provider.New(version), opts)
}
