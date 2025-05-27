package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/khofesh/misc-app/internal/htmltag"
)

func main() {
	inputFile := "input/input.html"
	outputFile := "output/html2text-result.txt"

	// read html file
	htmlContent, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("error reading HTML file: %v\n", err)
		os.Exit(1)
	}

	// convert html to text
	text, err := htmltag.HtmlToText(string(htmlContent))
	if err != nil {
		fmt.Printf("error converting HTML to text: %v\n", err)
		os.Exit(1)
	}

	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, []byte(text), 0644); err != nil {
		fmt.Printf("error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully converted HTML to text!\n")
	fmt.Printf("Input: %s\n", inputFile)
	fmt.Printf("Output: %s\n", outputFile)
}
