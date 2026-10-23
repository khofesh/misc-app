//go:build ocr

package pdf2md

import (
	"strings"
	"testing"
)

func TestOCRScannedPage(t *testing.T) {
	got := convertFixture(t, "scan", Options{OCR: OCRAuto, Tables: true})
	for _, want := range []string{"Quarterly Report", "Highlights", "Berlin office"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}
