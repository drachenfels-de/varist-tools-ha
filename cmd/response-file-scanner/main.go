package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.intern.drachenfels.de/drachenfels/varist-tools-ha/internal/parser"
)

func main() {
	var verbose bool
	var rating float64

	flag.Float64Var(&rating, "rating", 0, "Minimum rating to include")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Println("Usage: response-file-scanner [--verbose|-v] --rating <min> <path>")
		os.Exit(1)
	}

	filename := flag.Arg(0)
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Cannot open file: %v", err)
	}
	defer file.Close()

	p := parser.NewProcessor()
	err = p.ProcessFile(file, rating, verbose)
	if err == nil && verbose {
		p.PrintCategoryCount()
	}

	if err != nil {
		log.Fatalf("Processing failed: %v", err)
	}
}
