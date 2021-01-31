package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/bsquizz/crd-codegen/pkg/generate"
)

func main() {
	in := flag.String("in", "", "path to CRD yaml file")
	to := flag.String("to", "", "directory to output json schema files")
	flag.Parse()

	if *in == "" {
		log.Fatalf("input file must be specified with -in=")
	}

	if *to == "" {
		log.Fatalf("output directory must be specified with -to=")
	}

	if err := os.MkdirAll(*to, 0755); err != nil {
		log.Fatalf("unable to create output directory: %v", err)
	}

	inAbsPath, err := filepath.Abs(*in)
	if err != nil {
		log.Fatalf("invalid input file path: %v", err)
	}

	toAbsPath, err := filepath.Abs(*to)
	if err != nil {
		log.Fatalf("invald output dir path: %v", err)
	}

	err = generate.Generate(inAbsPath, toAbsPath)
	if err != nil {
		log.Fatalf("error generating schemas: %v", err)
	}

	log.Printf("saved to %s", toAbsPath)
}
