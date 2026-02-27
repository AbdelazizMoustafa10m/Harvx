package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// indicator returns the text indicator for a given status.
func indicator(s Status) string {
	switch s {
	case StatusPass:
		return "[PASS]"
	case StatusWarn:
		return "[WARN]"
	case StatusFail:
		return "[FAIL]"
	default:
		return "[????]"
	}
}

// FormatText writes the doctor report as a human-readable checklist to w.
func FormatText(w io.Writer, report *DoctorReport) {
	fmt.Fprintf(w, "Harvx Doctor — %s\n", report.Directory)
	fmt.Fprintln(w, strings.Repeat("─", 50))
	fmt.Fprintln(w)

	for _, check := range report.Checks {
		fmt.Fprintf(w, "  %s %s\n", indicator(check.Status), check.Name)
		fmt.Fprintf(w, "        %s\n", check.Message)
		for _, detail := range check.Details {
			fmt.Fprintf(w, "        · %s\n", detail)
		}
		fmt.Fprintln(w)
	}

	// Summary line.
	var pass, warn, fail int
	for _, check := range report.Checks {
		switch check.Status {
		case StatusPass:
			pass++
		case StatusWarn:
			warn++
		case StatusFail:
			fail++
		}
	}

	fmt.Fprintln(w, strings.Repeat("─", 50))
	fmt.Fprintf(w, "  %d passed, %d warnings, %d failures\n", pass, warn, fail)
}

// FormatJSON writes the doctor report as indented JSON to w.
func FormatJSON(w io.Writer, report *DoctorReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
