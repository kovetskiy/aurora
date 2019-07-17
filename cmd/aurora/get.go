package main

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
)

func handleGet(opts Options) error {
	client := NewClient(opts.Address)

	if opts.Package != "" {
		return handleGetPackage(client, opts.Package)
	}

	return handleListPackages(client)
}

func handleListPackages(client *Client) error {
	var reply proto.ResponseListPackages
	err := client.Call(
		(*rpc.PackageService).ListPackages,
		proto.RequestListPackages{},
		&reply,
	)
	if err != nil {
		return err
	}

	return printPackages(reply.Packages...)
}

func handleGetPackage(client *Client, name string) error {
	var reply proto.ResponseGetPackage
	err := client.Call(
		(*rpc.PackageService).GetPackage,
		proto.RequestGetPackage{
			Name: name,
		},
		&reply,
	)
	if err != nil {
		return err
	}

	if reply.Package == nil {
		return errors.New("package not found")
	}

	return printPackages(reply.Package)
}

func printPackages(pkgs ...*proto.Package) error {
	tab := tabwriter.NewWriter(os.Stdout, 1, 2, 3, ' ', 0)
	fmt.Fprintf(tab, "NAME\tSTATUS\tVERSION\tDATE\n")

	for _, pkg := range pkgs {
		fmt.Fprintf(
			tab,
			"%s\t%s\t%s\t%s\n",
			pkg.Name,
			pkg.Status,
			pkg.Version,
			pkg.Date.Format(time.RFC3339),
		)
	}

	return tab.Flush()
}
