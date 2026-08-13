// Command ifcgen writes the cottage as an IFC4 model and as a web page that
// renders it in 3D.
//
//	go run ./cmd/ifcgen -out .
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/vanDeventer/ifc/internal/cottage"
	"github.com/vanDeventer/ifc/internal/viewer"
)

func main() {
	out := flag.String("out", ".", "directory to write cottage.ifc and cottage.html into")
	date := flag.String("date", "2026-08-12", "creation date stamped into the model (YYYY-MM-DD)")
	flag.Parse()

	stamp, err := time.Parse("2006-01-02", *date)
	if err != nil {
		log.Fatalf("bad -date: %v", err)
	}

	p := cottage.Default()
	model := cottage.Build(p, stamp)

	var buf bytes.Buffer
	if err := model.Write(&buf); err != nil {
		log.Fatal(err)
	}

	ifcPath := filepath.Join(*out, "cottage.ifc")
	if err := os.WriteFile(ifcPath, buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}

	htmlPath := filepath.Join(*out, "cottage.html")
	if err := os.WriteFile(htmlPath, []byte(viewer.Document(buf.String(), p.Name)), 0o644); err != nil {
		log.Fatal(err)
	}

	// The body-only form, for publishing to a host that supplies its own
	// document skeleton.
	fragPath := filepath.Join(*out, "cottage.fragment.html")
	if err := os.WriteFile(fragPath, []byte(viewer.Fragment(buf.String(), p.Name)), 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s  %d bytes\n", ifcPath, buf.Len())
	fmt.Printf("%s\n", htmlPath)
	fmt.Printf("%s\n", fragPath)
}
