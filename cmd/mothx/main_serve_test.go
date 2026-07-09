package main

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestServeFlagsIncludeExtendedExecutionOptions(t *testing.T) {
	flags := &cliFlags{}
	fs := pflag.NewFlagSet("serve", pflag.ContinueOnError)
	registerServeFlags(fs, flags)

	if err := fs.Parse([]string{"--web-search", "--browser", "--enable-a2a-master", "--unsafe"}); err != nil {
		t.Fatalf("parse serve flags: %v", err)
	}
	opts := flags.serveOptions()
	if !opts.WebSearch {
		t.Fatal("expected web-search serve option")
	}
	if !opts.Browser {
		t.Fatal("expected browser serve option")
	}
	if !opts.A2AMaster {
		t.Fatal("expected A2A master serve option")
	}
	if !opts.Unsafe {
		t.Fatal("expected unsafe serve option")
	}
}
