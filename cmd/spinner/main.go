package main

import (
	"time"

	"github.com/khofesh/misc-app/internal/spinner"
)

func main() {
	s := spinner.New()
	s.SetMessage("Loading...")
	// s.SetFrames([]string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}) // Moon phases
	s.SetFrames([]string{"-", "\\", "|", "/"}) // Classic spinner
	// Start the spinner
	s.Start()

	// Do some work
	time.Sleep(5 * time.Second)

	// Stop the spinner
	s.Stop()
}
