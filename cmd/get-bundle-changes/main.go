// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"gopkg.in/juju/charm.v5"

	"github.com/juju/bundlechanges"
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if len(flag.Args()) > 1 {
		fmt.Fprintln(os.Stderr, "need a bundle path as first and only argument")
		os.Exit(2)
	}
	r := os.Stdin
	if path := flag.Arg(0); path != "" {
		var err error
		if r, err = os.Open(path); err != nil {
			fmt.Fprintf(os.Stderr, "invalid bundle path: %s\n", err)
			os.Exit(2)
		}
		defer r.Close()
	}
	if err := process(r, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "unable to parse bundle: %s\n", err)
		os.Exit(1)
	}
}

// usage outputs instructions on how to use this command.
func usage() {
	fmt.Fprintln(os.Stderr, "usage: get-bundle-changes [bundle]")
	fmt.Fprintln(os.Stderr, "bundle can also be provided on stdin")
	flag.PrintDefaults()
	os.Exit(2)
}

// process generates and print to w the set of changes required to deploy
// the bundle data to be retrieved using r.
func process(r io.Reader, w io.Writer) error {
	// Read the bundle data.
	data, err := charm.ReadBundleData(r)
	if err != nil {
		return err
	}
	// Validate the bundle.
	if err := data.Verify(nil); err != nil {
		return err
	}
	// Generate and print the changes.
	changes := bundlechanges.FromData(data)
	content, err := json.MarshalIndent(changes, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(content))
	return nil
}
