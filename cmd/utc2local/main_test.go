package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "time/tzdata" // don't depend on the system tz database
)

func TestConvertToTimezone(t *testing.T) {
	utc := time.Date(2025, 7, 9, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		timezone string
		want     string
	}{
		{"Asia/Singapore", "2025-07-09 08:00:00 +08"},
		{"Asia/Jakarta", "2025-07-09 07:00:00 WIB"},
		{"Asia/Kuala_Lumpur", "2025-07-09 08:00:00 +08"},
		{"Asia/Tokyo", "2025-07-09 09:00:00 JST"},
	}

	for _, tt := range tests {
		t.Run(tt.timezone, func(t *testing.T) {
			got := convertToTimezone(utc, tt.timezone)

			if !got.Equal(utc) {
				t.Errorf("convertToTimezone() changed the instant: got %v, want %v", got, utc)
			}
			if got.Location().String() != tt.timezone {
				t.Errorf("location = %q, want %q", got.Location(), tt.timezone)
			}
			if s := got.Format("2006-01-02 15:04:05 MST"); s != tt.want {
				t.Errorf("formatted = %q, want %q", s, tt.want)
			}
		})
	}
}

func TestConvertToTimezoneInvalid(t *testing.T) {
	utc := time.Date(2025, 7, 9, 0, 0, 0, 0, time.UTC)

	got := convertToTimezone(utc, "Not/AZone")

	if !got.Equal(utc) || got.Location() != time.UTC {
		t.Errorf("convertToTimezone() = %v, want the input unchanged (%v)", got, utc)
	}
}

// runMain re-runs this test binary as the utc2local program with args.
func runMain(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], append([]string{"-test.run=TestMainProcess", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "UTC2LOCAL_RUN_MAIN=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestMainProcess is not a real test: it runs main() when started by runMain.
func TestMainProcess(t *testing.T) {
	if os.Getenv("UTC2LOCAL_RUN_MAIN") != "1" {
		return
	}
	for i, a := range os.Args {
		if a == "--" {
			os.Args = append([]string{"utc2local"}, os.Args[i+1:]...)
			break
		}
	}
	main()
	os.Exit(0)
}

func TestMainOutput(t *testing.T) {
	out, err := runMain(t, "-utc=2025-07-09T00:00:00Z")
	if err != nil {
		t.Fatalf("main exited with error %v, output:\n%s", err, out)
	}

	for _, want := range []string{
		"UTC time: 2025-07-09 00:00:00 UTC",
		"singapore: 2025-07-09 08:00:00",
		"jakarta: 2025-07-09 07:00:00",
		"kuala lumpur: 2025-07-09 08:00:00",
		"tokyo: 2025-07-09 09:00:00",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}
}

func TestMainNoArgsShowsHelp(t *testing.T) {
	out, err := runMain(t)
	if err != nil {
		t.Fatalf("main exited with error %v, output:\n%s", err, out)
	}
	if !strings.Contains(out, "Usage:") {
		t.Errorf("output does not contain help text:\n%s", out)
	}
}

func TestMainInvalidTime(t *testing.T) {
	out, err := runMain(t, "-utc=not-a-time")

	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("want exit code 1, got err %v, output:\n%s", err, out)
	}
}
