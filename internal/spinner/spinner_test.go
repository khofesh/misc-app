package spinner

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// captureStdout runs fn with os.Stdout redirected and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	orig := os.Stdout
	os.Stdout = w

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()

	os.Stdout = orig
	w.Close()
	return <-done
}

func TestNewDefaults(t *testing.T) {
	s := New()

	if len(s.frames) == 0 {
		t.Error("New() frames is empty")
	}
	if s.interval != 100*time.Millisecond {
		t.Errorf("New() interval = %v, want 100ms", s.interval)
	}
	if s.stopChan == nil {
		t.Error("New() stopChan is nil")
	}
	if s.running {
		t.Error("New() spinner is running before Start")
	}
}

func TestSetters(t *testing.T) {
	s := New()
	frames := []string{"a", "b"}

	s.SetMessage("Loading...")
	s.SetFrames(frames)
	s.SetInterval(5 * time.Millisecond)

	if s.message != "Loading..." {
		t.Errorf("message = %q, want %q", s.message, "Loading...")
	}
	if strings.Join(s.frames, "") != "ab" {
		t.Errorf("frames = %v, want %v", s.frames, frames)
	}
	if s.interval != 5*time.Millisecond {
		t.Errorf("interval = %v, want 5ms", s.interval)
	}
}

func TestStartStop(t *testing.T) {
	s := New()
	s.SetMessage("working")
	s.SetFrames([]string{"X", "Y"})
	s.SetInterval(time.Millisecond)

	out := captureStdout(t, func() {
		s.Start()
		if !s.running {
			t.Error("running = false after Start")
		}
		time.Sleep(20 * time.Millisecond)
		s.Stop()
	})

	if s.running {
		t.Error("running = true after Stop")
	}
	for _, want := range []string{"X working", "Y working", "\r\033[K"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q does not contain %q", out, want)
		}
	}
	if !strings.HasSuffix(out, "\r\033[K") {
		t.Errorf("output %q should end by clearing the line", out)
	}
}

func TestStartTwiceIsNoop(t *testing.T) {
	s := New()
	s.SetInterval(time.Millisecond)

	captureStdout(t, func() {
		s.Start()
		s.Start()
		// with two goroutines running, the second Stop would block forever
		s.Stop()
	})
}

func TestStopWithoutStartIsNoop(t *testing.T) {
	s := New()

	out := captureStdout(t, func() {
		s.Stop()
	})

	if out != "" {
		t.Errorf("Stop() without Start wrote %q, want nothing", out)
	}
}

func TestRestart(t *testing.T) {
	s := New()
	s.SetInterval(time.Millisecond)

	captureStdout(t, func() {
		s.Start()
		s.Stop()
		s.Start()
		if !s.running {
			t.Error("running = false after restart")
		}
		s.Stop()
	})
}
