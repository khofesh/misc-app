package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func main() {
	filePath := "input/"
	filename := "raw_data.pdf"
	fullpath := filePath + filename

	// create temporary folder in /temp/pdf_extract420XYZ
	tmpDir, err := os.MkdirTemp("", "pdf_extract")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	imageDir := filepath.Join(tmpDir, "images")
	if err := os.Mkdir(imageDir, 0755); err != nil {
		log.Fatalf("Failed to create images directory: %v", err)
	}

	// extract pdf to images
	err = api.ExtractImagesFile(fullpath, imageDir, nil, nil)
	if err != nil {
		log.Fatalf("Failed to extract images: %v", err)
	}

	files, err := os.ReadDir(imageDir)
	if err != nil {
		log.Fatalf("Failed to read image directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		imgPath := filepath.Join(imageDir, file.Name())
		outFile := filepath.Join(tmpDir, fmt.Sprintf("ocr_%s", file.Name()))

		// call tesseract
		cmd := exec.Command("tesseract", imgPath, outFile)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to run Tesseract on %s: %v", file.Name(), err)
			continue
		}

		textFile := outFile + ".txt"
		text, err := os.ReadFile(textFile)
		if err != nil {
			log.Printf("Failed to read text file for %s: %v", file.Name(), err)
			continue
		}

		fmt.Printf("--- Text from %s ---\n", file.Name())
		fmt.Println(string(text))
	}
}
