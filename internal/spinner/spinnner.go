package spinner

import (
	"fmt"
	"time"
)

// Spinner represents a loading spinner
type Spinner struct {
	frames   []string
	message  string
	stopChan chan struct{}
	running  bool
	interval time.Duration
}

// New creates a new spinner with default settings
func New() *Spinner {
	return &Spinner{
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 100 * time.Millisecond,
		stopChan: make(chan struct{}),
	}
}

// SetMessage sets the message to display next to the spinner
func (s *Spinner) SetMessage(message string) {
	s.message = message
}

// SetFrames sets custom spinner frames
func (s *Spinner) SetFrames(frames []string) {
	s.frames = frames
}

// SetInterval sets the interval between frame updates
func (s *Spinner) SetInterval(interval time.Duration) {
	s.interval = interval
}

// Start starts the spinner
func (s *Spinner) Start() {
	if s.running {
		return
	}
	s.running = true

	go func() {
		frameIndex := 0
		for {
			select {
			case <-s.stopChan:
				return
			default:
				fmt.Printf("\r%s %s", s.frames[frameIndex], s.message)
				frameIndex = (frameIndex + 1) % len(s.frames)
				time.Sleep(s.interval)
			}
		}
	}()
}

// Stop stops the spinner and clears the line
func (s *Spinner) Stop() {
	if !s.running {
		return
	}
	s.running = false
	s.stopChan <- struct{}{}
	fmt.Print("\r\033[K") // Clear the line
}
