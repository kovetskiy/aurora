package main

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/kovetskiy/aurora/pkg/signature"
)

func handleGet(opts Options) error {
	client := NewClient(opts.Address)
	signer := NewSigner(opts.Key)

	if opts.Package != "" {
		return handleGetPackage(client, opts.Package, signer.sign())
	}

	return handleListPackages(client, signer.sign())
}

func handleListPackages(client *Client, signature *signature.Signature) error {
	var reply proto.ResponseListPackages
	err := client.Call(
		(*rpc.PackageService).ListPackages,
		proto.RequestListPackages{
			Signature: signature,
		},
		&reply,
	)
	if err != nil {
		return err
	}

	return printPackages(reply.Packages...)
}

func handleGetPackage(client *Client, name string, signature *signature.Signature) error {
	var reply proto.ResponseGetPackage
	err := client.Call(
		(*rpc.PackageService).GetPackage,
		proto.RequestGetPackage{
			Signature: signature,
			Name:      name,
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
	fmt.Fprintf(tab, "NAME\tSTATUS\tVERSION\tDATE\tVER TIME\tBUILD TIME\tPRIORITY\tFAILURES\n")

	for _, pkg := range pkgs {
		fmt.Fprintf(
			tab,
			"%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
			pkg.Name,
			pkg.Status,
			pkg.Version,
			pkg.Date.Format(time.RFC3339),
			pkg.PkgverTime.String(),
			pkg.BuildTime.String(),
			pkg.Priority,
			pkg.Failures,
		)
	}

	return tab.Flush()
}
