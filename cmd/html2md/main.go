package main

import (
	"fmt"
	"log"
	"os"

	md "github.com/JohannesKaufmann/html-to-markdown"
)

func main() {
	// read input.html file
	htmlBytes, err := os.ReadFile("./input/input.html")
	if err != nil {
		log.Fatal(err)
	}

	// convert html to markdown
	converter := md.NewConverter("", true, nil)

	markdown, err := converter.ConvertString(string(htmlBytes))
	if err != nil {
		panic(err)
	}

	// write output to markdown file
	err = os.WriteFile("./output/output.md", []byte(markdown), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("successfully converted input.html to output.md")
}
