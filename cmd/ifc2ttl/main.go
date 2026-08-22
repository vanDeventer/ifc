// Command ifc2ttl converts an IFC file into RDF using the Linked Building Data
// ontologies, and writes it as Turtle.
//
//	go run ./cmd/ifc2ttl cottage.ifc > cottage.ttl
//
// It reads any IFC4 STEP file, not only the one this repository generates.
// Entities it has no ontology class for become bot:Element and say so in a
// comment, rather than being given a class that may not exist.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vanDeventer/ifc/internal/lbd"
	"github.com/vanDeventer/ifc/internal/step"
)

func main() {
	base := flag.String("base", "https://vandeventer.github.io/ifc/cottage/",
		"namespace the instances are named in")
	out := flag.String("o", "", "write here instead of standard output")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: ifc2ttl [-base ns] [-o out.ttl] model.ifc")
		os.Exit(2)
	}
	in := flag.Arg(0)

	fh, err := os.Open(in)
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()

	model, err := step.Parse(fh)
	if err != nil {
		log.Fatal(err)
	}
	ttl, err := lbd.Convert(model, lbd.Options{Base: *base, Source: filepath.Base(in)})
	if err != nil {
		log.Fatal(err)
	}

	if *out == "" {
		fmt.Print(ttl)
		return
	}
	if err := os.WriteFile(*out, []byte(ttl), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "%s  %d bytes\n", *out, len(ttl))
}
