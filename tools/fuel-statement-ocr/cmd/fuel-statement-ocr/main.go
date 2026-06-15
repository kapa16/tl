package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tl/fuel-statement-ocr/internal/engine"
)

func main() {
	typeName := flag.String("type", "", "template type: zapravka|prihodnaya|perelivnaya")
	dumpCrops := flag.String("dump-crops", "", "directory to dump cell crops for debugging")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: FuelStatementOCR <image-path> --type <zapravka|prihodnaya|perelivnaya> [--dump-crops dir]")
		os.Exit(2)
	}
	if *typeName == "" {
		fmt.Fprintln(os.Stderr, "missing required --type")
		os.Exit(2)
	}
	res, err := engine.Run(engine.Options{
		ImagePath: flag.Arg(0),
		Type:      *typeName,
		DumpCrops: *dumpCrops,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
