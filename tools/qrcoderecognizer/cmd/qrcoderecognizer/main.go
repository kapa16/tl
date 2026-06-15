package main

import (
	"fmt"
	"os"

	"tl/qrcoderecognizer/internal/recognize"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: QRCodeRecognizer <image-path>")
		os.Exit(2)
	}

	text, err := recognize.FromFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	_, _ = os.Stdout.WriteString(text)
	if len(text) == 0 || text[len(text)-1] != '\n' {
		_, _ = os.Stdout.WriteString("\n")
	}
}
